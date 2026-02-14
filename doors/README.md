# Doors (DOSEMU drive C)

This folder holds DOS doors that are made available to DOSEMU2 as drive `C:\`.

## Layout

- `doors/drive_c/` is the root of DOSEMU drive `C:\`
- Put each door under its own folder, for example:
  - `doors/drive_c/HELLO/HELLO.BAT`  → runs as `C:\HELLO\HELLO.BAT`

## Docker image

The Docker build copies `doors/drive_c/` into the image at:

- `/opt/bbs/doors/drive_c`

During build, `*.bat` / `*.cmd` / `*.txt` files under that folder are normalized to CRLF.

## Live mounting doors (recommended for development)

You can override the baked-in doors with a bind mount:

```bash
docker run --rm -it ^
  -p 2323:2323 -p 2222:2222 ^
  -v "%cd%\\doors\\drive_c:/opt/bbs/doors/drive_c" ^
  twilight_bbs
```

For docker compose:

```yaml
services:
  bbs:
    image: twilight_bbs
    ports:
      - "2323:2323"
      - "2222:2222"
    volumes:
      - ./doors/drive_c:/opt/bbs/doors/drive_c
```

## DOS compatibility

Git attributes in the repo enforce:

- CRLF line endings for DOS text files under `doors/`
- binary-safe handling for common door binaries/archives (`.exe`, `.com`, `.zip`, etc.)

If you edit door batch files on a non-Windows host, ensure your editor preserves CRLF.
