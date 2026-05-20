#import <Foundation/Foundation.h>
#import <IOBluetooth/IOBluetooth.h>
#import "bluetooth_darwin.h"

// Forward declaration of the exported Go callback (defined in bluetooth_darwin.go).
extern void goOnBluetoothEvent(int kind, const char *mac);

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
    (void)mac;
    return -1;
}

int bt_connect(const char *mac) {
    (void)mac;
    return -1;
}

int bt_disconnect(const char *mac) {
    (void)mac;
    return -1;
}

int bt_start_monitoring(const char *mac) {
    (void)mac;
    return -1;
}

void bt_stop_monitoring(void) {}
