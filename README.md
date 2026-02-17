# HushApp

HushApp is a secure, serverless, local-first chat application that works over Wi-Fi (mDNS) with zero configuration. It features both a modern GUI and a terminal-based interface (TUI).

## Features
- **Serverless**: No central server needed. Peers discover each other automatically on the local network.
- **Secure**: Uses `libp2p` for encrypted communication.
- **Cross-Platform**: Runs on macOS, Windows, and Linux.
- **Dual Interface**: Use the native GUI or the lightweight terminal interface.

## Getting Started

### Prerequisites
- Go 1.25+
- Node.js & npm (for GUI build)
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Running the GUI
```bash
# Development mode (hot reload)
wails dev

# Build production binary
wails build
open build/bin/HushApp.app
```

### Running the Terminal UI (TUI)
You can run the terminal version directly from the source:
```bash
go run cmd/tui/main.go
```

## Project Structure
- `cmd/tui/`: Entry point for the Terminal UI.
- `frontend/`: React + Tailwind CSS frontend code for the GUI.
- `internal/`: Shared core logic (networking, chat protocol).
- `app.go`: Wails backend bindings.
- `main.go`: Wails GUI entry point.

## License
MIT