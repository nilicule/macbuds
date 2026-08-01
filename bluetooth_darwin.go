package main

/*
#cgo LDFLAGS: -framework IOBluetooth
#include <stdlib.h>
#include "bluetooth_darwin.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type BluetoothEventKind int

const (
	BluetoothConnected     BluetoothEventKind = C.BT_EVENT_CONNECTED
	BluetoothDisconnected  BluetoothEventKind = C.BT_EVENT_DISCONNECTED
	BluetoothConnectFailed BluetoothEventKind = C.BT_EVENT_CONNECT_FAILED
)

type BluetoothEvent struct {
	Kind BluetoothEventKind
	MAC  string
}

var btEvents = make(chan BluetoothEvent, 16)

func BluetoothEvents() <-chan BluetoothEvent { return btEvents }

//export goOnBluetoothEvent
func goOnBluetoothEvent(kind C.int, mac *C.char) {
	ev := BluetoothEvent{
		Kind: BluetoothEventKind(kind),
		MAC:  C.GoString(mac),
	}
	select {
	case btEvents <- ev:
	default:
	}
}

func PairedDevices() ([]BluetoothDevice, error) {
	var buf [C.BT_MAX_DEVICES]C.bt_device_t
	n := int(C.bt_paired_devices(&buf[0], C.int(len(buf))))
	if n < 0 {
		return nil, fmt.Errorf("bt_paired_devices failed")
	}
	out := make([]BluetoothDevice, 0, n)
	seen := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		addr := C.GoString(&buf[i].mac[0])
		if seen[addr] {
			continue
		}
		seen[addr] = true
		out = append(out, BluetoothDevice{
			Address: addr,
			Name:    C.GoString(&buf[i].name[0]),
		})
	}
	return out, nil
}

func IsConnected(mac string) (bool, error) {
	cmac := C.CString(mac)
	defer C.free(unsafe.Pointer(cmac))
	r := int(C.bt_is_connected(cmac))
	if r < 0 {
		return false, fmt.Errorf("bt_is_connected failed")
	}
	return r == 1, nil
}

func Connect(mac string) error {
	cmac := C.CString(mac)
	defer C.free(unsafe.Pointer(cmac))
	if int(C.bt_connect(cmac)) != 0 {
		return fmt.Errorf("bt_connect failed")
	}
	return nil
}

func Disconnect(mac string) error {
	cmac := C.CString(mac)
	defer C.free(unsafe.Pointer(cmac))
	if int(C.bt_disconnect(cmac)) != 0 {
		return fmt.Errorf("bt_disconnect failed")
	}
	return nil
}

// Battery is a device's charge level. Available is false when the device
// doesn't report one — disconnected, or simply not a battery-backed device.
type Battery struct {
	Percent   int
	Available bool
}

func DeviceBattery(mac string) (Battery, error) {
	cmac := C.CString(mac)
	defer C.free(unsafe.Pointer(cmac))
	var b C.bt_battery_t
	if int(C.bt_battery(cmac, &b)) != 0 {
		return Battery{}, fmt.Errorf("bt_battery failed")
	}
	if int(b.single) < 0 {
		return Battery{}, nil
	}
	return Battery{Percent: int(b.single), Available: true}, nil
}

func StartMonitoring(mac string) error {
	cmac := C.CString(mac)
	defer C.free(unsafe.Pointer(cmac))
	if int(C.bt_start_monitoring(cmac)) != 0 {
		return fmt.Errorf("bt_start_monitoring failed")
	}
	return nil
}

func StopMonitoring() {
	C.bt_stop_monitoring()
}
