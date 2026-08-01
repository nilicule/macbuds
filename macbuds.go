package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/getlantern/systray"
)

const maxDeviceSlots = 32

// batteryPollInterval is how often the selected device's battery is re-read.
// The level only ever moves by a percent or two a minute, and the read is a
// cheap in-process call.
const batteryPollInterval = 60 * time.Second

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
	}
	return saveConfig(config)
}

// normalizeMAC strips separators and lowercases a MAC address for comparison.
func normalizeMAC(mac string) string {
	return strings.ToLower(strings.NewReplacer(":", "", "-", "").Replace(mac))
}

// batteryThresholds are the levels that trigger a low-battery notification,
// highest first.
var batteryThresholds = []int{20, 10}

// lowBatteryAlert decides whether a new battery reading warrants a notification.
//
// announced is the lowest threshold already announced this session, or 0 if
// none. alert is the threshold to announce now (0 for silence) and updated is
// the new announced value. Climbing back above a threshold re-arms it silently,
// so a recharge/drain cycle notifies again.
func lowBatteryAlert(percent, announced int) (alert, updated int) {
	crossed := 0
	for _, t := range batteryThresholds {
		if percent <= t {
			crossed = t
		}
	}
	switch {
	case crossed == 0:
		return 0, 0
	case announced == 0 || crossed < announced:
		return crossed, crossed
	case crossed > announced:
		return 0, crossed
	default:
		return 0, announced
	}
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
	systray.AddSeparator()
	mToggle := systray.AddMenuItem("Connect", "")
	systray.AddSeparator()
	mDevices := systray.AddMenuItem("Devices", "")
	deviceSlots := make([]*systray.MenuItem, maxDeviceSlots)
	for i := range deviceSlots {
		deviceSlots[i] = mDevices.AddSubMenuItem("", "")
		deviceSlots[i].Hide()
	}
	mDevicesSep := mDevices.AddSubMenuItem("─────────────", "")
	mDevicesSep.Disable()
	mRefreshDevices := mDevices.AddSubMenuItem("Refresh Devices", "")
	mClearDevice := systray.AddMenuItem("Clear Selected Device", "")
	systray.AddSeparator()

	// Notifications submenu
	mNotifications := systray.AddMenuItem("Notifications", "")
	mNotifyConnect := mNotifications.AddSubMenuItem("Notify on connect", "")
	mNotifyDisconnect := mNotifications.AddSubMenuItem("Notify on disconnect", "")
	mNotifyLowBattery := mNotifications.AddSubMenuItem("Notify on low battery", "")
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

	// prevConnected is written by the event consumer and (via seedState) by
	// the menu-click goroutine. Concurrent menu clicks and inbound events are
	// effectively never simultaneous at human pace, so this stays mutex-free.
	prevConnected := false

	// battery holds the last reading for the selected device, or -1 when
	// unknown. announcedBattery is the lowest low-battery threshold already
	// notified for the current charge cycle. Both share prevConnected's
	// synchronisation story: the only writers are the event consumer and the
	// menu-click loop.
	battery := -1
	announcedBattery := 0

	deviceLabel := func() string {
		if config.DeviceName != "" {
			return config.DeviceName
		}
		return config.MacAddress
	}

	updateUIForState := func(connected bool) {
		if config.MacAddress == "" {
			mStatus.SetTitle("Status: No device selected")
			mToggle.Disable()
			mClearDevice.Disable()
			systray.SetIcon(iconNoneBytes)
			systray.SetTitle("")
			return
		}
		if connected {
			status := deviceLabel()
			if battery >= 0 {
				status = fmt.Sprintf("%s · %d%%", status, battery)
				systray.SetTitle(fmt.Sprintf("%d%%", battery))
			} else {
				systray.SetTitle("")
			}
			mStatus.SetTitle(fmt.Sprintf("Connected: %s", status))
			mToggle.SetTitle("Disconnect")
			systray.SetIcon(iconConnectedBytes)
		} else {
			mStatus.SetTitle(fmt.Sprintf("%s · Disconnected", deviceLabel()))
			mToggle.SetTitle("Connect")
			systray.SetIcon(iconDisconnectedBytes)
			systray.SetTitle("")
		}
		mToggle.Enable()
		mClearDevice.Enable()
	}

	// refreshBattery re-reads the selected device's level and fires a
	// notification when the reading crosses a low-battery threshold. A
	// just-connected device often hasn't reported yet — it stays unknown until
	// a later poll picks it up.
	refreshBattery := func(connected bool) {
		if !connected || config.MacAddress == "" {
			battery = -1
			announcedBattery = 0
			return
		}
		b, err := DeviceBattery(config.MacAddress)
		if err != nil || !b.Available {
			battery = -1
			return
		}
		battery = b.Percent
		alert, updated := lowBatteryAlert(battery, announcedBattery)
		announcedBattery = updated
		if alert > 0 && config.NotifyLowBattery {
			sendNotification("MacBuds", fmt.Sprintf("%s battery at %d%%", deviceLabel(), battery))
		}
	}

	seedState := func() {
		StopMonitoring()
		if config.MacAddress == "" {
			prevConnected = false
			refreshBattery(false)
			updateUIForState(false)
			return
		}
		connected, _ := IsConnected(config.MacAddress)
		prevConnected = connected
		refreshBattery(connected)
		updateUIForState(connected)
		_ = StartMonitoring(config.MacAddress)
	}

	// Devices submenu state. `paired` is read by both the device-click
	// goroutines and the refresh handler; protected by pairedMu.
	var pairedMu sync.Mutex
	var paired []BluetoothDevice

	refreshDeviceMenu := func() {
		newPaired, err := PairedDevices()
		if err != nil {
			mStatus.SetTitle(fmt.Sprintf("Error: %v", err))
			return
		}
		pairedMu.Lock()
		defer pairedMu.Unlock()
		paired = newPaired
		for i, slot := range deviceSlots {
			if i < len(paired) {
				slot.SetTitle(fmt.Sprintf("%s (%s)", paired[i].Name, paired[i].Address))
				slot.Show()
				if paired[i].Address == config.MacAddress {
					slot.Check()
				} else {
					slot.Uncheck()
				}
			} else {
				slot.Uncheck()
				slot.Hide()
			}
		}
	}

	selectDeviceByIndex := func(idx int) {
		pairedMu.Lock()
		if idx >= len(paired) {
			pairedMu.Unlock()
			return
		}
		d := paired[idx]
		pairedMu.Unlock()

		config.MacAddress = d.Address
		config.DeviceName = d.Name
		if err := saveConfig(config); err != nil {
			mStatus.SetTitle(fmt.Sprintf("Error saving config: %v", err))
		}
		refreshDeviceMenu()
		seedState()
	}

	// One click goroutine per device slot — forwards to selectDeviceByIndex.
	for i, slot := range deviceSlots {
		idx := i
		s := slot
		go func() {
			for range s.ClickedCh {
				selectDeviceByIndex(idx)
			}
		}()
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
			switch ev.Kind {
			case BluetoothConnected:
				refreshBattery(true)
				updateUIForState(true)
				if config.NotifyConnect && !prevConnected {
					sendNotification("MacBuds", fmt.Sprintf("%s connected", deviceLabel()))
				}
				prevConnected = true
			case BluetoothDisconnected:
				refreshBattery(false)
				updateUIForState(false)
				if config.NotifyDisconnect && prevConnected {
					sendNotification("MacBuds", fmt.Sprintf("%s disconnected", deviceLabel()))
				}
				prevConnected = false
			case BluetoothConnectFailed:
				mStatus.SetTitle("Error: connect failed")
			}
		}
	}()

	refreshDeviceMenu()
	seedState()

	// Handle menu clicks. The battery poll rides this loop rather than its own
	// goroutine: it fires on a schedule rather than at human pace, so a third
	// concurrent writer of the UI state would be a real race, not a theoretical
	// one.
	go func() {
		batteryTicker := time.NewTicker(batteryPollInterval)
		defer batteryTicker.Stop()
		for {
			select {
			case <-batteryTicker.C:
				if config.MacAddress == "" {
					continue
				}
				connected, _ := IsConnected(config.MacAddress)
				refreshBattery(connected)
				updateUIForState(connected)

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

			case <-mRefreshDevices.ClickedCh:
				refreshDeviceMenu()

			case <-mClearDevice.ClickedCh:
				if err := clearConfig(); err != nil {
					mStatus.SetTitle(fmt.Sprintf("Error clearing config: %v", err))
				} else {
					config.MacAddress = ""
					config.DeviceName = ""
					refreshDeviceMenu()
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
