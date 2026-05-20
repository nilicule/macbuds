package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/getlantern/systray"
	"github.com/ncruces/zenity"
)

//go:embed assets/icon_none.png
var iconNoneBytes []byte

//go:embed assets/icon_connected.png
var iconConnectedBytes []byte

//go:embed assets/icon_disconnected.png
var iconDisconnectedBytes []byte

type BluetoothDevice struct {
	Address string
	Name    string
}

type Config struct {
	MacAddress              string `json:"mac_address"`
	DeviceName              string `json:"device_name"`
	NotifyConnect           bool   `json:"notify_connect"`
	NotifyDisconnect        bool   `json:"notify_disconnect"`
	NotifyLowBattery        bool   `json:"notify_low_battery"`
	LowBatteryThreshold     int    `json:"low_battery_threshold"`
	NotificationsConfigured bool   `json:"notifications_configured"`
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
			return &Config{
				NotifyConnect:           true,
				NotifyDisconnect:        true,
				NotifyLowBattery:        true,
				LowBatteryThreshold:     20,
				NotificationsConfigured: true,
			}, nil
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Apply defaults for notification fields when upgrading from older config
	if !config.NotificationsConfigured {
		config.NotifyConnect = true
		config.NotifyDisconnect = true
		config.NotifyLowBattery = true
		config.NotificationsConfigured = true
	}
	if config.LowBatteryThreshold == 0 {
		config.LowBatteryThreshold = 20
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
	config := &Config{
		NotificationsConfigured: true,
		NotifyConnect:           true,
		NotifyDisconnect:        true,
		NotifyLowBattery:        true,
		LowBatteryThreshold:     20,
	}
	return saveConfig(config)
}

// normalizeMAC strips separators and lowercases a MAC address for comparison.
func normalizeMAC(mac string) string {
	return strings.ToLower(strings.NewReplacer(":", "", "-", "").Replace(mac))
}

func getBatteryLevel(macAddress string) (int, error) {
	out, err := exec.Command("system_profiler", "SPBluetoothDataType").Output()
	if err != nil {
		return -1, err
	}

	targetMAC := normalizeMAC(macAddress)
	inDevice := false

	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "Address:") {
			addr := strings.TrimSpace(strings.TrimPrefix(trimmed, "Address:"))
			inDevice = normalizeMAC(addr) == targetMAC
		}

		if inDevice && strings.HasPrefix(trimmed, "Battery Level:") {
			val := strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(trimmed, "Battery Level:")), "%")
			if n, err := strconv.Atoi(val); err == nil {
				return n, nil
			}
		}
	}
	return -1, nil
}

