package container

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/cgroup"
	"github.com/Yancy/YContainer/pkg/mount"
	"github.com/Yancy/YContainer/pkg/ns"
	"github.com/Yancy/YContainer/pkg/types"
)

type LifecycleManager struct {
	containerMgr *Manager
	logger       *utils.Logger
}

func NewLifecycleManager(containerMgr *Manager, logger *utils.Logger) *LifecycleManager {
	return &LifecycleManager{
		containerMgr: containerMgr,
		logger:       logger,
	}
}

type RunConfig struct {
	Name      string
	Image     string
	Cmd       []string
	Envs      []string
	Mounts    []types.Mount
	Ports     []types.PortMapping
	Resources types.ResourceLimit
	Labels    map[string]string
	RootFS    string
}

func (lm *LifecycleManager) Create(config RunConfig) (*types.Container, error) {
	id := utils.GenerateContainerID()

	container := &types.Container{
		ID:        id,
		Name:      config.Name,
		PID:       0,
		Status:    types.StatusCreated,
		Image:     config.Image,
		Cmd:       config.Cmd,
		Envs:      config.Envs,
		Mounts:    config.Mounts,
		Ports:     config.Ports,
		Resources: config.Resources,
		CreatedAt: time.Now(),
		Labels:    config.Labels,
	}

	if err := lm.containerMgr.Save(container); err != nil {
		return nil, fmt.Errorf("save container: %w", err)
	}

	lm.logger.Info("Created container %s (image: %s)", id, config.Image)
	return container, nil
}

func (lm *LifecycleManager) Start(container *types.Container, rootFS string) error {
	nsConfig := ns.DefaultConfig()

	cmd := exec.Command("/proc/self/exe", "child", container.ID)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: uintptr(nsConfig.CloneFlags()),
		Unshareflags: uintptr(syscall.CLONE_NEWNS),
	}

	cmd.Env = append(os.Environ(), container.Envs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start container process: %w", err)
	}

	container.PID = cmd.Process.Pid
	container.Status = types.StatusRunning
	lm.logger.Info("Started container %s with PID %d", container.ID, container.PID)

	cgroupMgr := cgroup.NewManager(container.ID, lm.logger)
	if err := cgroupMgr.Create(); err != nil {
		return fmt.Errorf("create cgroup: %w", err)
	}

	if err := cgroupMgr.SetLimits(container.Resources); err != nil {
		return fmt.Errorf("set resource limits: %w", err)
	}

	if err := cgroupMgr.AddProcess(container.PID); err != nil {
		return fmt.Errorf("add process to cgroup: %w", err)
	}

	if err := lm.containerMgr.Save(container); err != nil {
		return fmt.Errorf("save container state: %w", err)
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			lm.logger.Warn("Container %s exited: %v", container.ID, err)
		}
		container.Status = types.StatusExited
		lm.containerMgr.Save(container)
	}()

	return nil
}

func (lm *LifecycleManager) Stop(container *types.Container) error {
	if container.Status != types.StatusRunning {
		return fmt.Errorf("container %s is not running", container.ID)
	}

	process, err := os.FindProcess(container.PID)
	if err != nil {
		return fmt.Errorf("find process %d: %w", container.PID, err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		lm.logger.Warn("SIGTERM failed for PID %d, trying SIGKILL: %v", container.PID, err)
		if err := process.Kill(); err != nil {
			return fmt.Errorf("kill process %d: %w", container.PID, err)
		}
	}

	container.Status = types.StatusStopped
	if err := lm.containerMgr.Save(container); err != nil {
		return fmt.Errorf("save container state: %w", err)
	}

	cgroupMgr := cgroup.NewManager(container.ID, lm.logger)
	if err := cgroupMgr.Delete(); err != nil {
		lm.logger.Warn("Failed to clean cgroup for %s: %v", container.ID, err)
	}

	lm.logger.Info("Stopped container %s", container.ID)
	return nil
}

func (lm *LifecycleManager) Exec(containerID string, cmdArgs []string) error {
	container, err := lm.containerMgr.Load(containerID)
	if err != nil {
		return fmt.Errorf("load container: %w", err)
	}

	if container.Status != types.StatusRunning {
		return fmt.Errorf("container %s is not running", containerID)
	}

	nsConfig := ns.DefaultConfig()
	execCmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	execCmd.SysProcAttr = &syscall.SysProcAttr{}

	netNs := fmt.Sprintf("/proc/%d/ns/net", container.PID)
	pidNs := fmt.Sprintf("/proc/%d/ns/pid", container.PID)
	mntNs := fmt.Sprintf("/proc/%d/ns/mnt", container.PID)

	execCmd.SysProcAttr.Cloneflags = uintptr(nsConfig.CloneFlags())

	lm.logger.Info("Exec into container %s (PID %d): %v", containerID, container.PID, cmdArgs)
	lm.logger.Info("  net ns: %s", netNs)
	lm.logger.Info("  pid ns: %s", pidNs)
	lm.logger.Info("  mnt ns: %s", mntNs)

	return execCmd.Run()
}

func prepareRootFS(rootFS string) error {
	for _, dir := range []string{"proc", "sys", "dev", "tmp", "etc"} {
		if err := os.MkdirAll(filepath.Join(rootFS, dir), 0755); err != nil {
			return fmt.Errorf("create %s in rootfs: %w", dir, err)
		}
	}
	return nil
}

func ChildProcess(containerID string) error {
	logger := utils.DefaultLogger
	logger.Info("Starting child process for container %s", containerID)

	containerMgr := NewManager(logger)
	container, err := containerMgr.Load(containerID)
	if err != nil {
		return fmt.Errorf("load container config: %w", err)
	}

	rootFS := filepath.Join(YCDataDir, "containers", containerID, "rootfs")
	if err := prepareRootFS(rootFS); err != nil {
		return fmt.Errorf("prepare rootfs: %w", err)
	}

	if err := ns.SetHostname(fmt.Sprintf("yc-%s", containerID[:8])); err != nil {
		return fmt.Errorf("set hostname: %w", err)
	}

	if err := mount.SetupRootFS(rootFS, logger); err != nil {
		return fmt.Errorf("setup rootfs mounts: %w", err)
	}

	if err := mount.PivotRoot(rootFS); err != nil {
		return fmt.Errorf("pivot root: %w", err)
	}

	logger.Info("Container setup complete, executing command: %v", container.Cmd)
	
	cmd := exec.Command(container.Cmd[0], container.Cmd[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = container.Envs

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("execute command: %w", err)
	}

	return nil
}