-- message_list.lua - List messages in current area
local menu = {}

local function status(node, text)
    node:output_field("STATUS", text or "")
end

local function get_or_default_area(node)
    local area_id = node:get_session("current_area")
    if area_id ~= nil then
        return tonumber(area_id)
    end

    local areas = msg.areas()
    if areas and #areas > 0 then
        node:set_session("current_area", areas[1].id)
        node:set_session("current_area_name", areas[1].name or "")
        return areas[1].id
    end
    return nil
end

function menu.on_load(node)
    node:cls()
end

function menu.on_enter(node)
    local area_id = get_or_default_area(node)
    if area_id == nil then
        status(node, "No message areas available.")
        node:pause()
        node:goto_menu("message_menu")
        return
    end
    -- Clear any stale status text.
    status(node, "")

    local area = msg.get_area(area_id)
    if area then
        node:output_field("AREA_NAME", area.name or "")
    else
        node:output_field("AREA_NAME", tostring(area_id))
    end

    local messages = msg.list(area_id, 0, 20)
    if messages == nil or #messages == 0 then
        status(node, "No messages in this area.")
        node:pause()
        node:goto_menu("message_menu")
        return
    end

    local lines = {}
    for _, m in ipairs(messages) do
        local from = m.from or ""
        local subject = m.subject or ""
        local date = m.date or ""
        table.insert(lines, string.format("%-4d %-16s %-26s %-16s",
            m.id,
            string.sub(from, 1, 16),
            string.sub(subject, 1, 26),
            string.sub(date, 1, 16)
        ))
    end
    node:output_field("MESSAGE_LIST", table.concat(lines, "\n"))

    local choice = node:input_field("MSG_CHOICE", 6)
    if choice == nil then
        choice = node:ask("Read Msg #: ", 6)
    end
    if choice == nil or choice == "" or string.upper(choice) == "Q" then
        node:goto_menu("message_menu")
        return
    end

    local msg_id = tonumber(choice)
    if not msg_id then
        status(node, "Invalid message number.")
        node:pause()
        node:goto_menu("message_list")
        return
    end

    node:set_session("current_msg_id", msg_id)
    node:goto_menu("message_read")
end

return menu

