# YContainer

A lightweight container runtime built from scratch in Go, compatible with Docker CLI syntax, with built-in sidecar proxy capabilities for rate limiting, authentication, and circuit breaking.

一个使用 Go 语言从零构建的轻量级容器运行时，兼容 Docker CLI 语法，并内置支持限流、鉴权和熔断的 Sidecar 代理能力。

---

## Features / 特性

- **Container Lifecycle** / 容器生命周期: create, start, stop, remove, exec, logs -- Docker-compatible CLI
- **Namespace Isolation** / 命名空间隔离: PID, Network, Mount, UTS, IPC
- **Resource Control** / 资源控制: CPU quota, memory limit, process count via cgroup v2
- **OverlayFS** / 联合文件系统: layered image mounting with pivot_root
- **OCI Image Pulling** / OCI 镜像拉取: pull images from Docker Hub with manifest and layer management
- **Bridge Networking** / 桥接网络: veth pair, NAT, port mapping via iptables
- **Multi-Container Pod** / 多容器组: shared network namespace via pause container
- **Sidecar Proxy** / 边车代理: HTTP reverse proxy with middleware chain
- **Rate Limiting** / 限流: token bucket algorithm
- **Circuit Breaking** / 熔断: three-state machine (Closed / Open / Half-Open)
- **Authentication** / 鉴权: API key, JWT, Basic Auth middleware

---

## Architecture / 架构

```
CLI (yc)                          Docker-compatible commands
  |
Daemon Layer                      Container / Image / Network / Pod managers
  |
Runtime Layer                     Namespace / Cgroup / OverlayFS / Bridge
  |
Sidecar Layer                     HTTP Proxy / Rate Limiter / Circuit Breaker / Auth
```

Key design: The project follows a bottom-up learning path, starting from raw Linux kernel primitives (namespace, cgroup, pivot_root) and progressively building up to Pod and Sidecar levels.

核心理念：自底向上的学习路径，从 Linux 内核原语（命名空间、cgroup、pivot_root）开始，逐步构建到 Pod 和 Sidecar 层面。

---

## Getting Started / 快速开始

### Prerequisites / 前置要求

- Linux (or macOS with Docker Desktop / Lima for testing)
- Go 1.22+
- root privileges (for namespace and cgroup operations)
- cgroup v2 enabled (`/sys/fs/cgroup/cgroup.controllers` must exist)

### Build / 编译

```bash
git clone https://github.com/Yancy/YContainer.git
cd YContainer
go mod tidy
make build
```

The binary will be at `build/yc`.

### Basic Usage / 基本使用

```bash
# List containers / 列出容器
sudo ./build/yc ps

# Run a container / 运行容器
sudo ./build/yc run --name myapp alpine:latest /bin/sh

# Run with resource limits / 带资源限制运行
sudo ./build/yc run --memory 67108864 --cpus 0.5 alpine:latest /bin/sh

# Stop a container / 停止容器
sudo ./build/yc stop <container-id>

# Execute a command in a running container / 在运行中的容器中执行命令
sudo ./build/yc exec <container-id> /bin/ls

# Remove a container / 删除容器
sudo ./build/yc rm <container-id>
```

### Image Management / 镜像管理

```bash
# Pull an image / 拉取镜像
sudo ./build/yc pull alpine:latest

# List local images / 列出本地镜像
sudo ./build/yc images
```

### Pod (Multi-Container) / 多容器组

```bash
# Create a pod with a container / 创建 Pod 并运行容器
sudo ./build/yc pod run alpine:latest /bin/sleep 60

# List pods / 列出 Pod
sudo ./build/yc pod ps
```

### Sidecar Proxy / 边车代理

