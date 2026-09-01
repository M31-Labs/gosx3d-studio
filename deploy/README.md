# Kubernetes deployment

This packaging targets the existing M31 Labs k3s ingress path. It deliberately
does not contain credentials and does not use a mutable container tag.

## Build and publish an immutable image

Build for the public cluster's Linux/amd64 node from the exact source revision
you intend to deploy:

```bash
studio_revision=$(git rev-parse HEAD)
studio_tag="harbor.draco.quest/orchard/gosx3d-studio:${studio_revision}"
docker build --platform linux/amd64 \
  --build-arg "VCS_REF=${studio_revision}" \
  --tag "${studio_tag}" .
docker push "${studio_tag}"
```

Record the `sha256:` manifest digest returned by the push. The deployment input
must use the complete immutable reference:

```text
harbor.draco.quest/orchard/gosx3d-studio@sha256:<64 hexadecimal characters>
```

Do not substitute a tag, including the revision tag above, into the Kubernetes
template.

## Supply runtime secrets outside Git

Before the first deployment, create the `gosx3d-studio-runtime` Secret in the
`m31labs` namespace through the cluster's secret-management path. It must have
exactly these private values:

- `SESSION_SECRET`: at least 32 cryptographically random bytes, used to sign
  browser sessions and CSRF state.
- `STUDIO_ACTION_TOKEN`: at least 32 cryptographically random bytes, used by
  token-authorized Studio API routes.

Never place either value in this repository, a terminal transcript, a shell
history entry, a container image, or the rendered manifest. The tracked
manifest only names the Secret.

The namespace already owns the `regcred` image pull Secret. The manifest
references it without copying registry credentials into source control.

## Render and validate the manifest

Set `GOSX3D_IMAGE` to the immutable Harbor digest and render only that variable:

```bash
export GOSX3D_IMAGE="harbor.draco.quest/orchard/gosx3d-studio@sha256:<digest>"
envsubst '${GOSX3D_IMAGE}' < deploy/kubernetes.yaml \
  | kubectl create --dry-run=client --validate=false -f - -o name
```

The command above is local validation only. It does not create cluster
resources. Review the rendered image reference before applying it.

After explicit deployment approval, the same rendering can be piped to
`kubectl apply -f -`. The workload intentionally stays at one `Recreate`
replica on `ns1007492`, because the public demo shares process-local canonical
scene state. Its root filesystem is read-only; an `emptyDir` mounted at `/tmp`
holds the deliberately ephemeral demo workspace.

## Verify an approved deployment

```bash
kubectl -n m31labs rollout status deployment/gosx3d-studio --timeout=180s
kubectl -n m31labs get pod -l app.kubernetes.io/name=gosx3d-studio -o wide
kubectl -n m31labs get ingress gosx3d-studio
kubectl -n m31labs get certificate gosx3d-studio-tls
curl -fsS https://gosx3d.m31labs.dev/api/health
curl -fsS https://gosx3d.m31labs.dev/api/studio/demo/status
curl -fsSI https://gosx3d.m31labs.dev/
```

The pod must be ready on `ns1007492`; health must report `ok`; demo status must
report a shared ephemeral scene; and the root response must remain dynamic and
`no-store`. Complete the native WebMCP inspect, find, focus, preview, discard,
and human-apply browser flow before treating the deployment as submission-ready.
