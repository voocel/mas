# Sandbox 使用与架构（中文）

本仓库的 sandbox 相关说明以**中英文两份**为准，其余文档均为跳转或存档。

---

## 1. 这是什么

- MAS 是多 Agent 库。
- Sandbox 是“工具执行/治理层”，用于把工具调用隔离出去并进行策略控制。
- `mas-sandboxd` 是控制面服务，负责接收工具调用、执行策略、调度运行时。

**关键概念**：
- **Agent/Runner** 负责“思考与编排”。
- **Sandbox** 负责“工具执行与治理”。

---

## 2. 进程与责任（最简）

```
Agent/Runner → ToolExecutor
  ├─ LocalExecutor → mas-sandboxd（stdin/stdout）
  └─ SandboxExecutor → mas-sandboxd（HTTP）

mas-sandboxd → runtime/local | runtime/microvm
runtime/microvm → Firecracker VM → mas-toolrunner（guest）
```

- **mas-sandboxd**：控制面（宿主机进程）
- **mas-toolrunner**：VM 内工具执行服务
- **mas-toolrunner-client**：宿主机到 VM 的 vsock 调用器

---

## 3. 开发联调（最简单）

**示例**：`examples/sandbox_agent`

### 方式 A：本地进程（stdin/stdout）

```
go run ./examples/sandbox_agent
```

### 方式 B：HTTP 控制面（推荐结构）

启动控制面：
```
mas-sandboxd -listen :8080 -runtime local
```

SDK 侧：
```
MAS_SANDBOX_MODE=http \
MAS_SANDBOX_URL=http://127.0.0.1:8080 \
go run ./examples/sandbox_agent
```

---

## 4. microVM（真实隔离）使用方式

### 4.1 运行控制面（microVM）

```
mas-sandboxd -listen :8080 \
  -runtime microvm \
  -runtime-config ./scripts/microvm/microvm_config.example.json \
  -auth-token mytoken
```

### 4.2 SDK 调用（不变）

```
MAS_SANDBOX_MODE=http \
MAS_SANDBOX_URL=http://127.0.0.1:8080 \
MAS_SANDBOX_TOKEN=mytoken \
go run ./examples/sandbox_agent
```

---

## 5. mac 开发联调建议

- mac 无法原生运行 Firecracker
- 开发阶段使用 `local` 运行时即可：

```
mas-sandboxd -listen :8080 -runtime local
```

如果需要接近线上行为，建议在 mac 上启动 Linux VM，然后在 VM 内运行 microVM 运行时。

---

## 6. microVM 运行依赖（最小清单）

你需要准备：
- Firecracker（宿主机）
- Linux kernel image
- rootfs（ext4）
- rootfs 内置并自启 `mas-toolrunner`

相关脚本：
- `scripts/microvm/build_rootfs.sh`
- `scripts/microvm/run_firecracker.sh`
- `scripts/microvm/verify_vsock.sh`
- `scripts/microvm/microvm_e2e.sh`

---

## 7. microVM 配置要点

配置文件示例：`scripts/microvm/microvm_config.example.json`

关键字段：
- `firecracker_bin` / `kernel_image` / `rootfs`
- `vsock.cid / port / uds_path`
- `tool_runner.command`（宿主机侧调用器）
- `network.tap_device`（启用网络时必填）
- `pool.size`（>1 时需 `{id}` 占位符）

**注意：**
- `tool_runner.command` 是 **宿主机** 的 `mas-toolrunner-client`
- `mas-toolrunner` 在 **VM 内**运行（由 rootfs 自启）

---

## 8. E2E 验证清单（Linux + Firecracker）

### 8.1 编译工具
```
go build -o ./bin/mas-toolrunner ./cmd/mas-toolrunner
go build -o ./bin/mas-toolrunner-client ./cmd/mas-toolrunner-client
```

### 8.2 构建 rootfs
```
BASE_ROOTFS_TAR=/path/to/rootfs.tar \
TOOLRUNNER_BIN=./bin/mas-toolrunner \
scripts/microvm/build_rootfs.sh
```

### 8.3 启动 Firecracker（终端 A）
```
KERNEL_IMAGE=/path/to/vmlinux \
ROOTFS=./rootfs.ext4 \
VSOCK_UDS=/tmp/firecracker.vsock \
scripts/microvm/run_firecracker.sh
```

### 8.4 验证 vsock（终端 B）
```
VSOCK_UDS=/tmp/firecracker.vsock \
CLIENT_BIN=./bin/mas-toolrunner-client \
scripts/microvm/verify_vsock.sh
```

### 8.5 验证控制面
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

### 8.6 一键 E2E（可选）
```
KERNEL_IMAGE=/path/to/vmlinux \
BASE_ROOTFS_TAR=/path/to/rootfs.tar \
BUILD_ROOTFS=1 \
scripts/microvm/microvm_e2e.sh
```

---

## 9. 已知缺口（真实生产级尚需）

- 网络 allowlist 强制隔离（当前是参数校验）
- host 文件系统白名单挂载（microVM 目前只访问 VM 内路径）
- 资源限制（CPU/内存/IO 等）
- 真机 E2E 完整收口

---

## 10. 常见问题

**Q：为什么配置里是 `mas-toolrunner-client`，而不是 `mas-toolrunner`？**
- `mas-toolrunner` 在 VM 内运行，宿主机只能通过 vsock 调用它。
- 所以宿主机配置的是 client。

**Q：LocalExecutor 会不会启动沙箱？**
- 会。每次执行启动一个 `mas-sandboxd` 进程（stdin/stdout）。

**Q：mac 能跑 microVM 吗？**
- 不能。只能在 Linux 跑 Firecracker。
