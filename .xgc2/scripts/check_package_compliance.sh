#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${repo_root}"

bash -n .xgc2/scripts/*.sh tools/*.sh

nested_git="$(
  find . \
    -path ./.git -prune -o \
    -path ./.ci -prune -o \
    -path ./generated -prune -o \
    -name .git -print
)"
if [[ -n "${nested_git}" ]]; then
  echo "Nested .git directory found." >&2
  echo "${nested_git}" >&2
  exit 1
fi

# A failing git (for example an untrusted bind-mounted checkout) must fail the
# check instead of reporting that nothing is tracked.
tracked_files="$(git ls-files)"
tracked_artifacts="$(grep -E '(^|/)(build|devel|install|\.ci|generated)(/|$)' <<<"${tracked_files}" || true)"
if [[ -n "${tracked_artifacts}" ]]; then
  echo "Generated build artifacts are tracked." >&2
  echo "${tracked_artifacts}" >&2
  exit 1
fi

required_files=(
  README.md
  buf.yaml
  docs/architecture.md
  proto/xgc/v1/message.proto
  proto/xgc/adapter/v1/adapter.proto
  proto/xgc/robot/v1/message.proto
  proto/xgc/semantic/aerial/v1/control.proto
  registry/messages.yaml
  tools/generate_registry.py
  tools/check-breaking.sh
  .github/workflows/ci.yml
  .github/workflows/release.yml
  .xgc2/product.yml
  .xgc2/cmake/xgc2_protobufConfig.cmake
  .xgc2/pkgconfig/xgc2-protobuf.pc
  .xgc2/scripts/build_deb.sh
  .xgc2/scripts/check_package_compliance.sh
  .xgc2/scripts/smoke_test_installed.sh
  .xgc2/scripts/xgc2_artifact_manifest.py
)

for file in "${required_files[@]}"; do
  if [[ ! -f "${file}" ]]; then
    echo "Missing required file: ${file}" >&2
    exit 1
  fi
done

metadata=.xgc2/product.yml
product_version="$(sed -n 's/^version:[[:space:]]*//p' "${metadata}")"
apt_distributions="$(
  sed -n '/^apt:$/,/^[^[:space:]]/s/^  distribution:[[:space:]]*//p' "${metadata}"
)"

if ! grep -q '^id: xgc2-protobuf$' "${metadata}" ||
   ! grep -q '^kind: toolchain-apt$' "${metadata}"; then
  echo "product metadata identity/kind is inconsistent" >&2
  exit 1
fi
if [[ ! "${product_version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+-[0-9]+$ ]]; then
  echo "product metadata version is missing or invalid: ${product_version:-<empty>}" >&2
  exit 1
fi
if [[ -z "${apt_distributions}" ]]; then
  echo "product metadata apt.distribution is missing" >&2
  exit 1
fi

# The installed pkg-config version is the product version without its Debian
# revision; consumers read it as the schema release they build against.
pkgconfig_version="$(sed -n 's/^Version:[[:space:]]*//p' .xgc2/pkgconfig/xgc2-protobuf.pc)"
if [[ "${pkgconfig_version}" != "${product_version%-*}" ]]; then
  echo "pkg-config Version ${pkgconfig_version:-<empty>} must be ${product_version%-*} (product version ${product_version} without its revision)" >&2
  exit 1
fi

IFS=',' read -r -a distributions <<< "${apt_distributions}"
for distribution in "${distributions[@]}"; do
  distribution="${distribution//[[:space:]]/}"
  expected_apt_version="${product_version}~${distribution}"
  if ! grep -Fqx "    ${distribution}: ${expected_apt_version}" "${metadata}"; then
    echo "release.apt_versions.${distribution} must be ${expected_apt_version}" >&2
    exit 1
  fi
done

if [[ "$(find proto -type f -name '*.proto' | wc -l)" -eq 0 ]]; then
  echo "No protobuf schemas found." >&2
  exit 1
fi

if [[ -e profiles ]]; then
  echo "Native Adapter profile paths must be owned by concrete Adapter products." >&2
  exit 1
fi

echo "xgc2-protobuf-dev package compliance checks passed."
