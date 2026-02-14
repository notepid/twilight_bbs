-- message_areas.lua - Select message area
local menu = {}

local function status(node, text)
    node:output_field("STATUS", text or "")
end

local function set_current_area(node, area_id, area_name)
    node:set_session("current_area", area_id)
    node:set_session("current_area_name", area_name or "")
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

    local lines = {}
    for _, area in ipairs(areas) do
        table.insert(lines, string.format("%-3d %-30s %5d %5d", area.id, area.name, area.total, area.new))
    end
    node:output_field("AREA_LIST", table.concat(lines, "\n"))

    local choice = node:input_field("AREA_CHOICE", 5)
    if choice == nil then
        choice = node:ask("Area #: ", 5)
    end
    if choice == nil or choice == "" or string.upper(choice) == "Q" then
        node:goto_menu("message_menu")
        return
    end

    local area_id = tonumber(choice)
    if not area_id then
        status(node, "Invalid area number.")
        node:pause()
        node:goto_menu("message_areas")
        return
    end

    local area = msg.get_area(area_id)
    if area == nil then
        status(node, "Area not found.")
        node:pause()
        node:goto_menu("message_areas")
        return
    end

    set_current_area(node, area_id, area.name)
    status(node, "Current area set to: " .. area.name)
    --node:pause()
    node:goto_menu("message_menu")
end

return menu

