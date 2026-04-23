#!/usr/bin/env bash
set -euo pipefail

binary_path="${ARCHGUARD_BINARY:-}"
if [[ -z "${binary_path}" ]]; then
  echo "ARCHGUARD_BINARY is required." >&2
  exit 1
fi

config_path="${INPUT_CONFIG:-archguard.yaml}"
format="${INPUT_FORMAT:-sarif}"
changed_against="${INPUT_CHANGED_AGAINST:-}"
parse_error_policy="${INPUT_PARSE_ERROR_POLICY:-error}"
severity_threshold="${INPUT_SEVERITY_THRESHOLD:-error}"
output_file="${GITHUB_OUTPUT:-}"
if [[ -z "${output_file}" ]]; then
  output_file="$(mktemp)"
fi

args=(
  check
  --config "${config_path}"
  --format "${format}"
  --parse-error-policy "${parse_error_policy}"
  --severity-threshold "${severity_threshold}"
)
if [[ -n "${changed_against}" ]]; then
  args+=(--changed-against "${changed_against}")
fi

sarif_file=""
set +e
if [[ "${format}" == "sarif" ]]; then
  sarif_file="${RUNNER_TEMP:-$(pwd)}/archguard-results.sarif"
  "${binary_path}" "${args[@]}" > "${sarif_file}"
  exit_code=$?
else
  "${binary_path}" "${args[@]}"
  exit_code=$?
fi
set -e

{
  printf 'exit_code=%s\n' "${exit_code}"
  if [[ -n "${sarif_file}" ]]; then
    printf 'sarif_file=%s\n' "${sarif_file}"
  fi
} >> "${output_file}"
