package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/types"
)

const (
	CgroupRoot   = "/sys/fs/cgroup"
	YCGroupName  = "yc"
)

type Manager struct {
	cgroupPath string
	logger     *utils.Logger
}

func NewManager(containerID string, logger *utils.Logger) *Manager {
	return &Manager{
		cgroupPath: filepath.Join(CgroupRoot, YCGroupName, containerID),
		logger:     logger,
	}
}

func (m *Manager) Create() error {
	if err := os.MkdirAll(m.cgroupPath, 0755); err != nil {
		return fmt.Errorf("create cgroup directory: %w", err)
	}
	m.logger.Info("Created cgroup: %s", m.cgroupPath)
	return nil
}

func (m *Manager) Delete() error {
	if err := os.RemoveAll(m.cgroupPath); err != nil {
		return fmt.Errorf("remove cgroup: %w", err)
	}
	m.logger.Info("Removed cgroup: %s", m.cgroupPath)
	return nil
}

func (m *Manager) SetLimits(res types.ResourceLimit) error {
	if res.MemoryLimit > 0 {
		if err := writeFile(filepath.Join(m.cgroupPath, "memory.max"),
			strconv.FormatInt(res.MemoryLimit, 10)); err != nil {
			return fmt.Errorf("set memory limit: %w", err)
		}
		m.logger.Info("Set memory limit: %d bytes", res.MemoryLimit)
	}

	if res.CPUQuota > 0 || res.CPUPeriod > 0 {
		period := res.CPUPeriod
		if period == 0 {
			period = 100000
		}
		quota := res.CPUQuota
		if quota <= 0 {
			quota = -1
		}
		if err := writeFile(filepath.Join(m.cgroupPath, "cpu.max"),
			fmt.Sprintf("%d %d", quota, period)); err != nil {
			return fmt.Errorf("set cpu limit: %w", err)
		}
		m.logger.Info("Set cpu limit: quota=%d period=%d", quota, period)
	}

	if res.PidsMax > 0 {
		if err := writeFile(filepath.Join(m.cgroupPath, "pids.max"),
			strconv.FormatInt(res.PidsMax, 10)); err != nil {
			return fmt.Errorf("set pids limit: %w", err)
		}
		m.logger.Info("Set pids limit: %d", res.PidsMax)
	}

	return nil
}

func (m *Manager) AddProcess(pid int) error {
	if err := writeFile(filepath.Join(m.cgroupPath, "cgroup.procs"),
		strconv.Itoa(pid)); err != nil {
		return fmt.Errorf("add process %d to cgroup: %w", pid, err)
	}
	m.logger.Info("Added process %d to cgroup", pid)
	return nil
}

func (m *Manager) GetMemoryUsage() (int64, error) {
	data, err := os.ReadFile(filepath.Join(m.cgroupPath, "memory.current"))
	if err != nil {
		return 0, fmt.Errorf("read memory usage: %w", err)
	}
	val, err := strconv.ParseInt(string(data[:len(data)-1]), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse memory usage: %w", err)
	}
	return val, nil
}

func (m *Manager) Exists() bool {
	_, err := os.Stat(m.cgroupPath)
	return err == nil
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func IsCgroupV2() bool {
	data, err := os.ReadFile(filepath.Join(CgroupRoot, "cgroup.controllers"))
	return err == nil && len(data) > 0
}