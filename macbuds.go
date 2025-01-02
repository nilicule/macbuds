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

type Config struct {
	MacAddress string `json:"mac_address"`
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

func isConnected(macAddress string) (bool, error) {
	if macAddress == "" {
		return false, nil
	}
	cmd := exec.Command("blueutil", "--is-connected", macAddress)
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(string(output)) == "1", nil
}

func connectBluetooth(macAddress string) error {
	cmd := exec.Command("blueutil", "--connect", macAddress)
	return cmd.Run()
}

func disconnectBluetooth(macAddress string) error {
	cmd := exec.Command("blueutil", "--disconnect", macAddress)
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
	mConfig := systray.AddMenuItem("Configure MAC Address", "Set device MAC address")
	systray.AddSeparator()
	mLaunchAtLogin := systray.AddMenuItem("Launch at Login", "Toggle launch at login")
	if isLaunchAtLoginEnabled() {
		mLaunchAtLogin.Check()
	}
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Update status and menu items periodically
	go func() {
		for {
			connected, err := isConnected(config.MacAddress)
			if err != nil {
				mStatus.SetTitle("Status: Error")
				continue
			}

			if config.MacAddress == "" {
				mStatus.SetTitle("Status: No device configured")
				mToggle.Disable()
				systray.SetTitle("BT •")
			} else if connected {
				mStatus.SetTitle("Status: Connected")
				mToggle.SetTitle("Disconnect")
				mToggle.Enable()
				systray.SetTitle("BT ✓")
			} else {
				mStatus.SetTitle("Status: Disconnected")
				mToggle.SetTitle("Connect")
				mToggle.Enable()
				systray.SetTitle("BT ×")
			}

			time.Sleep(2 * time.Second)
		}
	}()

	// Handle menu item clicks
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

			case <-mConfig.ClickedCh:
				currentMac := config.MacAddress
				if currentMac == "" {
					currentMac = "Enter MAC address"
				}
				newMac, err := zenity.Entry(
					"Enter Bluetooth MAC Address:",
					zenity.Title("Configure Device"),
					zenity.EntryText(currentMac),
				)
				if err != nil {
					if err != zenity.ErrCanceled {
						mStatus.SetTitle(fmt.Sprintf("Error: %v", err))
					}
					continue
				}

				newMac = strings.TrimSpace(newMac)
				if newMac != config.MacAddress {
					config.MacAddress = newMac
					if err := saveConfig(config); err != nil {
						mStatus.SetTitle(fmt.Sprintf("Error saving config: %v", err))
					}
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
