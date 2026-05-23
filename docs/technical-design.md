# YContainer 技术设计文档

> **版本**: v1.0  
> **最后更新**: 2026-05-24  
> **项目目标**: 用 Go 构建一个兼容 Docker 语法的容器运行时，并逐步扩展为具备 Sidecar 基础设施能力的"容器 + 代理"平台

---

## 1. 项目概述

### 1.1 项目愿景

YContainer 是一个从零开始构建的容器化平台，目标包含两个层次：

| 层次 | 目标 | 对标 |
|------|------|------|
| **基础层** | 兼容 Docker CLI 语法的容器运行时 | Docker / podman |
| **扩展层** | 内置 Sidecar 代理的基础设施容器 | Istio + Envoy 的简化版 |

### 1.2 为什么做这个项目？

在生产环境中，每个服务旁边都会部署一个 Sidecar 代理，负责：
- 流量出入记录
- 限流（Rate Limiting）
- 鉴权（Authentication / Authorization）
- 熔断（Circuit Breaking）
- 可观测性（Metrics / Tracing）

将"容器运行时"和"基础设施代理"放在一起理解，能从根本上理解 **Service Mesh 的数据平面**是如何工作的。

### 1.3 设计原则

- **最小依赖**：尽量只使用 Go 标准库 + 系统调用
- **Docker 兼容**：CLI 语法和 OCI 标准尽量对齐
- **渐进复杂**：从单容器运行时 → 多容器管理 → Sidecar 代理，逐层叠加
- **可观测**：每个模块都有日志和 metrics

---

## 2. 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                    YContainer CLI                         │
│  (yc run / yc ps / yc exec / yc build / yc pull)        │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│                  YContainer Daemon                        │
│                                                          │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │ Container    │  │ Image        │  │ Network       │  │
│  │ Manager      │  │ Manager      │  │ Manager       │  │
│  └──────┬───────┘  └──────┬───────┘  └──────┬────────┘  │
│         │                 │                  │           │
│  ┌──────▼─────────────────▼──────────────────▼────────┐ │
│  │                  Runtime Layer                       │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │ │
│  │  │Namespace  │  │ Cgroup   │  │ OverlayFS       │  │ │
│  │  │Manager    │  │ Manager  │  │ Mount Manager   │  │ │
│  │  └──────────┘  └──────────┘  └──────────────────┘  │ │
│  └─────────────────────────────────────────────────────┘ │
│                                                          │
│  ┌─────────────────────────────────────────────────────┐ │
│  │              Sidecar Proxy Layer                     │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐  │ │
│  │  │HTTP Proxy│  │Rate      │  │Circuit Breaker   │  │ │
│  │  │          │  │Limiter   │  │+ Auth           │  │ │
│  │  └──────────┘  └──────────┘  └──────────────────┘  │ │
│  └─────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────┘
```

### 2.1 模块职责

| 模块 | 职责 | 关键组件 |
|------|------|---------|
| **CLI** | 命令行接口，兼容 Docker 语法 | `cmd/yc/` |
| **Container Manager** | 容器生命周期管理（创建/启动/停止/删除） | `pkg/container/` |
| **Image Manager** | OCI 镜像拉取、分层管理、本地存储 | `pkg/image/` |
| **Network Manager** | 容器网络（bridge/NAT/端口映射） | `pkg/network/` |
| **Namespace Manager** | PID/Net/MNT/USR 等命名空间隔离 | `pkg/ns/` |
| **Cgroup Manager** | CPU/内存/IO 资源限制 | `pkg/cgroup/` |
| **Mount Manager** | OverlayFS/UnionFS 挂载管理 | `pkg/mount/` |
| **Sidecar Proxy** | HTTP 代理 + 限流/熔断/鉴权 | `pkg/sidecar/` |

---

## 3. 核心数据结构

### 3.1 容器 (Container)

```go
type Container struct {
    ID        string            // 唯一 ID (sha256)
    Name      string            // 容器名
    PID       int               // 容器主进程 PID
    Status    ContainerStatus   // running/stopped/paused
    Image     string            // 镜像名
    Cmd       []string          // 启动命令
    Envs      []string          // 环境变量
    Mounts    []Mount           // 挂载点
    Ports     []PortMapping     // 端口映射
    Resources ResourceLimit     // 资源限制
    Network   ContainerNetwork  // 网络配置
    CreatedAt time.Time
    Labels    map[string]string
}

