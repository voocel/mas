#!/usr/bin/env bash
set -euo pipefail

# One-click E2E for Firecracker + mas-toolrunner (build, boot, verify, cleanup).

ROOTFS=${ROOTFS:-./rootfs.ext4}
KERNEL_IMAGE=${KERNEL_IMAGE:-}
BASE_ROOTFS_TAR=${BASE_ROOTFS_TAR:-}
BUILD_TOOLS=${BUILD_TOOLS:-1}
BUILD_ROOTFS=${BUILD_ROOTFS:-0}
WAIT_SEC=${WAIT_SEC:-1}
EXPRESSION=${EXPRESSION:-1+1}
WORK_DIR=${WORK_DIR:-/tmp/mas-fc-$$}
PID_FILE=${PID_FILE:-$WORK_DIR/firecracker.pid}
API_SOCK=${API_SOCK:-$WORK_DIR/firecracker.sock}
LOG_PATH=${LOG_PATH:-$WORK_DIR/firecracker.log}
METRICS_PATH=${METRICS_PATH:-$WORK_DIR/firecracker.metrics}
VSOCK_UDS=${VSOCK_UDS:-$WORK_DIR/firecracker.vsock}
TOOLRUNNER_BIN=${TOOLRUNNER_BIN:-./bin/mas-toolrunner}
CLIENT_BIN=${CLIENT_BIN:-./bin/mas-toolrunner-client}

if [ -z "$KERNEL_IMAGE" ]; then
  echo "KERNEL_IMAGE is required"
  exit 1
fi

mkdir -p "$WORK_DIR"

cleanup() {
  if [ -f "$PID_FILE" ]; then
    kill "$(cat "$PID_FILE")" >/dev/null 2>&1 || true
  fi
  rm -f "$API_SOCK"
}
trap cleanup EXIT

if [ "$BUILD_TOOLS" = "1" ]; then
  go build -o "$TOOLRUNNER_BIN" ./cmd/mas-toolrunner
  go build -o "$CLIENT_BIN" ./cmd/mas-toolrunner-client
fi

if [ "$BUILD_ROOTFS" = "1" ] || [ ! -f "$ROOTFS" ]; then
  if [ -z "$BASE_ROOTFS_TAR" ]; then
    echo "BASE_ROOTFS_TAR is required to build rootfs"
    exit 1
  fi
  BASE_ROOTFS_TAR="$BASE_ROOTFS_TAR" ROOTFS_IMG="$ROOTFS" TOOLRUNNER_BIN="$TOOLRUNNER_BIN" \
    scripts/microvm/build_rootfs.sh
fi

if [ ! -f "$ROOTFS" ]; then
  echo "rootfs not found: $ROOTFS"
  exit 1
fi

KERNEL_IMAGE="$KERNEL_IMAGE" ROOTFS="$ROOTFS" \
API_SOCK="$API_SOCK" LOG_PATH="$LOG_PATH" METRICS_PATH="$METRICS_PATH" \
VSOCK_UDS="$VSOCK_UDS" PID_FILE="$PID_FILE" KEEP_ALIVE=0 \
scripts/microvm/run_firecracker.sh

sleep "$WAIT_SEC"

VSOCK_UDS="$VSOCK_UDS" CLIENT_BIN="$CLIENT_BIN" EXPRESSION="$EXPRESSION" \
scripts/microvm/verify_vsock.sh

echo "microvm e2e ok"
