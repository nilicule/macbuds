package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"
)

type BluetoothDevice struct {
	Address string
	Name    string
}

type Config struct {
	MacAddress string `json:"mac_address"`
	DeviceName string `json:"device_name"`
}

func getBlueutilPath() string {
	// First try the bundled path (for production)
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)
	bundledPath := filepath.Join(execDir, "blueutil")
	if _, err := os.Stat(bundledPath); err == nil {
		return bundledPath
	}

	// If not found, try system path (for development)
	systemPath, err := exec.LookPath("blueutil")
	if err == nil {
		return systemPath
	}

	// Fall back to bundled path even if it doesn't exist
	return bundledPath
}

func getPairedDevices() ([]BluetoothDevice, error) {
	cmd := exec.Command(getBlueutilPath(), "--paired")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute blueutil: %v", err)
	}

	var devices []BluetoothDevice
	lines := strings.Split(string(output), "\n")

	// Add debug logging
	fmt.Printf("Found %d lines in blueutil output\n", len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fmt.Printf("Processing line: %s\n", line)

		// Extract address - it's always at the start after "address: "
		if !strings.Contains(line, "address: ") {
			continue
		}
		parts := strings.SplitN(line[9:], ",", 2) // Skip "address: "
		if len(parts) < 2 {
			continue
		}
		address := strings.TrimSpace(parts[0])

		// Extract name - it's between quotes after "name: "
		nameIdx := strings.Index(line, "name: \"")
		if nameIdx == -1 {
			continue
		}
		nameStart := nameIdx + 7 // len("name: \"")
		nameEnd := strings.Index(line[nameStart:], "\"")
		if nameEnd == -1 {
			continue
		}
		name := line[nameStart : nameStart+nameEnd]

		// Skip empty or malformed entries
		if address == "" || name == "" {
			continue
		}

		fmt.Printf("Found device: %s (%s)\n", name, address)

		devices = append(devices, BluetoothDevice{
			Address: address,
			Name:    name,
		})
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no paired devices found")
	}

	return devices, nil
}

func loadConfig() (*Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "bluetooth-menubar", "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func saveConfig(config *Config) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	appConfigDir := filepath.Join(configDir, "bluetooth-menubar")
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(appConfigDir, "config.json")
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func clearConfig() error {
	config := &Config{}
	return saveConfig(config)
}

func isConnected(macAddress string) (bool, error) {
	if macAddress == "" {
		return false, nil
	}
	cmd := exec.Command(getBlueutilPath(), "--is-connected", macAddress)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(output)) == "1", nil
}

func connectBluetooth(macAddress string) error {
	cmd := exec.Command(getBlueutilPath(), "--connect", macAddress)
	return cmd.Run()
}

func disconnectBluetooth(macAddress string) error {
	cmd := exec.Command(getBlueutilPath(), "--disconnect", macAddress)
	return cmd.Run()
}

func getLaunchAgentPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Library", "LaunchAgents", "org.rc6.macbuds.plist")
}

func getExecutablePath() string {
	exe, _ := os.Executable()
	return exe
}

func isLaunchAtLoginEnabled() bool {
	_, err := os.Stat(getLaunchAgentPath())
	return err == nil
}

