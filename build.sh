#!/usr/bin/env bash
set -euo pipefail

OUTPUT_DIR="dist"
BINARY="diffcsv"

PLATFORMS=(
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "linux/amd64"
)

rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"

for platform in "${PLATFORMS[@]}"; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    dir="${OUTPUT_DIR}/${BINARY}_${GOOS}_${GOARCH}"
    mkdir -p "${dir}"

    output_name="${BINARY}"
    if [ "${GOOS}" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo "Building ${GOOS}/${GOARCH}..."
    GOOS="${GOOS}" GOARCH="${GOARCH}" go build \
        -ldflags="-s -w" \
        -o "${dir}/${output_name}" \
        .

    if [ "${GOOS}" = "windows" ]; then
        (cd "${dir}" && zip -q "../${BINARY}_${GOOS}_${GOARCH}.zip" "${output_name}")
    else
        tar -czf "${dir}.tar.gz" -C "${dir}" "${output_name}"
    fi
    rm -rf "${dir}"
done

echo ""
echo "Builds:"
ls -lh "${OUTPUT_DIR}/"
