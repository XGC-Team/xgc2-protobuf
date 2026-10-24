#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../.." && pwd)"

product_version() {
  sed -n 's/^version:[[:space:]]*//p' "${repo_root}/.xgc2/product.yml" | head -n 1
}

package_base_version="${PACKAGE_BASE_VERSION:-$(product_version)}"
package_distribution="${PACKAGE_DISTRIBUTION:-}"
if [[ -z "${package_base_version}" ]]; then
  echo "package version is missing; set PACKAGE_BASE_VERSION or .xgc2/product.yml version" >&2
  exit 1
fi

if [[ -z "${package_distribution}" && -r /etc/os-release ]]; then
  # shellcheck disable=SC1091
  . /etc/os-release
  package_distribution="${VERSION_CODENAME:-${UBUNTU_CODENAME:-}}"
fi

case "${package_distribution}" in
  focal|jammy|noble) ;;
  *)
    echo "unsupported PACKAGE_DISTRIBUTION: ${package_distribution:-<empty>}" >&2
    exit 1
    ;;
esac

version="${PACKAGE_VERSION:-${package_base_version}~${package_distribution}}"
if [[ "${ALLOW_UNSCOPED_DEB_VERSION:-0}" != "1" ]]; then
  case "${version}" in
    *"~${package_distribution}"*|*"+${package_distribution}"*) ;;
    *)
      echo "Debian package version '${version}' must include distribution suffix '${package_distribution}'" >&2
      exit 1
      ;;
  esac
fi

package_name="xgc2-protobuf-dev"
architecture="all"
work_dir="${XGC2_PROTOBUF_WORK_DIR:-${repo_root}/.ci}"
generated_dir="${work_dir}/schema-generated"
package_root="${work_dir}/pkg/${package_name}"
output_dir="${XGC2_PROTOBUF_DEB_OUTPUT_DIR:-${repo_root}/debs}"
schema_root="${package_root}/usr/share/xgc2-protobuf"

rm -rf "${generated_dir}" "${work_dir}/pkg" "${output_dir}"
mkdir -p \
  "${generated_dir}/descriptors" \
  "${generated_dir}/registry/cpp" \
  "${generated_dir}/registry/go" \
  "${generated_dir}/registry/python" \
  "${package_root}/DEBIAN" \
  "${schema_root}/descriptors" \
  "${schema_root}/registry" \
  "${package_root}/usr/share/cmake/xgc2_protobuf" \
  "${package_root}/usr/share/pkgconfig" \
  "${package_root}/usr/share/doc/${package_name}" \
  "${output_dir}"

"${script_dir}/install_source.sh" "${generated_dir}" "${package_root}"
cp -a "${repo_root}/README.md" "${package_root}/usr/share/doc/${package_name}/"
cp -a "${repo_root}/docs/." "${package_root}/usr/share/doc/${package_name}/"

cat > "${package_root}/DEBIAN/control" <<EOF
Package: ${package_name}
Version: ${version}
Section: libdevel
Priority: optional
Architecture: ${architecture}
Maintainer: XGC2 <apt@example.com>
Depends: protobuf-compiler
Description: XGC2 language-neutral protobuf schema development files
 Versioned XGC2 proto sources, descriptor set, message registry, and
 CMake/pkg-config discovery metadata. This package deliberately
 does not install generated language runtime bindings.
EOF

find "${package_root}" -type d -exec chmod 0755 {} +
find "${package_root}" -type f -exec chmod 0644 {} +
chmod 0755 "${package_root}/DEBIAN"

test -f "${schema_root}/proto/xgc/v1/message.proto"
test -f "${schema_root}/proto/xgc/adapter/v1/adapter.proto"
test -f "${schema_root}/proto/xgc/robot/v1/message.proto"
test -f "${schema_root}/descriptors/xgc2-protocols.pb"
test -f "${schema_root}/registry/messages.yaml"
test -f "${schema_root}/registry/registry.json"
test ! -e "${schema_root}/profiles"
test -f "${package_root}/usr/share/cmake/xgc2_protobuf/xgc2_protobufConfig.cmake"
test -f "${package_root}/usr/share/pkgconfig/xgc2-protobuf.pc"
test ! -e "${schema_root}/generated"
test ! -e "${schema_root}/.xgc2"

deb_path="${output_dir}/${package_name}_${version}_${architecture}.deb"
fakeroot dpkg-deb --build "${package_root}" "${deb_path}" >/dev/null
dpkg-deb -I "${deb_path}"

if dpkg-deb -c "${deb_path}" | grep -Eq '/generated/(cpp|go|python)/|\.(pb\.(h|cc)|grpc\.pb\.(h|cc)|py)$'; then
  echo "generated language bindings leaked into ${deb_path}" >&2
  exit 1
fi

echo "Debian artifact written to ${deb_path}"
