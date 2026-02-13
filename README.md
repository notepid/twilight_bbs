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

### Lua Script Structure

```lua
local menu = {}

function menu.on_load(node)
    -- called BEFORE menu art is displayed
    -- use this to clear screen or set initial state
    node:cls()
end

function menu.on_enter(node)
    -- called AFTER menu art is displayed
end

function menu.on_key(node, key)
    -- called on single keypress (hotkey mode)
    if key == "Q" or key == "q" then
        node:goto_menu("main_menu")
    end
end

function menu.on_input(node, input)
    -- called when user types string + Enter
end

function menu.on_exit(node)
    -- called when leaving the menu
end

return menu
```

### Lua API

**Node I/O:** `node:send()`, `node:sendln()`, `node:cls()`, `node:display()`, `node:goto_xy()`, `node:color()`, `node:pause()`

**Input:** `node:getkey()`, `node:getline()`, `node:hotkey()`, `node:ask()`, `node:password()`, `node:yesno()`

**Navigation:** `node:goto_menu()`, `node:gosub_menu()`, `node:return_menu()`, `node:disconnect()`

**Users:** `users.login()`, `users.register()`, `users.exists()`, `users.get_current()`, `users.list()`

**Messages:** `msg.areas()`, `msg.list()`, `msg.read()`, `msg.post()`, `msg.scan_new()`

**Files:** `files.areas()`, `files.list()`, `files.search()`, `files.get_file()`

**Doors:** `door.list()`, `door.launch()`, `door.available()`

**Chat:** `chat.send()`, `chat.broadcast()`, `chat.online()`, `chat.enter_room()`, `chat.send_room()`

## Configuration

See `config.yaml` for all options. Key settings:

```yaml
bbs:
  name: "Twilight BBS"
  sysop: "Sysop"
  max_nodes: 32

server:
  telnet_port: 2323
  ssh_port: 2222

paths:
  menus: "./assets/menus"
  database: "./data/twilight.db"
```

## Technology

- **Go** - host language
- **gopher-lua** - embedded Lua 5.1 VM
- **modernc.org/sqlite** - pure-Go SQLite
- **golang.org/x/crypto/ssh** - SSH server

## License

MIT
