package ns

import (
	"fmt"
	"os"
	"syscall"

	"github.com/Yancy/YContainer/internal/utils"
)

type NamespaceType int

const (
	CLONE_NEWPID  NamespaceType = syscall.CLONE_NEWPID
	CLONE_NEWNS   NamespaceType = syscall.CLONE_NEWNS
	CLONE_NEWUTS  NamespaceType = syscall.CLONE_NEWUTS
	CLONE_NEWIPC  NamespaceType = syscall.CLONE_NEWIPC
	CLONE_NEWNET  NamespaceType = syscall.CLONE_NEWNET
	CLONE_NEWUSER NamespaceType = syscall.CLONE_NEWUSER
)

type NamespaceConfig struct {
	PID  bool
	MNT  bool
	UTS  bool
	IPC  bool
	NET  bool
	USER bool
}

func (c *NamespaceConfig) CloneFlags() uintptr {
	var flags uintptr
	if c.PID {
		flags |= uintptr(CLONE_NEWPID)
	}
	if c.MNT {
		flags |= uintptr(CLONE_NEWNS)
	}
	if c.UTS {
		flags |= uintptr(CLONE_NEWUTS)
	}
	if c.IPC {
		flags |= uintptr(CLONE_NEWIPC)
	}
	if c.NET {
		flags |= uintptr(CLONE_NEWNET)
	}
	if c.USER {
		flags |= uintptr(CLONE_NEWUSER)
	}
	return flags
}

func DefaultConfig() *NamespaceConfig {
	return &NamespaceConfig{
		PID: true,
		MNT: true,
		UTS: true,
		IPC: true,
		NET: true,
	}
}

func SetHostname(name string) error {
	return syscall.Sethostname([]byte(name))
}

func JoinNamespace(pid int, nsType NamespaceType) error {
	path := fmt.Sprintf("/proc/%d/ns/%s", pid, nsTypeToStr(nsType))
	fd, err := syscall.Open(path, syscall.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open namespace fd: %w", err)
	}
	defer syscall.Close(fd)

	if err := syscall.Setns(fd, 0); err != nil {
		return fmt.Errorf("setns %s: %w", nsTypeToStr(nsType), err)
	}
	return nil
}

func GetCurrentNamespace(nsType NamespaceType) string {
	path := fmt.Sprintf("/proc/self/ns/%s", nsTypeToStr(nsType))
	link, err := os.Readlink(path)
	if err != nil {
		return "unknown"
	}
	return link
}

func nsTypeToStr(nsType NamespaceType) string {
	switch nsType {
	case CLONE_NEWPID:
		return "pid"
	case CLONE_NEWNS:
		return "mnt"
	case CLONE_NEWUTS:
		return "uts"
	case CLONE_NEWIPC:
		return "ipc"
	case CLONE_NEWNET:
		return "net"
	case CLONE_NEWUSER:
		return "user"
	default:
		return "unknown"
	}
}

func LogCurrentNamespaces(logger *utils.Logger) {
	logger.Info("Current namespaces:")
	logger.Info("  PID: %s", GetCurrentNamespace(CLONE_NEWPID))
	logger.Info("  MNT: %s", GetCurrentNamespace(CLONE_NEWNS))
	logger.Info("  NET: %s", GetCurrentNamespace(CLONE_NEWNET))
	logger.Info("  UTS: %s", GetCurrentNamespace(CLONE_NEWUTS))
	logger.Info("  IPC: %s", GetCurrentNamespace(CLONE_NEWIPC))
}