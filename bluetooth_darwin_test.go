package main

import (
	"fmt"
	"testing"
)

// SmokePairedDevices is a manual probe — run with: go test -run SmokePairedDevices -v
// Prints the list of paired Bluetooth devices via the native bridge.
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
