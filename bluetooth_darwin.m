#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <IOBluetooth/IOBluetooth.h>
#import "bluetooth_darwin.h"

// Forward declaration of the exported Go callback (defined in bluetooth_darwin.go).
extern void goOnBluetoothEvent(int kind, const char *mac);

static IOBluetoothDevice *deviceForMAC(const char *mac) {
    if (mac == NULL) return nil;
    NSString *s = [NSString stringWithUTF8String:mac];
    return [IOBluetoothDevice deviceWithAddressString:s];
}

// Returns an autoreleased NSString — caller retains if it needs to keep it.
static NSString *normalizeMACString(NSString *s) {
    return [[s stringByReplacingOccurrencesOfString:@"-" withString:@":"] lowercaseString];
}

@interface BTMonitor : NSObject {
@public
    IOBluetoothUserNotification *connectNotification;
    IOBluetoothUserNotification *disconnectNotification;
    NSString *targetMAC; // normalized: lowercase, colon-separated
}
- (void)startForMAC:(NSString *)mac;
- (void)stop;
- (void)deviceConnected:(IOBluetoothUserNotification *)note device:(IOBluetoothDevice *)d;
- (void)deviceDisconnected:(IOBluetoothUserNotification *)note device:(IOBluetoothDevice *)d;
@end

static BTMonitor *g_monitor = nil;

@implementation BTMonitor

- (void)startForMAC:(NSString *)mac {
    [self stop];
    targetMAC = [normalizeMACString(mac) retain];
    // System-wide connect notification — we filter to our MAC in the handler.
    connectNotification = [[IOBluetoothDevice
        registerForConnectNotifications:self
        selector:@selector(deviceConnected:device:)] retain];
    // Per-device disconnect notification.
    IOBluetoothDevice *d = [IOBluetoothDevice deviceWithAddressString:mac];
    if (d != nil) {
        disconnectNotification = [[d
            registerForDisconnectNotification:self
            selector:@selector(deviceDisconnected:device:)] retain];
    }
}

- (void)stop {
    if (connectNotification != nil) {
        [connectNotification unregister];
        [connectNotification release];
        connectNotification = nil;
    }
    if (disconnectNotification != nil) {
        [disconnectNotification unregister];
        [disconnectNotification release];
        disconnectNotification = nil;
    }
    [targetMAC release];
    targetMAC = nil;
}

- (void)deviceConnected:(IOBluetoothUserNotification *)note device:(IOBluetoothDevice *)d {
    NSString *addr = normalizeMACString([d addressString]);
    if (targetMAC != nil && [addr isEqualToString:targetMAC]) {
        goOnBluetoothEvent(BT_EVENT_CONNECTED, [[d addressString] UTF8String]);
    }
}

- (void)deviceDisconnected:(IOBluetoothUserNotification *)note device:(IOBluetoothDevice *)d {
    goOnBluetoothEvent(BT_EVENT_DISCONNECTED, [[d addressString] UTF8String]);
}

@end

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
    @autoreleasepool {
        if (mac == NULL) return -1;
        if (g_monitor == nil) g_monitor = [[BTMonitor alloc] init];
        [g_monitor startForMAC:[NSString stringWithUTF8String:mac]];
        return 0;
    }
}

void bt_stop_monitoring(void) {
    @autoreleasepool {
        if (g_monitor != nil) {
            [g_monitor stop];
        }
    }
}

int bt_pick_device(const bt_device_t *devices, int count,
                   char *out_mac, int out_mac_size) {
    if (devices == NULL || count <= 0 || out_mac == NULL || out_mac_size <= 0) {
        return -1;
    }

    __block int result = -1;
    __block NSInteger selectedIndex = -1;

    dispatch_sync(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            NSAlert *alert = [[NSAlert alloc] init];
            [alert setMessageText:@"Select Bluetooth Device"];
            [alert setInformativeText:@"Choose a device to control:"];
            [alert addButtonWithTitle:@"OK"];
            [alert addButtonWithTitle:@"Cancel"];

            NSPopUpButton *popup = [[NSPopUpButton alloc]
                initWithFrame:NSMakeRect(0, 0, 300, 25) pullsDown:NO];
            for (int i = 0; i < count; i++) {
                NSString *title = [NSString stringWithFormat:@"%s (%s)",
                                   devices[i].name, devices[i].mac];
                [popup addItemWithTitle:title];
            }
            [alert setAccessoryView:popup];

            NSModalResponse response = [alert runModal];
            if (response == NSAlertFirstButtonReturn) {
                selectedIndex = [popup indexOfSelectedItem];
                result = 0;
            } else {
                result = 1;
            }
        }
    });

    if (result == 0 && selectedIndex >= 0 && selectedIndex < count) {
        strncpy(out_mac, devices[selectedIndex].mac, out_mac_size - 1);
        out_mac[out_mac_size - 1] = '\0';
    }
    return result;
}
