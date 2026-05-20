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
	for i := 0; i < n; i++ {
		out = append(out, BluetoothDevice{
			Address: C.GoString(&buf[i].mac[0]),
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

// PickDevice shows a native macOS picker for the supplied devices and returns
// the selected one. picked is false when the user cancels.
func PickDevice(devices []BluetoothDevice) (BluetoothDevice, bool, error) {
	if len(devices) == 0 {
		return BluetoothDevice{}, false, nil
	}

	cDevices := make([]C.bt_device_t, len(devices))
	for i, d := range devices {
		mac := []byte(d.Address)
		if len(mac) > len(cDevices[i].mac)-1 {
			mac = mac[:len(cDevices[i].mac)-1]
		}
		for j, b := range mac {
			cDevices[i].mac[j] = C.char(b)
		}
		cDevices[i].mac[len(mac)] = 0

		name := []byte(d.Name)
		if len(name) > len(cDevices[i].name)-1 {
			name = name[:len(cDevices[i].name)-1]
		}
		for j, b := range name {
			cDevices[i].name[j] = C.char(b)
		}
		cDevices[i].name[len(name)] = 0
	}

	var outMAC [32]C.char
	r := int(C.bt_pick_device(&cDevices[0], C.int(len(cDevices)),
		&outMAC[0], C.int(len(outMAC))))

	switch r {
	case 0:
		mac := C.GoString(&outMAC[0])
		for _, d := range devices {
			if d.Address == mac {
				return d, true, nil
			}
		}
		return BluetoothDevice{}, false, fmt.Errorf("picked device not found: %s", mac)
	case 1:
		return BluetoothDevice{}, false, nil
	default:
		return BluetoothDevice{}, false, fmt.Errorf("bt_pick_device failed")
	}
}
