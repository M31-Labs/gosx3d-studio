#!/bin/sh

set -eu

# GoSX production packaging compiles its browser runtime with TinyGo. Render's
# native Go image does not bundle TinyGo, so install the exact compiler release
# used by the Studio when it is not already available on PATH.
studio_tinygo_version="0.41.1"
studio_tinygo_report=""
if command -v tinygo >/dev/null 2>&1; then
	studio_tinygo_report=$(tinygo version 2>/dev/null || true)
fi

case "$studio_tinygo_report" in
	"tinygo version ${studio_tinygo_version} "*)
		printf '%s\n' "Using ${studio_tinygo_report}"
		;;
	*)
		case "$(uname -m)" in
			x86_64 | amd64)
				studio_tinygo_arch="amd64"
				studio_tinygo_sha256="e156d1d93a376eef639a4143d13be07e8c463fb6cf2d7d447698ed4474d23e91"
				;;
			aarch64 | arm64)
				studio_tinygo_arch="arm64"
				studio_tinygo_sha256="789733bc3b5bace0bd1835a267b3ea267804a7ef1cfe69bc522c295f5226d624"
				;;
			*)
				printf '%s\n' "Unsupported TinyGo build architecture: $(uname -m)" >&2
				exit 1
				;;
		esac

		studio_build_tmp=$(mktemp -d "${TMPDIR:-/tmp}/gosx3d-render-build.XXXXXX")
		readonly studio_build_tmp
		trap 'rm -r -- "$studio_build_tmp"' EXIT
		studio_tinygo_archive="${studio_build_tmp}/tinygo.tar.gz"
		studio_tinygo_url="https://github.com/tinygo-org/tinygo/releases/download/v${studio_tinygo_version}/tinygo${studio_tinygo_version}.linux-${studio_tinygo_arch}.tar.gz"

		printf '%s\n' "Downloading TinyGo ${studio_tinygo_version} for linux/${studio_tinygo_arch}"
		curl --fail --location --retry 3 --retry-delay 2 --silent --show-error \
			--output "$studio_tinygo_archive" "$studio_tinygo_url"
		printf '%s  %s\n' "$studio_tinygo_sha256" "$studio_tinygo_archive" | sha256sum --check --status
		tar -xzf "$studio_tinygo_archive" -C "$studio_build_tmp"
		if [ ! -x "${studio_build_tmp}/tinygo/bin/tinygo" ]; then
			printf '%s\n' "TinyGo archive did not contain the expected executable" >&2
			exit 1
		fi
		PATH="${studio_build_tmp}/tinygo/bin:${PATH}"
		export PATH
		studio_tinygo_report=$(tinygo version)
		case "$studio_tinygo_report" in
			"tinygo version ${studio_tinygo_version} "*) ;;
			*)
				printf '%s\n' "Unexpected TinyGo version: ${studio_tinygo_report}" >&2
				exit 1
				;;
		esac
		printf '%s\n' "Using ${studio_tinygo_report}"
		;;
esac

go run m31labs.dev/gosx/cmd/gosx@v0.54.0 build --prod .
