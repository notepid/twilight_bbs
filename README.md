# Twilight BBS

A multi-node ANSI BBS written in Go with Lua scripting, supporting telnet and SSH connections.

## Features

- **Multi-node** telnet and SSH server
- **ANSI-BBS standard** terminal emulation (per bansi.txt)
- **Lua-scripted menus** with ANS/ASC display file support
- **User system** with registration, login, security levels
- **Message bases** with areas, threading, new-message scan
- **File areas** with browsing, search, and download tracking
- **Multi-node chat** with rooms and private messaging
- **DOS door support** via dosemu2 with DOOR.SYS/DORINFO1.DEF drop files
- **Docker deployment** with multi-stage build

## Quick Start

```bash
# Build and run locally
go build -o tbbs ./cmd/bbs/
./tbbs -config config.yaml

# Admin TUI
go run ./cmd/bbs-admin/            # uses config.yaml by default
go run ./cmd/bbs-admin/ -config config.yaml

# Connect
telnet localhost 2323
# or
ssh -p 2222 localhost
```

## Docker

```bash
docker compose up -d
```

## Menu System

Menus are defined as file triplets in `assets/menus/`:

- `menu_name.ans` - ANSI color display (preferred)
- `menu_name.asc` - Plain ASCII fallback
- `menu_name.lua` - Lua script with key/input handlers

See [docs/menu_scripting.md](./docs/menu_scripting.md) for complete menu scripting guide.

### Configuring doors

See [docs/doors.md](./docs/doors.md) for complete door configuration reference.

## Configuration

See [docs/configuration.md](./docs/configuration.md) for all configuration options.

## Technology

- **Go** - host language
- **gopher-lua** - embedded Lua 5.1 VM
- **modernc.org/sqlite** - pure-Go SQLite
- **golang.org/x/crypto/ssh** - SSH server

## License

MIT