type ContainerStatus string

const (
    StatusCreated  ContainerStatus = "created"
    StatusRunning  ContainerStatus = "running"
    StatusStopped  ContainerStatus = "stopped"
    StatusPaused   ContainerStatus = "paused"
    StatusExited   ContainerStatus = "exited"
)

type ResourceLimit struct {
    MemoryLimit  int64  // bytes
    CPUShares    uint64 // 相对权重
    CPUPeriod    uint64 // μs
    CPUQuota     int64  // μs
    PidsMax      int64
}

type PortMapping struct {
    HostPort      int
    ContainerPort int
    Protocol      string // tcp/udp
}

type ContainerNetwork struct {
    Namespace string // 网络命名空间路径
    IPAddress string
    Bridge    string
}
```

### 3.2 Pod (多容器组)

```go
type Pod struct {
    ID         string
    Containers []*Container
    Namespaces PodNamespaces // 共享的命名空间
    Volumes    []Volume
    Network    PodNetwork
}

type PodNamespaces struct {
    PID   bool // 共享 PID 命名空间
    Net   bool // 共享 Net 命名空间
    IPC   bool // 共享 IPC 命名空间
    UTS   bool // 共享 UTS 命名空间
}

type PodNetwork struct {
    IP        string
    PortMaps  []PortMapping
    DNSServer string
}
```

### 3.3 Sidecar 配置

```go
type SidecarConfig struct {
    Enabled    bool
    ProxyPort  int              // Sidecar 监听端口
    AppPort    int              // 业务服务端口
    RateLimit  RateLimitConfig  // 限流配置
    Circuit    CircuitConfig    // 熔断配置
    Auth       AuthConfig       // 鉴权配置
    Logging    LoggingConfig    // 流量日志配置
}

type RateLimitConfig struct {
    Enabled     bool
    RequestsPer int     // 请求数
    PerSeconds  float64 // 时间窗口 (秒)
    Burst       int     // 突发上限
}

type CircuitConfig struct {
    Enabled           bool
    MaxRequests       int     // 半开后最大请求数
    Timeout           int     // 请求超时 (ms)
    SleepWindow       int     // 熔断后休眠 (ms)
    ErrorPercent      float64 // 触发熔断的错误率
}

type AuthConfig struct {
    Enabled  bool
    Mode     string   // jwt / apikey / oauth2
    Endpoint string   // 鉴权服务地址
}

type LoggingConfig struct {
    Enabled     bool
    LogDir      string
    AccessLog   bool   // 访问日志
    LatencyLog  bool   // 延迟日志
}
```

---

## 4. 阶段设计

### Phase 1: 容器运行时核心

#### 4.1 容器生命周期

```
yc run --name myapp alpine:latest /bin/sh
                    │
                    ▼
        1. 解析镜像 (或使用本地 rootfs)
                    │
                    ▼
        2. 创建容器目录 (rootfs 挂载点)
                    │
                    ▼
        3. 创建 Cgroup (cpu/memory/pids)
                    │
                    ▼
        4. 创建 Namespace (PID/Net/MNT/UTS/IPC)
                    │
                    ▼
        5. 挂载 OverlayFS / proc / sys
                    │
                    ▼
        6. pivot_root 切换到容器文件系统
                    │
                    ▼
        7. exec 用户命令
