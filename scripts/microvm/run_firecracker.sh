#!/usr/bin/env bash
set -euo pipefail

# Boot a Firecracker microVM with a rootfs that starts mas-toolrunner.
# This script configures the VM and starts the instance, but does not run any tool call.

FIRECRACKER_BIN=${FIRECRACKER_BIN:-/usr/local/bin/firecracker}
API_SOCK=${API_SOCK:-/tmp/firecracker.sock}
LOG_PATH=${LOG_PATH:-/tmp/firecracker.log}
METRICS_PATH=${METRICS_PATH:-/tmp/firecracker.metrics}
KERNEL_IMAGE=${KERNEL_IMAGE:-}
ROOTFS=${ROOTFS:-}
BOOT_ARGS=${BOOT_ARGS:-"console=ttyS0 reboot=k panic=1 pci=off"}
VCPU_COUNT=${VCPU_COUNT:-1}
MEM_MIB=${MEM_MIB:-512}
VSOCK_CID=${VSOCK_CID:-3}
VSOCK_UDS=${VSOCK_UDS:-/tmp/firecracker.vsock}
TAP_DEVICE=${TAP_DEVICE:-}
GUEST_MAC=${GUEST_MAC:-}
PID_FILE=${PID_FILE:-/tmp/firecracker.pid}
KEEP_ALIVE=${KEEP_ALIVE:-1}

if [ -z "$KERNEL_IMAGE" ] || [ -z "$ROOTFS" ]; then
  echo "KERNEL_IMAGE and ROOTFS are required"
  exit 1
fi

if [ ! -x "$FIRECRACKER_BIN" ]; then
  echo "firecracker not found or not executable: $FIRECRACKER_BIN"
  exit 1
fi

rm -f "$API_SOCK"

# Start Firecracker in the background.
"$FIRECRACKER_BIN" --api-sock "$API_SOCK" --log-path "$LOG_PATH" --metrics-path "$METRICS_PATH" &
FC_PID=$!
echo "$FC_PID" > "$PID_FILE"

cleanup() {
  if ps -p "$FC_PID" >/dev/null 2>&1; then
    kill "$FC_PID" || true
  fi
}

if [ "$KEEP_ALIVE" = "1" ]; then
  trap cleanup EXIT
fi

sleep 0.2

# Configure VM resources.
curl --silent --show-error --unix-socket "$API_SOCK" -X PUT "http://localhost/machine-config" \
  -H "Accept: application/json" -H "Content-Type: application/json" \
  -d "{\"vcpu_count\":$VCPU_COUNT,\"mem_size_mib\":$MEM_MIB}"

# Configure boot source.
curl --silent --show-error --unix-socket "$API_SOCK" -X PUT "http://localhost/boot-source" \
  -H "Accept: application/json" -H "Content-Type: application/json" \
  -d "{\"kernel_image_path\":\"$KERNEL_IMAGE\",\"boot_args\":\"$BOOT_ARGS\"}"

# Configure rootfs.
curl --silent --show-error --unix-socket "$API_SOCK" -X PUT "http://localhost/drives/rootfs" \
  -H "Accept: application/json" -H "Content-Type: application/json" \
  -d "{\"drive_id\":\"rootfs\",\"path_on_host\":\"$ROOTFS\",\"is_root_device\":true,\"is_read_only\":true}"

# Configure vsock device.
curl --silent --show-error --unix-socket "$API_SOCK" -X PUT "http://localhost/vsock" \
  -H "Accept: application/json" -H "Content-Type: application/json" \
  -d "{\"guest_cid\":$VSOCK_CID,\"uds_path\":\"$VSOCK_UDS\"}"

# Configure network if requested.
if [ -n "$TAP_DEVICE" ]; then
  if [ -n "$GUEST_MAC" ]; then
    MAC_FIELD=",\"guest_mac\":\"$GUEST_MAC\""
  else
    MAC_FIELD=""
  fi
  curl --silent --show-error --unix-socket "$API_SOCK" -X PUT "http://localhost/network-interfaces/eth0" \
    -H "Accept: application/json" -H "Content-Type: application/json" \
    -d "{\"iface_id\":\"eth0\",\"host_dev_name\":\"$TAP_DEVICE\"${MAC_FIELD}}"
fi

# Start VM.
curl --silent --show-error --unix-socket "$API_SOCK" -X PUT "http://localhost/actions" \
  -H "Accept: application/json" -H "Content-Type: application/json" \
  -d "{\"action_type\":\"InstanceStart\"}"

echo "firecracker started (pid=$FC_PID, api=$API_SOCK, vsock=$VSOCK_UDS)"

if [ "$KEEP_ALIVE" = "1" ]; then
  wait "$FC_PID"
fi
