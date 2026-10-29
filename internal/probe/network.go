package probe

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gobox/internal/sysfs"
)

const netRoot = "/sys/class/net/"

// NetworkInterface represents a physical network interface
type NetworkInterface struct {
	Name       string // e.g., "eth0", "wlan0"
	MACAddress string // e.g., "00:1a:2b:3c:4d:5e"
	Type       string // "ethernet", "wifi"
	IsUp       bool   // operational state
	Speed      string // link speed (e.g., "1000 Mbps")
	Carrier    bool   // physical signal detected
	IPAddress  string // current IPv4 address (optional)
}

// ListNetworkInterfaces lists all physical network interfaces
func ListNetworkInterfaces() ([]NetworkInterface, error) {
	entries, err := os.ReadDir(netRoot)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", netRoot, err)
	}

	interfaces := make([]NetworkInterface, 0, 8)

	for _, entry := range entries {
		name := entry.Name()

		// Security validation
		if err := sysfs.ValidateSysfsName(name); err != nil {
			continue
		}

		// Filter virtual interfaces and loopback
		if name == "lo" ||
			strings.HasPrefix(name, "docker") ||
			strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "br-") ||
			strings.HasPrefix(name, "virbr") {
			continue
		}

		basePath := filepath.Join(netRoot, name)

		iface := NetworkInterface{
			Name: name,
		}

		// MAC address (required)
		if mac, err := sysfs.ReadFile(filepath.Join(basePath, "address")); err == nil {
			iface.MACAddress = mac
		}

		// Operational state
		if operstate, err := sysfs.ReadFileOptional(filepath.Join(basePath, "operstate")); err == nil {
			iface.IsUp = (operstate == "up")
		}

		// Carrier (cable connected)
		if carrier, err := sysfs.ReadBool(filepath.Join(basePath, "carrier")); err == nil {
			iface.Carrier = carrier
		}

		// Speed (optional, may not exist if disconnected)
		if speed, err := sysfs.ReadFileOptional(filepath.Join(basePath, "speed")); err == nil &&
			speed != "" {
			iface.Speed = speed + " Mbps"
		}

		// Determine type (WiFi vs Ethernet)
		wirelessPath := filepath.Join(basePath, "wireless")
		if _, err := os.Stat(wirelessPath); err == nil {
			iface.Type = "wifi"
		} else {
			iface.Type = "ethernet"
		}

		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}