```

#### 4.2 Namespace 隔离

使用 Go `syscall` 包的 `clone()` 或 `unshare()` + `syscall.SysProcAttr`：

```go
// 创建新进程时的命名空间配置
cmd := exec.Command("/proc/self/exe", childArgs...)
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWPID |
                syscall.CLONE_NEWNS  |
                syscall.CLONE_NEWUTS |
                syscall.CLONE_NEWIPC |
                syscall.CLONE_NEWNET,
}
```

**涉及的 Namespace：**

| Namespace | 隔离内容 | Linux 常量 |
|-----------|---------|-----------|
| PID | 进程 ID | `CLONE_NEWPID` |
| MNT | 挂载点 | `CLONE_NEWNS` |
| NET | 网络栈 | `CLONE_NEWNET` |
| UTS | 主机名 | `CLONE_NEWUTS` |
| IPC | 进程间通信 | `CLONE_NEWIPC` |
| USER | 用户 ID | `CLONE_NEWUSER` |

#### 4.3 Cgroup 资源控制

通过写入 `/sys/fs/cgroup/` (cgroup v2) 来实现：

```
/sys/fs/cgroup/yc/<container-id>/
├── cpu.max          # CPU 配额
├── memory.max       # 内存上限
├── memory.current   # 当前内存
├── pids.max         # 最大进程数
├── io.max           # IO 限制
└── cgroup.procs     # 该组下的进程 PID
```

Phase 1 实现：
- `memory.max` - 内存硬限制
- `cpu.max` - CPU 时间配额
- `pids.max` - 最大进程数

#### 4.4 文件系统 OverlayFS

容器镜像的多层结构：

```
Upper Layer  (可写层)  ─ 容器修改部分
Diff Layer 2 (镜像层)  ─ 软件包
Diff Layer 1 (镜像层)  ─ 基础层
Lower Layer (镜像层)   ─ rootfs 基础
```

挂载命令对应的 Go 实现：

```go
func MountOverlayFS(lowerDir, upperDir, workDir, mergedDir string) error {
    opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s",
        lowerDir, upperDir, workDir)
    return syscall.Mount("overlay", mergedDir, "overlay", 0, opts)
}
```

---

### Phase 2: OCI 镜像管理

#### 2.1 镜像仓库交互

```
yc pull nginx:alpine
    │
    ▼
1. 解析镜像名 → registry/repo:tag
    │
    ▼
2. 获取 manifest (v2s2 或 OCI)
    │
    ▼
3. 获取 config (JSON) → 确定 Entrypoint/Cmd
    │
    ▼
4. 获取各层 blobs → 下载到本地缓存
    │
    ▼
5. 解压各层到 /var/lib/yc/images/<name>/<layer>/
```

**支持的 Registry：** Docker Hub、Harbor、自建 Registry

**认证支持：** Basic Auth / Bearer Token (Docker Registry v2)

#### 2.2 本地镜像存储结构

```
/var/lib/yc/
├── images/
│   └── <image-name>/
│       ├── manifest.json
│       ├── config.json
│       └── layers/
│           ├── <sha256-layer1>/
│           ├── <sha256-layer2>/
│           └── <sha256-layer3>/
├── containers/
│   └── <container-id>/
│       ├── config.json      # 容器配置
│       ├── rootfs/          # 合并后的 rootfs
│       └── log/             # 容器日志
└── volumes/
    └── <volume-name>/
```

#### 2.3 Dockerfile 构建（简化版）

```
yc build -t myapp:latest -f Dockerfile .
```

支持最基本的 Dockerfile 指令：

| 指令 | 实现方式 |
|------|---------|
| `FROM` | 拉取基础镜像 |
| `RUN` | 在容器中执行命令，提交为新层 |
| `COPY` | 复制文件到容器 |
| `WORKDIR` | 设置工作目录 |
| `CMD`/`ENTRYPOINT` | 设置启动命令 |
| `EXPOSE` | 声明端口 |
| `ENV` | 设置环境变量 |

---

### Phase 3: 多容器管理 (Pod)

#### 3.1 Pod 概念

```
┌────────────── Pod ──────────────┐
│                                  │
│  ┌─────────────┐                 │
│  │ pause 容器   │  ← 持有 Namespace    │
│  │ 仅 sleep     │                 │
│  └──────┬──────┘                 │
│         │ 共享 Namespace          │
│  ┌──────▼──────┐ ┌────────────┐ │
│  │ 业务容器     │ │ Sidecar   │ │
│  │ Go 服务     │ │ 代理       │ │
│  └─────────────┘ └────────────┘ │
└──────────────────────────────────┘
```

**pause 容器的作用：**
1. 创建并持有 Namespace
2. 当 Pod 内其他容器重启时，Namespace 不会销毁
3. 作为 Pod 的"命根子"

#### 3.2 共享 Network Namespace

```go
// 1. 创建 pause 容器，获取其 NetNS
pauseNetNS := fmt.Sprintf("/proc/%d/ns/net", pausePID)

