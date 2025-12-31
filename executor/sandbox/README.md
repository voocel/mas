# Sandbox Usage & Architecture

---

## 1. What this is

- MAS is a multi-agent library.
- Sandbox is the tool execution/governance layer. It isolates tool calls and enforces policy.
- `mas-sandboxd` is the control-plane service that receives tool calls, applies policy, and schedules runtimes.

**Key concepts**:
- **Agent/Runner** handles reasoning and orchestration.
- **Sandbox** handles tool execution and governance.

---

## 2. Processes & responsibilities (minimal)

```
Agent/Runner → ToolExecutor
  ├─ LocalExecutor → mas-sandboxd (stdin/stdout)
  └─ SandboxExecutor → mas-sandboxd (HTTP)

mas-sandboxd → runtime/local | runtime/microvm
runtime/microvm → Firecracker VM → mas-toolrunner (guest)
```

- **mas-sandboxd**: control-plane (host process)
- **mas-toolrunner**: tool execution service inside the VM
- **mas-toolrunner-client**: host vsock client to the VM

---

## 3. Local development (simplest)

**Single recommended example**: `examples/sandbox_agent`

### Option A: local process (stdin/stdout)

```
go run ./examples/sandbox_agent
```

### Option B: HTTP control plane (recommended structure)

Start control plane:
```
mas-sandboxd -listen :8080 -runtime local
```

SDK side:
```
MAS_SANDBOX_MODE=http \
MAS_SANDBOX_URL=http://127.0.0.1:8080 \
go run ./examples/sandbox_agent
```

---

## 4. microVM (real isolation)

### 4.1 Run control plane (microVM)

```
mas-sandboxd -listen :8080 \
  -runtime microvm \
  -runtime-config ./scripts/microvm/microvm_config.example.json \
  -auth-token mytoken
```

### 4.2 SDK call (unchanged)

```
MAS_SANDBOX_MODE=http \
MAS_SANDBOX_URL=http://127.0.0.1:8080 \
MAS_SANDBOX_TOKEN=mytoken \
go run ./examples/sandbox_agent
```

---

## 5. mac development notes

- Firecracker does not run natively on macOS.
- Use `local` runtime for dev:

```
mas-sandboxd -listen :8080 -runtime local
```

If you need production-like behavior, run a Linux VM on mac and run microVM runtime inside it.

---

## 6. microVM prerequisites (minimal)

You need:
- Firecracker (host)
- Linux kernel image
- rootfs (ext4)
- rootfs that embeds and auto-starts `mas-toolrunner`

Scripts:
- `scripts/microvm/build_rootfs.sh`
- `scripts/microvm/run_firecracker.sh`
- `scripts/microvm/verify_vsock.sh`
- `scripts/microvm/microvm_e2e.sh`

---

## 7. microVM config notes

Example config: `scripts/microvm/microvm_config.example.json`

Key fields:
- `firecracker_bin` / `kernel_image` / `rootfs`
- `vsock.cid / port / uds_path`
- `tool_runner.command` (host-side caller)
- `network.tap_device` (required when network enabled)
- `network.allowed_cidrs` (optional: host-enforced egress allowlist)
- `drives` (optional: host disk image allowlist)
- `cgroup.path / cpu_quota_us / memory_max_bytes` (optional: hard limits)
- `pool.size` (when > 1, `{id}` placeholders are required)

**Notes:**
- `tool_runner.command` is the host `mas-toolrunner-client`
- `mas-toolrunner` runs inside the VM (auto-started by rootfs)
- `cgroup` requires host cgroup v2 and sufficient privileges (usually root)
- `network.allowed_cidrs` requires host iptables (usually root) and only supports IP/CIDR
- `drives` only attaches disk images (ext4); mount inside the rootfs is your responsibility

---

## 8. E2E checklist (Linux + Firecracker)

### 8.1 Build tools
```
go build -o ./bin/mas-toolrunner ./cmd/mas-toolrunner
go build -o ./bin/mas-toolrunner-client ./cmd/mas-toolrunner-client
```

### 8.2 Build rootfs
```
BASE_ROOTFS_TAR=/path/to/rootfs.tar \
TOOLRUNNER_BIN=./bin/mas-toolrunner \
scripts/microvm/build_rootfs.sh
```

### 8.3 Start Firecracker (terminal A)
```
KERNEL_IMAGE=/path/to/vmlinux \
ROOTFS=./rootfs.ext4 \
VSOCK_UDS=/tmp/firecracker.vsock \
scripts/microvm/run_firecracker.sh
```

### 8.4 Verify vsock (terminal B)
```
VSOCK_UDS=/tmp/firecracker.vsock \
CLIENT_BIN=./bin/mas-toolrunner-client \
scripts/microvm/verify_vsock.sh
```

### 8.5 Verify control plane
```
mas-sandboxd -listen :8080 \
  -runtime microvm \
  -runtime-config ./scripts/microvm/microvm_config.example.json \
  -auth-token mytoken
```

```
MAS_SANDBOX_MODE=http \
MAS_SANDBOX_URL=http://127.0.0.1:8080 \
MAS_SANDBOX_TOKEN=mytoken \
go run ./examples/sandbox_agent
```

### 8.6 One-click E2E (optional)
```
KERNEL_IMAGE=/path/to/vmlinux \
BASE_ROOTFS_TAR=/path/to/rootfs.tar \
BUILD_ROOTFS=1 \
scripts/microvm/microvm_e2e.sh
```

---

## 9. Known gaps (not production-grade yet)

- Enforced network allowlist (currently validation only)
- Host filesystem allowlist mounting (microVM only sees VM paths today)
- Resource limits beyond cgroup + vCPU/mem (e.g., IO throttling)
- Full production E2E validation

---

## 10. FAQ

**Q: Why is the config using `mas-toolrunner-client` instead of `mas-toolrunner`?**
- `mas-toolrunner` runs inside the VM; the host must call it over vsock.
- So the host config uses the client.

**Q: Does LocalExecutor start the sandbox?**
- Yes. Each execution starts a `mas-sandboxd` process (stdin/stdout).

**Q: Can mac run microVM?**
- No. Firecracker only runs on Linux.
