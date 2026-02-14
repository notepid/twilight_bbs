-- door_menu.lua - DOS doors menu
local menu = {}

function menu.on_load(node)
    --node:cls()
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
    if key == "L" or key == "l" then
        list_doors(node)
    elseif key == "Q" or key == "q" then
        node:goto_menu("main_menu")
    end
end

function list_doors(node)
    local doors = door.list()
    if doors == nil or #doors == 0 then
        node:sendln("\r\n  No doors configured.")
        node:pause()
        node:goto_menu("door_menu")
        return
    end

    node:sendln("")
    node:sendln("  #   Name                   Description")
    node:sendln("  --- ---------------------- ---------------------------------")
    for i, d in ipairs(doors) do
        local line = string.format("  %-3d %-22s %s",
            i, d.name, d.description)
        node:sendln(line)
    end
    node:sendln("")

    local choice = node:ask("  Enter door name (or Q to cancel): ", 30)
    if choice == nil or choice == "" or string.upper(choice) == "Q" then
        node:goto_menu("door_menu")
        return
    end

    local err = door.launch(choice)
    if err then
        node:sendln("  Error: " .. err)
        node:pause()
    end
    node:goto_menu("door_menu")
end

return menu
