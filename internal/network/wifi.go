package network

import (
	"os/exec"
	"runtime"
	"strings"
)

// GetWiFiSSID returns the current Wi-Fi network name (SSID).
// Returns "unknown" if it cannot be determined.
func GetWiFiSSID() string {
	switch runtime.GOOS {
	case "darwin":
		// Primary: ipconfig getsummary (works on macOS Sonoma+)
		out, err := exec.Command("ipconfig", "getsummary", "en0").Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "SSID : ") {
					ssid := strings.TrimPrefix(trimmed, "SSID : ")
					if ssid != "" {
						return ssid
					}
				}
			}
		}
		// Fallback: networksetup (older macOS)
		out, err = exec.Command("networksetup", "-getairportnetwork", "en0").Output()
		if err == nil {
			line := strings.TrimSpace(string(out))
			if strings.HasPrefix(line, "Current Wi-Fi Network: ") {
				return strings.TrimPrefix(line, "Current Wi-Fi Network: ")
			}
		}
		return "unknown"

	case "linux":
		out, err := exec.Command("iwgetid", "-r").Output()
		if err != nil {
			return "unknown"
		}
		ssid := strings.TrimSpace(string(out))
		if ssid == "" {
			return "unknown"
		}
		return ssid

	case "windows":
		out, err := exec.Command("netsh", "wlan", "show", "interfaces").Output()
		if err != nil {
			return "unknown"
		}
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "SSID") && !strings.HasPrefix(line, "BSSID") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
		return "unknown"

	default:
		return "unknown"
	}
}
