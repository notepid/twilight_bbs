-- message_scan.lua - Show new message counts
local menu = {}

local function status(node, text)
    node:output_field("STATUS", text or "")
end

function menu.on_load(node)
    node:cls()
end

function menu.on_enter(node)
    status(node, "")
    local areas = msg.areas()
    if areas == nil or #areas == 0 then
        status(node, "No message areas available.")
        node:pause()
        node:goto_menu("message_menu")
        return
    end

    local total_new = 0
    local lines = {}
    for _, area in ipairs(areas) do
        local new_count = area.new or 0
        total_new = total_new + new_count
        table.insert(lines, string.format("%-30s  %4d", area.name or "", new_count))
    end

    node:output_field("SCAN_LIST", table.concat(lines, "\n"))
    if total_new == 0 then
        status(node, "No new messages.")
    else
        status(node, "Total new messages: " .. tostring(total_new))
    end

    node:pause()
    node:goto_menu("message_menu")
end

return menu

