#!/usr/bin/env bash
set -euo pipefail

output_file="${GITHUB_OUTPUT:-}"
if [[ -z "${output_file}" ]]; then
  output_file="$(mktemp)"
fi

binary_override="${ARCHGUARD_BINARY:-}"
if [[ -n "${binary_override}" ]]; then
  if [[ ! -x "${binary_override}" ]]; then
    echo "ARCHGUARD_BINARY is set but not executable: ${binary_override}" >&2
    exit 1
  fi
  {
    printf 'binary=%s\n' "${binary_override}"
    printf 'version=override\n'
  } >> "${output_file}"
  exit 0
fi

version="${INPUT_VERSION:-latest}"
if [[ "${version}" == "latest" ]]; then
  api_url="${GITHUB_API_URL:-https://api.github.com}/repos/honzikec/archguard/releases/latest"
  version="$(curl -fsSL -H "Accept: application/vnd.github+json" "${api_url}" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  if [[ -z "${version}" ]]; then
    echo "Failed to resolve the latest ArchGuard release tag." >&2
    exit 1
  fi
fi

normalized_version="${version#v}"
uname_s="$(uname -s)"
case "${uname_s}" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  MINGW*|MSYS*|CYGWIN*) os="windows" ;;
  *)
    echo "Unsupported platform: ${uname_s}" >&2
    exit 1
    ;;
esac

uname_m="$(uname -m)"
case "${uname_m}" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "Unsupported architecture: ${uname_m}" >&2
    exit 1
    ;;
esac

extension="tar.gz"
binary_name="archguard"
if [[ "${os}" == "windows" ]]; then
  extension="zip"
  binary_name="archguard.exe"
fi

archive="archguard_${normalized_version}_${os}_${arch}.${extension}"
download_url="${GITHUB_SERVER_URL:-https://github.com}/honzikec/archguard/releases/download/${version}/${archive}"
tmp_dir="$(mktemp -d)"
download_path="${tmp_dir}/${archive}"

curl -fsSL -o "${download_path}" "${download_url}"
tar -xf "${download_path}" -C "${tmp_dir}"
binary_path="${tmp_dir}/${binary_name}"
chmod +x "${binary_path}"

{
  printf 'binary=%s\n' "${binary_path}"
  printf 'version=%s\n' "${version}"
} >> "${output_file}"
