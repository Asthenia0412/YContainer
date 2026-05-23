package pod

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/container"
	"github.com/Yancy/YContainer/pkg/types"
)

type Pod struct {
	ID         string
	Name       string
	Containers []*types.Container
	PausePID   int
	CreatedAt  time.Time
	Status     string
	Namespaces PodNamespaces
}

type PodNamespaces struct {
	Net bool
	PID bool
	IPC bool
	UTS bool
}

type Manager struct {
	logger       *utils.Logger
	containerMgr *container.Manager
}

func NewManager(logger *utils.Logger, containerMgr *container.Manager) *Manager {
	return &Manager{
		logger:       logger,
		containerMgr: containerMgr,
	}
}

func (m *Manager) CreatePod(name string) (*Pod, error) {
	id := utils.GeneratePodID()

	pausePID, err := m.startPauseContainer(id)
	if err != nil {
		return nil, fmt.Errorf("start pause container: %w", err)
	}

	pod := &Pod{
		ID:        id,
		Name:      name,
		PausePID:  pausePID,
		CreatedAt: time.Now(),
		Status:    "running",
		Namespaces: PodNamespaces{
			Net: true,
			PID: true,
			IPC: true,
			UTS: true,
		},
	}

	m.logger.Info("Created pod %s (pause PID: %d)", id, pausePID)
	return pod, nil
}

func (m *Manager) AddContainer(pod *Pod, c *types.Container) error {
	netNsPath := fmt.Sprintf("/proc/%d/ns/net", pod.PausePID)

	cmd := exec.Command("/proc/self/exe", "child", c.ID)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC,
	}

	m.logger.Info("Adding container %s to pod %s (netns: %s)", c.ID, pod.ID, netNsPath)

	pod.Containers = append(pod.Containers, c)
	return nil
}

func (m *Manager) startPauseContainer(podID string) (int, error) {
	cmd := exec.Command("/proc/self/exe", "pause", podID)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNET,
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start pause: %w", err)
	}

	time.Sleep(100 * time.Millisecond)
	m.logger.Info("Pause container started for pod %s (PID: %d)", podID, cmd.Process.Pid)
	return cmd.Process.Pid, nil
}

func PauseProcess() error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
	return nil
}