func sendNotification(title, message string) {
	safeMessage := strings.ReplaceAll(message, `"`, `\"`)
	safeTitle := strings.ReplaceAll(title, `"`, `\"`)
	script := fmt.Sprintf(`display notification "%s" with title "%s"`, safeMessage, safeTitle)
	exec.Command("osascript", "-e", script).Run() //nolint:errcheck
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
	systray.SetIcon(iconNoneBytes)
	systray.SetTitle("")
	systray.SetTooltip("MacBuds - Bluetooth Controller")

	config, err := loadConfig()
	if err != nil {
		systray.Quit()
		return
	}

	// Menu items
	mStatus := systray.AddMenuItem("Status: Unknown", "")
	mStatus.Disable()
	mBattery := systray.AddMenuItem("Battery: –", "")
	mBattery.Disable()
	systray.AddSeparator()
	mToggle := systray.AddMenuItem("Connect", "")
	systray.AddSeparator()
	mSelectDevice := systray.AddMenuItem("Select Device", "")
	mClearDevice := systray.AddMenuItem("Clear Selected Device", "")
	systray.AddSeparator()

	// Notifications submenu
	mNotifications := systray.AddMenuItem("Notifications", "")
	mNotifyConnect := mNotifications.AddSubMenuItem("Notify on connect", "")
	mNotifyDisconnect := mNotifications.AddSubMenuItem("Notify on disconnect", "")
	mNotifyLowBattery := mNotifications.AddSubMenuItem(
		fmt.Sprintf("Low battery warning (< %d%%)", config.LowBatteryThreshold), "")
	if config.NotifyConnect {
		mNotifyConnect.Check()
	}
	if config.NotifyDisconnect {
		mNotifyDisconnect.Check()
	}
	if config.NotifyLowBattery {
		mNotifyLowBattery.Check()
	}

	systray.AddSeparator()
	mLaunchAtLogin := systray.AddMenuItem("Launch at Login", "")
	if isLaunchAtLoginEnabled() {
		mLaunchAtLogin.Check()
	}
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "")

	// Shared state — written by the event consumer (prevConnected) and the
	// battery ticker (prevBattery). seedState also writes prevConnected from
	// the menu-click goroutine; concurrent menu clicks and inbound events are
	// effectively never simultaneous (human-paced), so this stays mutex-free.
	prevConnected := false
	prevBattery := -1

	updateUIForState := func(connected bool) {
		if config.MacAddress == "" {
			mStatus.SetTitle("Status: No device selected")
			mBattery.SetTitle("Battery: –")
			mToggle.Disable()
			mClearDevice.Disable()
			systray.SetIcon(iconNoneBytes)
			return
		}
		deviceInfo := config.DeviceName
		if deviceInfo == "" {
			deviceInfo = config.MacAddress
		}
		if connected {
			mStatus.SetTitle(fmt.Sprintf("Connected: %s", deviceInfo))
			mToggle.SetTitle("Disconnect")
			systray.SetIcon(iconConnectedBytes)
		} else {
			mStatus.SetTitle(fmt.Sprintf("%s · Disconnected", deviceInfo))
			mToggle.SetTitle("Connect")
			systray.SetIcon(iconDisconnectedBytes)
		}
		mToggle.Enable()
		mClearDevice.Enable()
	}

	seedState := func() {
		StopMonitoring()
		if config.MacAddress == "" {
			prevConnected = false
			updateUIForState(false)
			return
		}
		connected, _ := IsConnected(config.MacAddress)
		prevConnected = connected
		updateUIForState(connected)
		_ = StartMonitoring(config.MacAddress)
	}

	// Event consumer
	go func() {
		for ev := range BluetoothEvents() {
			if config.MacAddress == "" {
				continue
			}
			if normalizeMAC(ev.MAC) != normalizeMAC(config.MacAddress) {
				continue
			}
			deviceInfo := config.DeviceName
			if deviceInfo == "" {
				deviceInfo = config.MacAddress
			}
			switch ev.Kind {
			case BluetoothConnected:
				updateUIForState(true)
				if config.NotifyConnect && !prevConnected {
					sendNotification("MacBuds", fmt.Sprintf("%s connected", deviceInfo))
				}
				prevConnected = true
			case BluetoothDisconnected:
				updateUIForState(false)
				if config.NotifyDisconnect && prevConnected {
					sendNotification("MacBuds", fmt.Sprintf("%s disconnected", deviceInfo))
				}
				prevConnected = false
			case BluetoothConnectFailed:
				mStatus.SetTitle("Error: connect failed")
			}
		}
	}()

	// Battery ticker
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		firstRun := true
		updateBattery := func() {
			if config.MacAddress == "" {
				mBattery.SetTitle("Battery: –")
				return
			}
			level, _ := getBatteryLevel(config.MacAddress)
			if level < 0 {
				mBattery.SetTitle("Battery: –")
				prevBattery = -1
				return
			}
			mBattery.SetTitle(fmt.Sprintf("Battery: %d%%", level))
			if !firstRun && config.NotifyLowBattery &&
				level <= config.LowBatteryThreshold &&
				(prevBattery > config.LowBatteryThreshold || prevBattery < 0) {
				sendNotification("MacBuds",
					fmt.Sprintf("%s battery is low (%d%%)", config.DeviceName, level))
			}
			prevBattery = level
			firstRun = false
		}
		updateBattery()
		for range ticker.C {
			updateBattery()
		}
	}()

	seedState()

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-mToggle.ClickedCh:
				if config.MacAddress == "" {
					continue
				}
				connected, _ := IsConnected(config.MacAddress)
				if connected {
					if err := Disconnect(config.MacAddress); err != nil {
						mStatus.SetTitle(fmt.Sprintf("Error: %v", err))
					}
				} else {
					if err := Connect(config.MacAddress); err != nil {
						mStatus.SetTitle(fmt.Sprintf("Error: %v", err))
					}
				}

			case <-mSelectDevice.ClickedCh:
				devices, err := PairedDevices()
				if err != nil {
					mStatus.SetTitle(fmt.Sprintf("Error: %v", err))
					continue
				}

				options := make([]string, len(devices))
				deviceMap := make(map[string]BluetoothDevice)
				for i, device := range devices {
					options[i] = fmt.Sprintf("%s (%s)", device.Name, device.Address)
					deviceMap[options[i]] = device
				}

				selected, err := zenity.List(
					"Select a Bluetooth device to control:",
					options,
					zenity.Title("Select Bluetooth Device"),
					zenity.Width(400),
					zenity.Height(300),
				)
				if err != nil {
					if err != zenity.ErrCanceled {
						mStatus.SetTitle(fmt.Sprintf("Error: %v", err))
					}
					continue
				}

				device := deviceMap[selected]
				config.MacAddress = device.Address
				config.DeviceName = device.Name
				if err := saveConfig(config); err != nil {
					mStatus.SetTitle(fmt.Sprintf("Error saving config: %v", err))
				}
				seedState()

			case <-mClearDevice.ClickedCh:
				if err := clearConfig(); err != nil {
					mStatus.SetTitle(fmt.Sprintf("Error clearing config: %v", err))
				} else {
					config.MacAddress = ""
					config.DeviceName = ""
					seedState()
				}

			case <-mNotifyConnect.ClickedCh:
				config.NotifyConnect = !config.NotifyConnect
				if config.NotifyConnect {
					mNotifyConnect.Check()
				} else {
					mNotifyConnect.Uncheck()
				}
				saveConfig(config) //nolint:errcheck

			case <-mNotifyDisconnect.ClickedCh:
				config.NotifyDisconnect = !config.NotifyDisconnect
				if config.NotifyDisconnect {
					mNotifyDisconnect.Check()
				} else {
					mNotifyDisconnect.Uncheck()
				}
				saveConfig(config) //nolint:errcheck

			case <-mNotifyLowBattery.ClickedCh:
				config.NotifyLowBattery = !config.NotifyLowBattery
				if config.NotifyLowBattery {
					mNotifyLowBattery.Check()
				} else {
					mNotifyLowBattery.Uncheck()
				}
				saveConfig(config) //nolint:errcheck

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
