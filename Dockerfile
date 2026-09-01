# syntax=docker/dockerfile:1.7

# Both base images are pinned to multi-platform manifest digests. GoSX, TinyGo,
# and the module graph are pinned independently by scripts/render-build.sh,
# go.mod, and go.sum.
ARG GO_IMAGE="golang:1.26.0-bookworm@sha256:2a0ba12e116687098780d3ce700f9ce3cb340783779646aafbabed748fa6677c"
ARG RUNTIME_IMAGE="debian:bookworm-slim@sha256:88200866dfff7ea7f5cbcb6ec7c8a701889efe6fe859fe64d6990e4b07ea4171"

FROM ${GO_IMAGE} AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download \
    && go mod verify

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    ./scripts/render-build.sh \
    && test -x dist/run.sh \
    && test -x dist/server/app \
    && test -f dist/build.json

FROM ${RUNTIME_IMAGE} AS runtime

ARG VCS_REF="unknown"
LABEL org.opencontainers.image.title="GoSX 3D Studio" \
      org.opencontainers.image.description="Agent-native, revision-safe GoSX 3D scene studio" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.source="https://github.com/M31-Labs/gosx3d-studio" \
      org.opencontainers.image.revision="${VCS_REF}"

WORKDIR /opt/gosx3d-studio

# The packaged server is dynamically linked only to glibc. Copying the CA
# bundle keeps future outbound HTTPS calls safe without installing packages in
# the runtime image.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build --chown=65532:65532 /src/dist/ ./

ENV GOSX_APP_ROOT="/opt/gosx3d-studio" \
    PORT="8080"

EXPOSE 8080
USER 65532:65532
STOPSIGNAL SIGTERM

# Invoke the packaged binary directly. GOSX_APP_ROOT preserves the same
# runtime-root contract as dist/run.sh without requiring a shell in the image.
ENTRYPOINT ["/opt/gosx3d-studio/server/app"]
