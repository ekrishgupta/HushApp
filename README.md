# 👻 HushApp

**Secure, serverless chat that just works.** No accounts. No servers. Just open it and talk to anyone on the same Wi-Fi.

[![Latest Release](https://img.shields.io/github/v/release/ekrishgupta/HushApp?style=flat-square&label=latest)](https://github.com/ekrishgupta/HushApp/releases/latest)

---

## Download

| Platform | Download |
|----------|----------|
| 🍎 **macOS** | [**Download for Mac**](https://github.com/ekrishgupta/HushApp/releases/latest/download/Hush-macOS.dmg) |
| 🪟 **Windows** | [**Download for Windows**](https://github.com/ekrishgupta/HushApp/releases/latest/download/Hush-Windows-installer.exe) |

> **macOS:** Open the `.dmg` and drag Hush to Applications. On first launch, right-click → Open (the app is not yet code-signed).

---

## Features

- 👻 **Zero Config** — Open the app, start chatting. No accounts, no servers.
- 🔒 **Encrypted** — All messages are encrypted via `libp2p`.
- 📡 **Local Network** — Peers discover each other automatically over Wi-Fi (mDNS).
- 💻 **Cross-Platform** — Native apps for macOS and Windows.
- ⌨️ **Terminal Mode** — Lightweight TUI for power users.

---

## For Developers

### Prerequisites
- Go 1.25+
- Node.js & npm
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

### Build from source
```bash
# Development mode (hot reload)
wails dev

# Production build
wails build
```

### Terminal UI
```bash
go run cmd/tui/main.go
```

### Project Structure
| Path | Description |
|------|-------------|
| `cmd/tui/` | Terminal UI entry point |
| `frontend/` | React + Tailwind CSS frontend |
| `internal/` | Core networking & chat logic |
| `app.go` | Wails backend bindings |
| `main.go` | GUI entry point |

### Creating a Release
```bash
git tag v1.0.0
git push origin v1.0.0
# GitHub Actions will build and publish automatically
```

## License
MIT