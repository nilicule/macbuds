# MacBuds

A simple macOS menubar application to control your Bluetooth earbuds. It allows you to quickly connect and disconnect your earbuds directly from the menubar, with visual status indication.

## Features

- Lives in your menubar with visual connection status (✓ connected, × disconnected)
- Quick connect/disconnect with one click
- Configurable target device (via MAC address)
- Optional launch at login
- Lightweight and native macOS experience

## Prerequisites

- macOS
- Go 1.16 or later
- [blueutil](https://github.com/toy/blueutil) (can be installed via Homebrew)

## Installation

1. Install blueutil:
```bash
brew install blueutil
```

2. Clone and build the application:
```bash
git clone https://github.com/yourusername/macbuds.git
cd macbuds
go build -o macbuds
```

3. Run the application:
```bash
./macbuds
```

## Configuration

1. Find your earbuds' MAC address:
```bash
blueutil --paired
```

2. Click the menubar icon and select "Configure MAC Address"
3. Enter your device's MAC address in the text editor that opens
4. Save and close the file

## Auto-start Configuration

To have MacBuds start automatically when you log in:

1. Click the menubar icon
2. Toggle "Launch at Login"

## Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/macbuds.git

# Navigate to the directory
cd macbuds

# Get dependencies
go mod tidy

# Build the application
go build -o macbuds
```

## License

This project is licensed under the GNU General Public License v3.0 - see [LICENSE](LICENSE) file for details.

GPL-3.0 is a copyleft license that requires anyone who distributes your code or a derivative work to make the source code available under the same terms. This ensures that modifications and larger works based on your code must also be free and open source.