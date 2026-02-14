# Doors (DOSEMU2 Drive C)

This folder holds DOS doors that are made available to DOSEMU2 as drive `C:\`.

## Architecture

```
BBS terminal (SSH/Telnet)
    ↕  io.ReadWriter
PTY master fd  (Go creack/pty)
    ↕  stdin/stdout
dosemu2  (-quiet -f dosemurc -E RUN.BAT)
    ↕  $_com1 = "virtual"  →  maps COM1 to stdio
BNU FOSSIL driver  (C:\BNU\BNU.COM)
    ↕  INT 14h FOSSIL API
DOS door game  (e.g. DARK16.EXE)
```

The BBS spawns dosemu2 directly in a PTY using Go's `creack/pty` package.
No socat, no TCP sockets, no wrapper scripts — just a clean PTY bridge.

dosemu2's `$_com1 = "virtual"` maps DOS COM1 to its stdin/stdout, which
is the PTY slave. The BNU FOSSIL driver provides the INT 14h FOSSIL API
that door games use to communicate over the virtual serial port.

## Layout

- `doors/drive_c/` is the root of DOSEMU drive `C:\`
- Put each door under its own folder, for example:
  - `doors/drive_c/HELLO/HELLO.BAT`  → runs as `C:\HELLO\HELLO.BAT`
  - `doors/drive_c/DOORS/DARKNESS/DARK16.EXE` → runs as `C:\DOORS\DARKNESS\DARK16.EXE`

### Key directories

| Host path                         | DOS path          | Purpose                          |
|-----------------------------------|-------------------|----------------------------------|
| `doors/drive_c/BNU/BNU.COM`      | `C:\BNU\BNU.COM`  | BNU FOSSIL driver (v1.70)        |
| `doors/drive_c/HELLO/HELLO.BAT`  | `C:\HELLO\...`    | Smoke test door                  |
| `doors/drive_c/DOORS/DARKNESS/`  | `C:\DOORS\DARKNESS\` | Darkness v2.0 door game       |
| `doors/drive_c/NODES/TEMP{N}/`   | `C:\NODES\TEMP{N}\`  | Drop files + RUN.BAT (generated per session) |

## Adding a new door

1. Create a directory under `doors/drive_c/` for your door (e.g. `MYDOOR/`).
2. Copy the door's files into it.
3. Edit `assets/menus/door_menu.lua` and add a hotkey entry:

```lua
["M"] = {
    name = "MyDoor",
    description = "My cool door game",
    command = "C:\\MYDOOR\\MYDOOR.EXE /N{NODE} /DC:\\NODES\\TEMP{NODE}",
    drop_file_type = "DOOR.SYS",   -- or "DORINFO1.DEF"
    security_level = 10,
},
```

### Placeholders

| Placeholder | Replaced with                         |
|-------------|---------------------------------------|
| `{NODE}`    | Node number (1, 2, 3, ...)            |
| `{DROP}`    | DOS path to drop file dir (`C:\NODES\TEMP1`) |

### Drop files

The BBS auto-generates a drop file (DOOR.SYS or DORINFO1.DEF) in
`C:\NODES\TEMP{N}\` before launching the door. The wrapper batch file
(`RUN.BAT`) is also placed there.

## BNU FOSSIL driver

BNU v1.70 is included at `doors/drive_c/BNU/BNU.COM`. It is loaded
automatically by the generated `RUN.BAT` wrapper before the door runs:

```bat
@ECHO OFF
C:\BNU\BNU.COM /L0:57600,8N1 /F
C:\DOORS\DARKNESS\DARK16.EXE /N1 /D3 /PC:\NODES\TEMP1
EXITEMU
```

Source: http://www.pcmicro.com/bnu/

## Docker

The Docker build copies `doors/drive_c/` into the image at `/opt/bbs/doors/drive_c`.
During build, `*.bat` / `*.cmd` / `*.txt` files are normalized to CRLF.

### Live mounting (development)

```yaml
services:
  bbs:
    volumes:
      - ./doors/drive_c:/opt/bbs/doors/drive_c
```

## dosemu2 configuration

A per-session `dosemurc` is generated at runtime in `data/doors_tmp/node{N}/.dosemu/dosemurc`:

```
$_cpu = "80486"
$_cpu_emu = "vm86"
$_com1 = "virtual"
$_rawkeyboard = (0)
$_term_updfreq = (8)
$_term_color = (on)
$_external_char_set = "cp437"
$_internal_char_set = "cp437"
```

## DOS compatibility

- CRLF line endings for DOS text files under `doors/`
- Binary-safe handling for `.exe`, `.com`, `.zip`, etc.
- If editing batch files on Linux, ensure CRLF is preserved.
