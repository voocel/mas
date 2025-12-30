#!/usr/bin/env bash
set -euo pipefail

# Send a calculator request to mas-toolrunner over vsock.
# The VM must already be running and exposing the vsock UDS path.

VSOCK_UDS=${VSOCK_UDS:-/tmp/firecracker.vsock}
CLIENT_BIN=${CLIENT_BIN:-./bin/mas-toolrunner-client}
EXPRESSION=${EXPRESSION:-1+1}

if [ ! -x "$CLIENT_BIN" ]; then
  echo "toolrunner client not found: $CLIENT_BIN"
  exit 1
fi

REQ=$(cat <<EOF
{"tool_call_id":"call-1","tool":{"name":"calculator","args":{"expression":"$EXPRESSION"}},"policy":{"allowed_tools":["calculator"]}}
EOF
)

echo "$REQ" | "$CLIENT_BIN" --vsock "$VSOCK_UDS"
