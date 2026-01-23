#!/usr/bin/env bash
set -e

BASE_ROOTFS_TAR=${BASE_ROOTFS_TAR:-}
ROOTFS_IMG=${ROOTFS_IMG:-./rootfs.ext4}
ROOTFS_SIZE_MB=${ROOTFS_SIZE_MB:-512}
MOUNT_DIR=${MOUNT_DIR:-/tmp/mas-rootfs}
TOOLRUNNER_BIN=${TOOLRUNNER_BIN:-./bin/mas-toolrunner}
SERVICE_FILE=${SERVICE_FILE:-./scripts/microvm/microvm_toolrunner.service}

if [ -z "$BASE_ROOTFS_TAR" ]; then
  echo "BASE_ROOTFS_TAR is required (e.g. a minimal distro rootfs tar)"
  exit 1
fi

if [ ! -f "$TOOLRUNNER_BIN" ]; then
  echo "mas-toolrunner binary not found at $TOOLRUNNER_BIN"
  exit 1
fi

if [ ! -f "$SERVICE_FILE" ]; then
  echo "service file not found at $SERVICE_FILE"
  exit 1
fi

mkdir -p "$(dirname "$ROOTFS_IMG")"
rm -f "$ROOTFS_IMG"

dd if=/dev/zero of="$ROOTFS_IMG" bs=1M count="$ROOTFS_SIZE_MB"
mkfs.ext4 -F "$ROOTFS_IMG"

mkdir -p "$MOUNT_DIR"
sudo mount -o loop "$ROOTFS_IMG" "$MOUNT_DIR"

sudo tar -xpf "$BASE_ROOTFS_TAR" -C "$MOUNT_DIR"

sudo install -d "$MOUNT_DIR/usr/local/bin"
sudo install -m 0755 "$TOOLRUNNER_BIN" "$MOUNT_DIR/usr/local/bin/mas-toolrunner"

sudo install -d "$MOUNT_DIR/etc/systemd/system"
sudo install -m 0644 "$SERVICE_FILE" "$MOUNT_DIR/etc/systemd/system/mas-toolrunner.service"

sudo chroot "$MOUNT_DIR" systemctl enable mas-toolrunner

sync
sudo umount "$MOUNT_DIR"

echo "rootfs created: $ROOTFS_IMG"
