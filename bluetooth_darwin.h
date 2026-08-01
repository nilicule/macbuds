#ifndef MACBUDS_BLUETOOTH_DARWIN_H
#define MACBUDS_BLUETOOTH_DARWIN_H

#define BT_MAX_DEVICES 64

typedef struct {
    char mac[32];
    char name[256];
} bt_device_t;

typedef struct {
    int single; // 0-100, or -1 when the level is unknown
} bt_battery_t;

typedef enum {
    BT_EVENT_CONNECTED = 1,
    BT_EVENT_DISCONNECTED = 2,
    BT_EVENT_CONNECT_FAILED = 3
} bt_event_kind_t;

int  bt_paired_devices(bt_device_t *out, int max_count);
int  bt_is_connected(const char *mac);
int  bt_connect(const char *mac);
int  bt_disconnect(const char *mac);
int  bt_battery(const char *mac, bt_battery_t *out);

int  bt_start_monitoring(const char *mac);
void bt_stop_monitoring(void);

#endif
