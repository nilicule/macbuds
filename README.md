# MacBuds

A simple macOS menubar application to control your Bluetooth earbuds. It allows you to quickly connect and disconnect your earbuds directly from the menubar, with visual status indication.

## Features

- Lives in your menubar with visual connection status (✓ connected, × disconnected)
- One-click connect/disconnect
- Pick your target device from a list of paired Bluetooth devices
- Live battery level for the selected device
- Optional notifications on connect, disconnect, and low battery
- Optional launch at login
- Lightweight and native macOS experience

## Installation

The easiest way is to grab a prebuilt release:

1. Download the latest `MacBuds-vX.Y.Z.zip` from the [Releases page](https://github.com/nilicule/macbuds/releases).
2. Unzip and drag `MacBuds.app` to `/Applications`.
3. Because the app isn't code-signed, the first launch needs a right-click → **Open** to bypass Gatekeeper. After the first launch, it opens normally from the menubar.

The first launch may prompt for Bluetooth permission. Grant it so MacBuds can list paired devices and trigger connect/disconnect.

## Configuration

1. Click the menubar icon and choose **Select Device**.
2. Pick your earbuds (or any paired Bluetooth device) from the list.
3. Use **Connect** / **Disconnect** to toggle the connection. The icon and status line update automatically.

To switch devices later, just choose **Select Device** again. **Clear Selected Device** resets the selection.

### Notifications

Open the **Notifications** submenu to toggle:

- Notify on connect
- Notify on disconnect
- Low battery warning (fires when the device drops below 20%)

## Auto-start Configuration

To have MacBuds start automatically when you log in:

1. Click the menubar icon
2. Toggle "Launch at Login"

## Building from Source

Prerequisites:

- macOS
- Go 1.26 or later (matches `go.mod`)
- Xcode Command Line Tools (for cgo + the `IOBluetooth` framework headers)

```bash
git clone https://github.com/nilicule/macbuds.git
cd macbuds
go mod tidy
go build -o macbuds
./macbuds
```

Dev builds run from a raw binary (not a `.app`) may have Bluetooth access attributed to the terminal app rather than MacBuds — the simplest realistic dev loop is to run the workflow-built `.app` for end-to-end testing.

Release artifacts (packaged `.app` zips) are produced automatically by the GitHub Actions workflow at `.github/workflows/release.yml` whenever a GitHub Release is published.

## License

This project is licensed under the GNU General Public License v3.0 - see [LICENSE](LICENSE) file for details.

GPL-3.0 is a copyleft license that requires anyone who distributes your code or a derivative work to make the source code available under the same terms. This ensures that modifications and larger works based on your code must also be free and open source.