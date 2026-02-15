# Doors Configuration

This document covers door configuration for running DOS games via DOSEMU2.

## Configuring doors

Doors are configured in the door menu Lua script: `assets/menus/door_menu.lua`.

- Edit `assets/menus/door_menu.ans` / `assets/menus/door_menu.asc` for the on-screen text.
- Edit the `doors` table in `assets/menus/door_menu.lua` to map single-key hotkeys to door configs.

## Door Configuration Reference

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name of the door |
| `command` | string | DOS command to execute |
| `description` | string | Description shown to users |
| `drop_file_type` | string | "DOOR.SYS" or "DORINFO1.DEF" (default: "DOOR.SYS") |
| `security_level` | number | Minimum security level to access (default: 10) |
| `multiuser` | bool | Allow concurrent users (default: true). If false, only one user can run it at a time. |

### Placeholders

| Placeholder | Replaced with |
|-------------|---------------|
| `{NODE}` | Node number (1, 2, 3, ...) |
| `{DROP}` | DOS path to drop file dir (`C:\NODES\TEMP1`) |

Example:
```lua
["M"] = {
    name = "MyDoor",
    description = "My cool door game",
    command = "C:\\MYDOOR\\MYDOOR.EXE /N{NODE} /DC:\\NODES\\TEMP{NODE}",
    drop_file_type = "DOOR.SYS",
    security_level = 10,
    multiuser = true,
},
```

## Docker Deployment

See [doors/README.md](../doors/README.md) for complete door system architecture and deployment details.
