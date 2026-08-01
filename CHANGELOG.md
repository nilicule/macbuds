# Changelog

All notable changes to MacBuds are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.0] - 2026-08-01

### Added

- Battery level for the selected device, shown next to the menubar icon and in
  the status line (`Connected: Remco's Pixel Buds Pro · 92%`), refreshed once a
  minute while connected.
- Low battery notifications at 20% and 10%, each firing once per discharge
  cycle. Recharging past a threshold re-arms it. Toggle under **Notifications →
  Notify on low battery**, on by default.
- `notify_low_battery` config field, defaulted on for existing configs through
  the usual `notifications_configured` upgrade path.

### Notes

Battery level had been removed in 1.1.0 on the conclusion that no API exposed it
for non-Apple Bluetooth audio devices. That held for the approach used at the
time — parsing `system_profiler SPBluetoothDataType`, which reports no battery
fields at all. The level instead comes from an `IOBluetoothDevice` selector that
macOS's own Bluetooth settings pane uses. It is absent from the public headers,
so MacBuds probes for it at runtime and omits the percentage rather than failing
if a future macOS removes it. Devices that report no level, and any disconnected
device, show no percentage rather than a wrong one.

## [1.2.0] - 2026-05-20

### Changed

- Device picker is now an inline menubar submenu instead of a modal dialog.
- Paired devices are deduplicated by MAC; IOBluetooth occasionally lists a
  device twice.

### Removed

- `zenity` dependency and its transitive dependencies.

## [1.1.0] - 2026-05-20

### Changed

- Replaced `blueutil` subprocess calls with a native IOBluetooth cgo bridge, so
  MacBuds no longer shells out or requires an external binary.
- Connection state is tracked with IOBluetooth notifications instead of polling.

### Removed

- Battery level display. See the note under 1.3.0 for why it came back.
- `blueutil` from the release workflow.

## [1.0.5] - 2026-05-20

Release automation moved to GitHub Actions.

## Earlier releases

See the [Releases page](https://github.com/nilicule/macbuds/releases) for 1.0.0
through 1.0.4.

[1.3.0]: https://github.com/nilicule/macbuds/releases/tag/v1.3.0
[1.2.0]: https://github.com/nilicule/macbuds/releases/tag/v1.2.0
[1.1.0]: https://github.com/nilicule/macbuds/releases/tag/v1.1.0
[1.0.5]: https://github.com/nilicule/macbuds/releases/tag/v1.0.5
