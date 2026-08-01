package main

import (
	"fmt"
	"testing"
)

// Manual probes — run with: go test -run TestSmoke -v
// They exercise the native bridge against real paired devices.

func TestSmokePairedDevices(t *testing.T) {
	devices, err := PairedDevices()
	if err != nil {
		t.Fatalf("PairedDevices failed: %v", err)
	}
	if len(devices) == 0 {
		t.Log("no paired devices found")
	}
	for _, d := range devices {
		fmt.Printf("  %s  %s\n", d.Address, d.Name)
	}
}

func TestSmokeBattery(t *testing.T) {
	devices, err := PairedDevices()
	if err != nil {
		t.Fatalf("PairedDevices failed: %v", err)
	}
	for _, d := range devices {
		connected, _ := IsConnected(d.Address)
		b, err := DeviceBattery(d.Address)
		if err != nil {
			t.Errorf("DeviceBattery(%s) failed: %v", d.Address, err)
			continue
		}
		level := "unknown"
		if b.Available {
			level = fmt.Sprintf("%d%%", b.Percent)
		}
		fmt.Printf("  %s  %-26s connected=%-5v battery=%s\n", d.Address, d.Name, connected, level)
	}
}

func TestSmokeIsConnected(t *testing.T) {
	devices, err := PairedDevices()
	if err != nil {
		t.Fatalf("PairedDevices failed: %v", err)
	}
	for _, d := range devices {
		connected, err := IsConnected(d.Address)
		if err != nil {
			t.Errorf("IsConnected(%s) failed: %v", d.Address, err)
			continue
		}
		fmt.Printf("  %s  %s  connected=%v\n", d.Address, d.Name, connected)
	}
}
