-- sysop_menu.lua - Sysop administration menu
-- Requires security level 100 (LevelSysop)
local menu = {}

function menu.on_load(node)
    node:cls()
end

function menu.on_enter(node)
    local user = users.get_current()
    if user == nil or user.level < 100 then
        node:sendln("\r\n  Access denied. Sysop level required.")
        node:pause()
        node:goto_menu("main_menu")
        return
    end
end

function menu.on_key(node, key)
    if key == "U" or key == "u" then
        user_management(node)
        node:goto_menu("sysop_menu")
    elseif key == "N" or key == "n" then
        node_control(node)
        node:goto_menu("sysop_menu")
    elseif key == "R" or key == "r" then
        node:sendln("\r\n  Menus reloaded.")
        node:pause()
        node:goto_menu("sysop_menu")
    elseif key == "S" or key == "s" then
        system_stats(node)
        node:goto_menu("sysop_menu")
    elseif key == "Q" or key == "q" then
        node:goto_menu("main_menu")
    end
end

function user_management(node)
    node:sendln("")
    node:sendln("  -- User List --")
    node:sendln("")

    local user_list = users.list()
    if user_list == nil or #user_list == 0 then
        node:sendln("  No users found.")
        node:pause()
        return
    end

    node:sendln("  ID   Username         Level  Calls  Last On")
    node:sendln("  ---- ---------------- ------ ------ ----------------")
    for _, u in ipairs(user_list) do
        local line = string.format("  %-4d %-16s %6d %6d %s",
            u.id, u.name, u.level, u.calls, u.last_on or "never")
        node:sendln(line)
    end
    node:sendln("")
    node:pause()
end

function node_control(node)
    node:sendln("")
    node:sendln("  -- Online Nodes --")
    node:sendln("")

    local online = chat.online()
    if online == nil or #online == 0 then
        node:sendln("  No users online.")
    else
        node:sendln("  Node  User               Status")
        node:sendln("  ----  -----------------  --------")
        for _, u in ipairs(online) do
            local status = "Online"
            if u.room and u.room ~= "" then
                status = "Chat: " .. u.room
            end
            local line = string.format("  %-4d  %-17s  %s",
                u.node_id, u.name, status)
            node:sendln(line)
        end
    end
    node:sendln("")
    node:pause()
end

function system_stats(node)
    node:sendln("")
    node:sendln("  -- System Statistics --")
    node:sendln("")

    local online = chat.online()
    local online_count = 0
    if online then
        online_count = #online
    end

    node:sendln("  Nodes online: " .. online_count)
    node:sendln("")

    -- Show message area stats
    local areas = msg.areas()
    if areas and #areas > 0 then
        node:sendln("  Message Areas:")
        for _, a in ipairs(areas) do
            node:sendln(string.format("    %s: %d messages", a.name, a.total))
        end
    end

    node:sendln("")
    node:pause()
end

return menu
