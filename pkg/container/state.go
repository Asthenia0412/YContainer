package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ContainerState struct {
	PID        int    `json:"pid"`
	Status     string `json:"status"`
	BundlePath string `json:"bundle_path"`
}

func (m *Manager) SaveState(containerID string, state *ContainerState) error {
	stateDir := filepath.Join(m.storeDir, containerID)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	statePath := filepath.Join(stateDir, "state.json")
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}

	return nil
}

func (m *Manager) LoadState(containerID string) (*ContainerState, error) {
	statePath := filepath.Join(m.storeDir, containerID, "state.json")
	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}

	var state ContainerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}

	return &state, nil
}