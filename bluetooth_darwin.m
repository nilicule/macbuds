#import <Foundation/Foundation.h>
#import <IOBluetooth/IOBluetooth.h>
#import "bluetooth_darwin.h"

// Forward declaration of the exported Go callback (defined in bluetooth_darwin.go).
extern void goOnBluetoothEvent(int kind, const char *mac);

int bt_paired_devices(bt_device_t *out, int max_count) {
    (void)out; (void)max_count;
    return -1;
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