func enableLaunchAtLogin() error {
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>org.rc6.macbuds</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>`, getExecutablePath())

	return os.WriteFile(getLaunchAgentPath(), []byte(plistContent), 0644)
}

func disableLaunchAtLogin() error {
	err := os.Remove(getLaunchAgentPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func onReady() {
	systray.SetTitle("BT •")
	systray.SetTooltip("Bluetooth Earbuds Controller")

	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		systray.Quit()
		return
	}

	// Menu items
	mStatus := systray.AddMenuItem("Status: Unknown", "Current connection status")
	mStatus.Disable()
	systray.AddSeparator()
	mToggle := systray.AddMenuItem("Connect", "Toggle connection")
	systray.AddSeparator()
	mSelectDevice := systray.AddMenuItem("Select Device", "Choose Bluetooth device")
	mClearDevice := systray.AddMenuItem("Clear Selected Device", "Clear currently selected device")
	systray.AddSeparator()
	mLaunchAtLogin := systray.AddMenuItem("Launch at Login", "Toggle launch at login")
	if isLaunchAtLoginEnabled() {
		mLaunchAtLogin.Check()
	}
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Update status
	go func() {
		for {
			connected, err := isConnected(config.MacAddress)
			if err != nil {
				mStatus.SetTitle("Status: Error")
				continue
			}

			if config.MacAddress == "" {
				mStatus.SetTitle("Status: No device selected")
				mToggle.Disable()
				mClearDevice.Disable()
				systray.SetTitle("BT •")
			} else {
				deviceInfo := config.DeviceName
				if deviceInfo == "" {
					deviceInfo = config.MacAddress
				}

				if connected {
					mStatus.SetTitle(fmt.Sprintf("Status: Connected to %s", deviceInfo))
					mToggle.SetTitle("Disconnect")
					mToggle.Enable()
					mClearDevice.Enable()
					systray.SetTitle("BT ✓")
				} else {
					mStatus.SetTitle(fmt.Sprintf("Status: Disconnected from %s", deviceInfo))
					mToggle.SetTitle("Connect")
					mToggle.Enable()
					mClearDevice.Enable()
					systray.SetTitle("BT ×")
				}
			}

			time.Sleep(2 * time.Second)
		}
	}()

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-mToggle.ClickedCh:
				if config.MacAddress == "" {
					continue
				}
				connected, _ := isConnected(config.MacAddress)
				if connected {
					if err := disconnectBluetooth(config.MacAddress); err != nil {
						mStatus.SetTitle(fmt.Sprintf("Error: %v", err))
					}
				} else {
					if err := connectBluetooth(config.MacAddress); err != nil {
						mStatus.SetTitle(fmt.Sprintf("Error: %v", err))
					}
				}

			case <-mSelectDevice.ClickedCh:
				fmt.Println("Select Device clicked")
				devices, err := getPairedDevices()
				if err != nil {
					fmt.Printf("Error getting paired devices: %v\n", err)
					mStatus.SetTitle(fmt.Sprintf("Error: %v", err))
					continue
				}

				fmt.Printf("Found %d devices\n", len(devices))

				options := make([]string, len(devices))
				deviceMap := make(map[string]BluetoothDevice)
				for i, device := range devices {
					options[i] = fmt.Sprintf("%s (%s)", device.Name, device.Address)
					deviceMap[options[i]] = device
					fmt.Printf("Added option: %s\n", options[i])
				}

				fmt.Println("Showing selection dialog")
				selected, err := zenity.List(
					"Select a Bluetooth device to control:",
					options,
					zenity.Title("Select Bluetooth Device"),
					zenity.Width(400),
					zenity.Height(300),
				)

				fmt.Printf("Dialog result: %v, err: %v\n", selected, err)

				if err != nil {
					if err == zenity.ErrCanceled {
						fmt.Println("User canceled selection")
						continue
					}
					fmt.Printf("Error with selection dialog: %v\n", err)
					mStatus.SetTitle(fmt.Sprintf("Error: %v", err))
					continue
				}

				device := deviceMap[selected]
				fmt.Printf("Selected device: %s (%s)\n", device.Name, device.Address)

				config.MacAddress = device.Address
				config.DeviceName = device.Name
				if err := saveConfig(config); err != nil {
					fmt.Printf("Error saving config: %v\n", err)
					mStatus.SetTitle(fmt.Sprintf("Error saving config: %v", err))
				} else {
					fmt.Println("Config saved successfully")
					mStatus.SetTitle(fmt.Sprintf("Device selected: %s", device.Name))
				}

			case <-mClearDevice.ClickedCh:
				if err := clearConfig(); err != nil {
					mStatus.SetTitle(fmt.Sprintf("Error clearing config: %v", err))
				} else {
					config.MacAddress = ""
					config.DeviceName = ""
				}

			case <-mLaunchAtLogin.ClickedCh:
				if isLaunchAtLoginEnabled() {
					if err := disableLaunchAtLogin(); err != nil {
						mStatus.SetTitle(fmt.Sprintf("Error: %v", err))
					} else {
						mLaunchAtLogin.Uncheck()
					}
				} else {
					if err := enableLaunchAtLogin(); err != nil {
						mStatus.SetTitle(fmt.Sprintf("Error: %v", err))
					} else {
						mLaunchAtLogin.Check()
					}
				}

			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	// Cleanup code here
}

func main() {
	systray.Run(onReady, onExit)
}