```bash
# Start sidecar proxy forwarding to app on port 8080 / 启动 Sidecar 代理转发到 8080 端口
sudo ./build/yc proxy --app-port 8080 --proxy-port 8443

# With rate limiting / 启用限流 (100 requests/second)
sudo ./build/yc proxy --app-port 8080 --rate-limit 100 --burst 10

# With authentication / 启用鉴权
sudo ./build/yc proxy --app-port 8080 --auth apikey --api-key my-secret-key

# With circuit breaker / 启用熔断
sudo ./build/yc proxy --app-port 8080 --circuit-error 50 --circuit-sleep 5000
```

---

## Project Structure / 项目结构

```
YContainer/
├── cmd/yc/               # CLI entry point and command definitions
│   ├── main.go           # root command
│   ├── run.go            # yc run
│   ├── ps.go             # yc ps
│   ├── stop.go           # yc stop
│   ├── rm.go             # yc rm
│   ├── exec.go           # yc exec
│   ├── logs.go           # yc logs
│   ├── pull.go           # yc pull
│   ├── images.go         # yc images
│   ├── build.go          # yc build
│   ├── pod.go            # yc pod
│   ├── proxy.go          # yc proxy (sidecar)
│   ├── child.go          # internal: container child process
│   └── pause.go          # internal: pause container
├── pkg/
│   ├── types/            # core data structures (Container, ResourceLimit, etc.)
│   ├── ns/               # namespace operations (PID/Net/MNT/UTS/IPC)
│   ├── cgroup/           # cgroup v2 resource control
│   ├── mount/            # OverlayFS mount and pivot_root
│   ├── container/        # container CRUD, lifecycle, state persistence
│   ├── network/          # bridge, veth pair, port mapping
│   ├── image/            # OCI image pulling, manifest parsing
│   ├── pod/              # pod management and pause container
│   └── sidecar/          # HTTP proxy, rate limiter, circuit breaker, auth, logging
├── internal/utils/       # ID generation, structured logger
├── docs/                 # technical design documents
└── Makefile              # build, test, clean targets
```

---

## Learning Path / 学习路径

This project is designed as a progressive learning tool. Each phase builds on the previous one:

本项目设计为渐进式学习工具，每个阶段建立在前一阶段的基础上：

| Phase | Topic / 主题 | Core Concepts / 核心概念 |
|-------|-------------|------------------------|
| P1 | Container Runtime / 容器运行时 | Namespace, Cgroup, OverlayFS, pivot_root, proc/sys mounting |
| P2 | Image Management / 镜像管理 | OCI Image Spec, Docker Registry v2 API, layer download and extraction |
| P3 | Pod Semantics / 多容器组 | pause container, shared namespaces, multi-process lifecycle |
| P4 | Sidecar Proxy / 边车代理 | HTTP reverse proxy, middleware chain, token bucket, circuit breaker |
| P5 | K8s Compatibility (optional) | Pod lifecycle, CNI network, CRI interface, eBPF traffic interception |

---

## System Calls Used / 涉及的系统调用

| Syscall | Go Binding | Purpose / 用途 |
|---------|-----------|----------------|
| clone() | `SysProcAttr.Cloneflags` | Create process with new namespaces |
| unshare() | `syscall.Unshare()` | Disassociate from parent namespaces |
| setns() | `syscall.Setns()` | Join an existing namespace |
| pivot_root() | `syscall.PivotRoot()` | Change root filesystem |
| mount() | `syscall.Mount()` | Mount filesystems (proc, sys, overlay, tmpfs) |
| sethostname() | `syscall.Sethostname()` | Set container hostname |

---

## References / 参考资源

- [runc](https://github.com/opencontainers/runc) - OCI container runtime reference implementation
- [containerd](https://github.com/containerd/containerd) - Industrial-grade container runtime
- [Envoy](https://github.com/envoyproxy/envoy) - High-performance sidecar proxy
- [Istio](https://github.com/istio/istio) - Service mesh control plane
- [OCI Runtime Spec](https://github.com/opencontainers/runtime-spec) - Open Container Initiative standard

---

## License / 许可证

MIT