// 2. 业务容器加入该 NetNS
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | ...,
}
// 注意：创建后再加入已有的 NetNS
```

#### 3.3 容器间通信

同一 Pod 内的容器通过 `localhost` 通信，因为共享同一个 NetNS。

---

### Phase 4: Sidecar 基础设施代理

#### 4.1 架构

```
             ┌───────── 外部请求 ─────────┐
             │                            │
             ▼                            │
      Sidecar 监听 :8443                   │ (入流量)
             │                            │
     ┌───────┴───────┐                    │
     │  流量拦截器     │                    │
     │  (iptables)    │                    │
     └───────┬───────┘                    │
             │                            │
     ┌───────▼───────┐                    │
     │  前置处理链     │                    │
     │  ├ 限流检查     │                    │
     │  ├ 鉴权检查     │                    │
     │  └ 熔断检查     │                    │
     └───────┬───────┘                    │
             │                            │
     ┌───────▼───────┐                    │
     │  HTTP 代理转发  │──→ localhost:8080 │ (业务服务)
     └───────┬───────┘                    │
             │                            │
     ┌───────▼───────┐                    │
     │  后置处理链     │                    │
     │  ├ 流量日志     │                    │
     │  ├ 延迟记录     │                    │
     │  └ Metrics     │                    │
     └───────────────┘                    │
                                          │
     出流量同理 (← 反向)                    │
     业务服务 → Sidecar → 外部              │
             ┌──────────────┘─────────────┘
```

#### 4.2 核心组件

**HTTP 反向代理（基于 `net/http/httputil`）：**

```go
type Proxy struct {
    target       *url.URL
    reverseProxy *httputil.ReverseProxy
    middleware    []Middleware
}

type Middleware func(http.Handler) http.Handler
```

**限流器（Token Bucket 算法）：**

```go
type TokenBucket struct {
    rate     float64    // 每秒放入的令牌数
    burst    int        // 桶容量
    tokens   float64    // 当前令牌数
    lastTime time.Time  // 上次放令牌时间
    mu       sync.Mutex
}

func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()
    now := time.Now()
    // 补充令牌
    elapsed := now.Sub(tb.lastTime).Seconds()
    tb.tokens = math.Min(tb.tokens+elapsed*tb.rate, float64(tb.burst))
    tb.lastTime = now
    // 消费令牌
    if tb.tokens >= 1 {
        tb.tokens--
        return true
    }
    return false
}
```

**熔断器（状态机）：**

```
状态转换:
    ┌─────────┐
    │  CLOSED  │  ← 正常工作
    └────┬─────┘
         │ 错误率 > 阈值
         ▼
    ┌─────────┐
    │  OPEN    │  ← 直接拒绝请求
    └────┬─────┘
         │ 超时时间到
         ▼
    ┌───────────┐
    │  HALF-OPEN │  ← 放少量请求试探
    └─────┬─────┘
       ┌──┴──┐
      成功    失败
       ▼      ▼
    CLOSED   OPEN
```

**流量记录：**

```go
type AccessLog struct {
    Timestamp   time.Time
    Method      string
    Path        string
    StatusCode  int
    Latency     time.Duration
    ClientIP    string
    RequestSize int64
    ResponseSize int64
}
```

#### 4.3 Middleware 链

```go
// 构建 Middleware 链
proxy := NewProxy(backendURL)
chain := Chain(
    proxy.Recovery(),      // 1. 异常恢复
    proxy.AccessLog(),     // 2. 访问日志
    proxy.RateLimit(),     // 3. 限流
    proxy.Auth(),          // 4. 鉴权
    proxy.CircuitBreaker(),// 5. 熔断
)(proxy.Handler())
```

#### 4.4 流量劫持（可选 - iptables 方式）

```bash
# 入流量重定向到 Sidecar
iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8443

