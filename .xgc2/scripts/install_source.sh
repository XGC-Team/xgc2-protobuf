#!/usr/bin/env bash
# Generate/install schemas and discovery metadata. No APT or package publication.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
work="${1:?usage: install_source.sh BUILD_DIR INSTALL_ROOT}"
stage="${2:?install root}"
schema="$stage/usr/share/xgc2-protobuf"
mkdir -p "$work/descriptors" "$work/registry/"{cpp,go,python} \
  "$schema/descriptors" "$schema/registry" "$stage/usr/share/cmake/xgc2_protobuf" "$stage/usr/share/pkgconfig"
mapfile -t protos < <(cd "$root/proto" && find xgc -name '*.proto' -type f | sort)
[[ ${#protos[@]} -gt 0 ]]
(cd "$root/proto" && protoc -I . --include_imports --descriptor_set_out="$work/descriptors/xgc2-protocols.pb" "${protos[@]}")
python3 "$root/tools/generate_registry.py" --registry "$root/registry/messages.yaml" \
  --descriptors "$work/descriptors/xgc2-protocols.pb" \
  --cpp-out "$work/registry/cpp" --go-out "$work/registry/go" --python-out "$work/registry/python" \
  --metadata-out "$work/registry/registry.json"
cp -a "$root/proto" "$schema/proto"
find "$schema/proto" -type d -empty -delete
cp "$work/descriptors/xgc2-protocols.pb" "$schema/descriptors/"
cp "$root/registry/messages.yaml" "$work/registry/registry.json" "$schema/registry/"
cp "$root/.xgc2/cmake/xgc2_protobufConfig.cmake" "$stage/usr/share/cmake/xgc2_protobuf/"
cp "$root/.xgc2/pkgconfig/xgc2-protobuf.pc" "$stage/usr/share/pkgconfig/"
