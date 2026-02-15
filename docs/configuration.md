# Configuration Reference

This document covers all configuration options for the BBS.

## Configuration File

The BBS is configured via `config.yaml`. All paths can be absolute or relative to the working directory.

## BBS Settings

```yaml
bbs:
  name: "Twilight BBS"     # BBS name displayed to users
  sysop: "Sysop"          # Sysop name
  max_nodes: 32           # Maximum concurrent connections
```

## Server Settings

```yaml
server:
  telnet_port: 2323       # Telnet server port
  ssh_port: 2222          # SSH server port  
  health_port: 2223       # Health check endpoint port
```

## Path Settings

```yaml
paths:
  menus: "./assets/menus"    # Menu definitions (ANS/ASC/Lua triplets)
  text: "./assets/text"      # Text files
  doors: "./assets/doors"   # Door assets
  data: "./data"            # Runtime data (door temp files, etc)
  database: "./data/twilight.db"  # SQLite database
```

## Door Settings

```yaml
doors:
  dosemu_path: "/usr/bin/dosemu"   # Path to dosemu2 binary
  drive_c: "./doors/drive_c"        # DOS drive C root
```

## Transfer Settings

```yaml
transfer:
  sexyz_path: "/usr/local/bin/sexyz"  # Path to SEXYZ binary for ZMODEM
```