# 出流量重定向到 Sidecar
iptables -t nat -A OUTPUT -p tcp --dport 80 -j REDIRECT --to-port 8443
```

---

### Phase 5: (可选) K8s 兼容

- Pod 生命周期对齐（Pending → Running → Succeeded/Failed）
- CNI 网络插件接口
- Container Runtime Interface (CRI) 部分实现
- 健康检查（Liveness / Readiness Probe）

---

## 5. CLI 命令设计

兼容 Docker 语法，使用 `yc` 作为命令名：

| 命令 | 功能 | Phase |
|------|------|-------|
| `yc run [OPTIONS] IMAGE COMMAND` | 创建并启动容器 | P1 |
| `yc ps` | 列出容器 | P1 |
| `yc stop CONTAINER` | 停止容器 | P1 |
| `yc rm CONTAINER` | 删除容器 | P1 |
| `yc exec CONTAINER COMMAND` | 进入容器执行命令 | P1 |
| `yc logs CONTAINER` | 查看日志 | P1 |
| `yc pull IMAGE` | 拉取镜像 | P2 |
| `yc images` | 列出镜像 | P2 |
| `yc rmi IMAGE` | 删除镜像 | P2 |
| `yc build -t TAG PATH` | 构建镜像 | P2 |
| `yc pod run ...` | 创建 Pod (多容器) | P3 |
| `yc pod ps` | 列出 Pod | P3 |
| `yc proxy --app-port PORT` | 启动 Sidecar | P4 |

---

## 6. 目录结构

```
YContainer/
├── cmd/
│   └── yc/
│       ├── main.go              # 入口
│       ├── run.go               # yc run
│       ├── ps.go                # yc ps
│       ├── stop.go              # yc stop
│       ├── rm.go                # yc rm
│       ├── exec.go              # yc exec
│       ├── logs.go              # yc logs
│       ├── pull.go              # yc pull
│       ├── images.go            # yc images
│       ├── build.go             # yc build
│       ├── pod.go               # yc pod subcommand
│       └── proxy.go             # yc proxy (sidecar)
├── pkg/
│   ├── container/
│   │   ├── container.go         # Container struct
│   │   ├── manager.go           # 容器管理器
│   │   ├── lifecycle.go         # 生命周期 (创建/启动/停止)
│   │   └── state.go             # 状态持久化
│   ├── ns/
│   │   └── namespace.go         # Namespace 操作
│   ├── cgroup/
│   │   ├── cgroup.go            # Cgroup v2 管理器
│   │   ├── cpu.go               # CPU 控制
│   │   ├── memory.go            # 内存控制
│   │   └── pids.go              # 进程数控制
│   ├── mount/
│   │   ├── overlay.go           # OverlayFS 挂载
│   │   └── mount.go             # 通用挂载工具
│   ├── image/
│   │   ├── image.go             # Image struct
│   │   ├── puller.go            # 镜像拉取
│   │   ├── local.go             # 本地镜像存储
│   │   └── builder.go           # Dockerfile 构建
│   ├── network/
│   │   ├── network.go           # 网络管理器
│   │   ├── bridge.go            # Bridge 网络
│   │   └── portmap.go           # 端口映射
│   ├── pod/
│   │   ├── pod.go               # Pod struct
│   │   ├── manager.go           # Pod 管理器
│   │   └── pause.go             # pause 容器
│   ├── sidecar/
│   │   ├── proxy.go             # HTTP 反向代理
│   │   ├── rate_limit.go        # 限流器
│   │   ├── circuit_breaker.go   # 熔断器
│   │   ├── auth.go              # 鉴权
│   │   ├── access_log.go        # 流量日志
│   │   └── middleware.go        # Middleware 框架
│   └── types/
│       └── types.go             # 通用类型定义
├── internal/
│   └── utils/
│       ├── id.go                # ID 生成
│       ├── logger.go            # 日志工具
│       └── cmd.go               # 命令执行工具
├── docs/
│   ├── technical-design.md      # 本文件
│   ├── api.md                   # API 文档
│   └── examples/               # 使用示例
├── scripts/
│   ├── setup.sh                 # 环境准备
│   └── test.sh                  # 测试脚本
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 7. 关键系统调用

