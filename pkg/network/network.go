package network

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Yancy/YContainer/internal/utils"
	"github.com/Yancy/YContainer/pkg/types"
)

const (
	BridgeName  = "yc0"
	BridgeCIDR  = "172.17.0.0/16"
	ContainerIP = "172.17.0."
)

type Manager struct {
	logger *utils.Logger
}

func NewManager(logger *utils.Logger) *Manager {
	return &Manager{logger: logger}
}

func (m *Manager) CreateBridge() error {
	exists, err := bridgeExists()
	if err != nil {
		return err
	}
	if exists {
		m.logger.Info("Bridge %s already exists", BridgeName)
		return nil
	}

	if err := exec.Command("ip", "link", "add", "name", BridgeName, "type", "bridge").Run(); err != nil {
		return fmt.Errorf("create bridge: %w", err)
	}

	_, ipnet, _ := net.ParseCIDR(BridgeCIDR)
	if err := exec.Command("ip", "addr", "add", ipnet.String(), "dev", BridgeName).Run(); err != nil {
		return fmt.Errorf("assign bridge IP: %w", err)
	}

	if err := exec.Command("ip", "link", "set", BridgeName, "up").Run(); err != nil {
		return fmt.Errorf("bring bridge up: %w", err)
	}

	m.logger.Info("Created bridge %s with subnet %s", BridgeName, BridgeCIDR)
	return nil
}

func (m *Manager) ConnectContainer(containerID string, pid int, portMappings []types.PortMapping) (*types.ContainerNetwork, error) {
	vethHost := fmt.Sprintf("veth-%s", containerID[:8])
	vethContainer := fmt.Sprintf("eth0-%s", containerID[:8])

	if err := exec.Command("ip", "link", "add", vethHost, "type", "veth", "peer", "name", vethContainer).Run(); err != nil {
		return nil, fmt.Errorf("create veth pair: %w", err)
	}

	if err := exec.Command("ip", "link", "set", vethHost, "master", BridgeName).Run(); err != nil {
		return nil, fmt.Errorf("attach veth to bridge: %w", err)
	}

	if err := exec.Command("ip", "link", "set", vethHost, "up").Run(); err != nil {
		return nil, fmt.Errorf("bring up host veth: %w", err)
	}

	netnsPath := fmt.Sprintf("/proc/%d/ns/net", pid)
	if err := exec.Command("ip", "link", "set", vethContainer, "netns", netnsPath).Run(); err != nil {
		return nil, fmt.Errorf("move veth to container netns: %w", err)
	}

	ipAddr := ContainerIP + containerID[len(containerID)-3:]
	if err := exec.Command("nsenter", "--net="+netnsPath, "ip", "addr", "add", ipAddr+"/16", "dev", vethContainer).Run(); err != nil {
		return nil, fmt.Errorf("assign container IP: %w", err)
	}

	if err := exec.Command("nsenter", "--net="+netnsPath, "ip", "link", "set", vethContainer, "up").Run(); err != nil {
		return nil, fmt.Errorf("bring up container veth: %w", err)
	}

	if err := exec.Command("nsenter", "--net="+netnsPath, "ip", "link", "set", "lo", "up").Run(); err != nil {
		return nil, fmt.Errorf("bring up loopback: %w", err)
	}

	for _, pm := range portMappings {
		if err := m.addPortForwarding(pm); err != nil {
			return nil, fmt.Errorf("add port forwarding: %w", err)
		}
	}

	m.logger.Info("Connected container %s to bridge, IP: %s", containerID, ipAddr)

	return &types.ContainerNetwork{
		NamespacePath: netnsPath,
		IPAddress:     ipAddr,
		Bridge:        BridgeName,
		PortMappings:  portMappings,
	}, nil
}

func (m *Manager) Disconnect(containerID string, network *types.ContainerNetwork) error {
	vethHost := fmt.Sprintf("veth-%s", containerID[:8])

	if err := exec.Command("ip", "link", "delete", vethHost).Run(); err != nil {
		m.logger.Warn("Failed to delete veth %s: %v", vethHost, err)
	}

	for _, pm := range network.PortMappings {
		m.removePortForwarding(pm)
	}

	m.logger.Info("Disconnected container %s from bridge", containerID)
	return nil
}

func (m *Manager) addPortForwarding(pm types.PortMapping) error {
	proto := strings.ToLower(pm.Protocol)
	if proto == "" {
		proto = "tcp"
	}
	return exec.Command("iptables", "-t", "nat", "-A", "PREROUTING",
		"-p", proto, "--dport", strconv.Itoa(pm.HostPort),
		"-j", "REDIRECT", "--to-port", strconv.Itoa(pm.ContainerPort),
	).Run()
}

func (m *Manager) removePortForwarding(pm types.PortMapping) error {
	proto := strings.ToLower(pm.Protocol)
	if proto == "" {
		proto = "tcp"
	}
	return exec.Command("iptables", "-t", "nat", "-D", "PREROUTING",
		"-p", proto, "--dport", strconv.Itoa(pm.HostPort),
		"-j", "REDIRECT", "--to-port", strconv.Itoa(pm.ContainerPort),
	).Run()
}

func bridgeExists() (bool, error) {
	out, err := exec.Command("ip", "link", "show", BridgeName).Output()
	if err != nil {
		return false, nil
	}
	return strings.Contains(string(out), BridgeName), nil
}

func GetAvailableIP() (string, error) {
	out, err := exec.Command("ip", "addr", "show", BridgeName).Output()
	if err != nil {
		return ContainerIP + "2", nil
	}
	lines := strings.Split(string(out), "\n")
	return fmt.Sprintf("%s%d", ContainerIP, len(lines)+2), nil
}