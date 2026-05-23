package types

import "time"

type ContainerStatus string

const (
	StatusCreated ContainerStatus = "created"
	StatusRunning ContainerStatus = "running"
	StatusStopped ContainerStatus = "stopped"
	StatusPaused  ContainerStatus = "paused"
	StatusExited  ContainerStatus = "exited"
)

type ResourceLimit struct {
	MemoryLimit int64  // bytes, 0 means unlimited
	CPUShares   uint64 // relative weight
	CPUPeriod   uint64 // μs
	CPUQuota    int64  // μs, -1 means unlimited
	PidsMax     int64  // max number of processes
}

type PortMapping struct {
	HostPort      int
	ContainerPort int
	Protocol      string // tcp / udp
}

type Mount struct {
	Source      string
	Destination string
	Type        string
	Options     string
	ReadOnly    bool
}

type ContainerNetwork struct {
	NamespacePath string
	IPAddress     string
	Bridge        string
	PortMappings  []PortMapping
}

type Container struct {
	ID        string
	Name      string
	PID       int
	Status    ContainerStatus
	Image     string
	Cmd       []string
	Envs      []string
	Mounts    []Mount
	Ports     []PortMapping
	Resources ResourceLimit
	Network   ContainerNetwork
	CreatedAt time.Time
	Labels    map[string]string
}