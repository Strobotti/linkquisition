#!/bin/bash
# Generates Wayland protocol headers for GLFW v3.4.
# Requires: wayland-scanner (from libwayland-dev on Debian/Ubuntu)
#
# These generated files are committed to the vendor tree so that builds
# work without wayland-scanner being installed at build time.

set -euo pipefail

GLFW_DIR="vendor/github.com/go-gl/glfw/v3.4/glfw/glfw"
PROTO_DIR="${GLFW_DIR}/deps/wayland"
OUT_DIR="${GLFW_DIR}/src"

PROTOCOLS=(
    wayland.xml
    viewporter.xml
    xdg-shell.xml
    idle-inhibit-unstable-v1.xml
    pointer-constraints-unstable-v1.xml
    relative-pointer-unstable-v1.xml
    fractional-scale-v1.xml
    xdg-activation-v1.xml
    xdg-decoration-unstable-v1.xml
)

for proto in "${PROTOCOLS[@]}"; do
    base="${proto%.xml}"
    echo "Generating ${base}-client-protocol.h and ${base}-client-protocol-code.h"
    wayland-scanner client-header "${PROTO_DIR}/${proto}" "${OUT_DIR}/${base}-client-protocol.h"
    wayland-scanner private-code "${PROTO_DIR}/${proto}" "${OUT_DIR}/${base}-client-protocol-code.h"
done

echo "Done. Generated $(( ${#PROTOCOLS[@]} * 2 )) files in ${OUT_DIR}/"
