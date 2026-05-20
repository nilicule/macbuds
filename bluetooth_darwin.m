#import <Foundation/Foundation.h>
#import <IOBluetooth/IOBluetooth.h>
#import "bluetooth_darwin.h"

// Forward declaration of the exported Go callback (defined in bluetooth_darwin.go).
extern void goOnBluetoothEvent(int kind, const char *mac);

static IOBluetoothDevice *deviceForMAC(const char *mac) {
    if (mac == NULL) return nil;
    NSString *s = [NSString stringWithUTF8String:mac];
    return [IOBluetoothDevice deviceWithAddressString:s];
}

int bt_paired_devices(bt_device_t *out, int max_count) {
    @autoreleasepool {
        NSArray<IOBluetoothDevice *> *devices = [IOBluetoothDevice pairedDevices];
        if (devices == nil) {
            return 0;
        }
        int count = 0;
        for (IOBluetoothDevice *d in devices) {
            if (count >= max_count) break;
            NSString *addr = [d addressString] ?: @"";
            NSString *name = [d name] ?: @"";
            strncpy(out[count].mac, [addr UTF8String], sizeof(out[count].mac) - 1);
            out[count].mac[sizeof(out[count].mac) - 1] = '\0';
            strncpy(out[count].name, [name UTF8String], sizeof(out[count].name) - 1);
            out[count].name[sizeof(out[count].name) - 1] = '\0';
            count++;
        }
        return count;
    }
}

int bt_is_connected(const char *mac) {
    @autoreleasepool {
        IOBluetoothDevice *d = deviceForMAC(mac);
        if (d == nil) return -1;
        return [d isConnected] ? 1 : 0;
    }
}

int bt_connect(const char *mac) {
    @autoreleasepool {
        IOBluetoothDevice *d = deviceForMAC(mac);
        if (d == nil) return -1;
        // openConnection is synchronous — run on a background queue so the
        // Go caller and systray main loop stay responsive. On success the
        // registered connect notification will fire; only emit a synthetic
        // failure event here.
        char *mac_copy = strdup(mac);
        dispatch_async(dispatch_get_global_queue(QOS_CLASS_USER_INITIATED, 0), ^{
            IOReturn r = [d openConnection];
            if (r != kIOReturnSuccess) {
                goOnBluetoothEvent(BT_EVENT_CONNECT_FAILED, mac_copy);
            }
            free(mac_copy);
        });
        return 0;
    }
}

int bt_disconnect(const char *mac) {
    @autoreleasepool {
        IOBluetoothDevice *d = deviceForMAC(mac);
        if (d == nil) return -1;
        return [d closeConnection] == kIOReturnSuccess ? 0 : -1;
    }
}

int bt_start_monitoring(const char *mac) {
    (void)mac;
    return -1;
}

void bt_stop_monitoring(void) {}
