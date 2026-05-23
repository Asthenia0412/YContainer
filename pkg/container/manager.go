package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/types"
)

const YCDataDir = "/var/lib/yc"

type Manager struct {
	mu       sync.RWMutex
	logger   *utils.Logger
	storeDir string
}

func NewManager(logger *utils.Logger) *Manager {
	return &Manager{
		logger:   logger,
		storeDir: filepath.Join(YCDataDir, "containers"),
	}
}

func (m *Manager) Save(container *types.Container) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	containerDir := filepath.Join(m.storeDir, container.ID)
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return fmt.Errorf("create container dir: %w", err)
	}

	data, err := json.MarshalIndent(container, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal container: %w", err)
	}

	configPath := filepath.Join(containerDir, "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	m.logger.Info("Saved container %s to %s", container.ID, configPath)
	return nil
}

func (m *Manager) Load(containerID string) (*types.Container, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	configPath := filepath.Join(m.storeDir, containerID, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var container types.Container
	if err := json.Unmarshal(data, &container); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &container, nil
}

func (m *Manager) Delete(containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	containerDir := filepath.Join(m.storeDir, containerID)
	if err := os.RemoveAll(containerDir); err != nil {
		return fmt.Errorf("remove container dir: %w", err)
	}

	m.logger.Info("Deleted container %s", containerID)
	return nil
}

func (m *Manager) List() ([]*types.Container, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, err := os.ReadDir(m.storeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*types.Container{}, nil
		}
		return nil, fmt.Errorf("read containers dir: %w", err)
	}

	var containers []*types.Container
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		container, err := m.Load(entry.Name())
		if err != nil {
			m.logger.Warn("Failed to load container %s: %v", entry.Name(), err)
			continue
		}
		containers = append(containers, container)
	}

	return containers, nil
}

func (m *Manager) Exists(containerID string) bool {
	configPath := filepath.Join(m.storeDir, containerID, "config.json")
	_, err := os.Stat(configPath)
	return err == nil
}