Go 操作容器的核心系统调用：

| 系统调用 | Go 表示 | 用途 |
|---------|---------|------|
| `clone()` | `syscall.SysProcAttr.Cloneflags` | 创建新进程并设置 Namespace |
| `unshare()` | `syscall.Unshare()` | 为当前进程设置 Namespace |
| `setns()` | `syscall.Setns()` | 加入已有 Namespace |
| `pivot_root()` | `syscall.PivotRoot()` | 切换进程根文件系统 |
| `mount()` | `syscall.Mount()` | 挂载文件系统 |
| `unmount()` | `syscall.Unmount()` | 卸载文件系统 |
| `sethostname()` | `syscall.Sethostname()` | 设置容器主机名 |
| `prctl()` | `syscall.Prctl()` | 设置子进程信号处理 |

---

## 8. 技术栈

| 技术 | 用途 | 理由 |
|------|------|------|
| **Go** | 主开发语言 | 系统调用封装好，并发能力强，静态编译 |
| **Cobra** | CLI 框架 | 成熟的 Go CLI 框架，便于子命令管理 |
| **Go 标准库** | HTTP/IO/Syscall | 最小依赖原则 |
| **libcontainer** (参考) | Linux 容器参考实现 | runc 的核心库，学习但不直接依赖 |
| **vishvananda/netlink** | 网络配置 | 操作 bridge/veth 的 Go 库 |

> **注**：初期尽量用标准库实现，逐步引入外部依赖。

---

## 9. 开发与测试

### 9.1 环境要求

- Linux (或 macOS + Docker Desktop / Lima)
- Go 1.22+
- root 权限（容器操作需要 Capabilities）
- 支持 OverlayFS 的文件系统

### 9.2 测试策略

| 级别 | 方式 | 内容 |
|------|------|------|
| Unit | `go test` | 各模块单元测试 |
| Integration | Linux 环境下的端到端测试 | 完整创建一个容器并验证 |
| Sidecar | HTTP 请求测试 | 验证限流/熔断/鉴权逻辑 |

### 9.3 Makefile 目标

```makefile
build    # 编译 yc 二进制
test     # 运行单元测试
test-e2e # 运行端到端测试
clean    # 清理构建产物
install  # 安装到系统
```

---

## 10. 术语表

| 术语 | 含义 |
|------|------|
| **Namespace** | Linux 内核提供的资源隔离机制 |
| **Cgroup** | Linux 内核提供的资源限制机制 |
| **OverlayFS** | Linux 联合文件系统，实现容器分层 |
| **OCI** | Open Container Initiative，容器标准 |
| **Pod** | 一组共享 Namespace 的容器集合 |
| **Sidecar** | 与服务容器一同部署的辅助容器 |
| **Service Mesh** | 服务间通信的基础设施层 |
| **CNI** | Container Network Interface，容器网络标准 |
| **CRI** | Container Runtime Interface，容器运行时标准 |
| **pause 容器** | 用于持有 Pod Namespace 的占位容器 |

---

## 附录 A: 参考项目

- [runc](https://github.com/opencontainers/runc) - OCI 容器运行时参考实现
- [containerd](https://github.com/containerd/containerd) - 工业级容器运行时
- [Envoy](https://github.com/envoyproxy/envoy) - 高性能 Sidecar 代理
- [Istio](https://github.com/istio/istio) - Service Mesh 控制平面
- [gVisor](https://github.com/google/gvisor) - 用户态内核容器
- [Porter](https://github.com/skx/porter) - Go 编写的简易容器（学习参考）

## 附录 B: 学习路径

1. **Linux 内核基础**：Namespace、Cgroup、文件系统
2. **Go 系统编程**：`syscall`、`os/exec`、信号处理
3. **OCI 标准**：Image Spec、Runtime Spec
4. **网络基础**：Bridge、NAT、iptables、veth pair
5. **服务网格**：Sidecar 模式、Envoy 架构、Istio 控制平面