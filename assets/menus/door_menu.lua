-- door_menu.lua - DOS doors menu
local menu = {}

-- Sysop-configurable door hotkeys.
-- Edit `door_menu.ans` / `door_menu.asc` to match what you list on-screen, then
-- configure the actual door launch parameters here.
--
-- Required fields per door:
-- - name (string)
-- - command (string, DOS path inside drive C, e.g. "C:\\HELLO\\HELLO.BAT")
-- Optional:
-- - description (string)
-- - drop_file_type (string; default "DOOR.SYS")
-- - security_level (number; default 10)
local doors = {
    -- Example smoke-test door (matches prior DB seed)
    ["H"] = {
        name = "HELLO",
        description = "Example test door (DOSEMU2 smoke test)",
        command = "C:\\HELLO\\HELLO.BAT",
        drop_file_type = "DOOR.SYS",
        security_level = 10,
    },
}

function menu.on_load(node)
    node:cls()
end

function menu.on_enter(node)
    -- Check if dosemu2 is available
    if not door.available() then
        node:sendln("\r\n  DOS door support requires dosemu2.")
        node:sendln("  dosemu2 is not currently installed.")
        node:sendln("")
        node:pause()
        node:goto_menu("main_menu")
        return
    end
end

function menu.on_key(node, key)
    if key == "Q" or key == "q" then
        node:goto_menu("main_menu")
        return
    end

    -- Normalize to uppercase so sysop can define only one key per door.
    local k = string.upper(key or "")
    if k == "" then
        return
    end

    local cfg = doors[k]
    if cfg == nil then
        --node:sendln("\r\n  Unknown door option.")
        --node:pause()
        node:goto_menu("door_menu")
        return
    end

    local err = door.launch(cfg)
    if err then
        node:sendln("\r\n  Error: " .. err)
        --node:pause()
    end
    node:pause()
    node:goto_menu("door_menu")
end

return menu
