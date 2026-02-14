-- message_post.lua - Post a new message
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

local function preview_text(lines, height)
    if height <= 0 then
        height = 10
    end
    if #lines <= height then
        return table.concat(lines, "\n")
    end
    local start = (#lines - height) + 1
    local slice = {}
    for i = start, #lines do
        table.insert(slice, lines[i])
    end
    return table.concat(slice, "\n")
end

function menu.on_load(node)
    node:cls()
end

function menu.on_enter(node)
    status(node, "")
    local area_id = get_or_default_area(node)
    if area_id == nil then
        status(node, "No message areas available.")
        node:pause()
        node:goto_menu("message_menu")
        return
    end

    local area = msg.get_area(area_id)
    node:output_field("AREA_NAME", area and area.name or tostring(area_id))

    local subject = node:input_field("SUBJECT", 60)
    if subject == nil then
        subject = node:ask("Subject: ", 60)
    end
    subject = subject and tostring(subject) or ""
    if subject == "" then
        status(node, "Cancelled.")
        node:pause()
        node:goto_menu("message_menu")
        return
    end

    local lines = {}
    node:output_field("BODY_PREVIEW", "")

    while true do
        node:output_field("BODY_PREVIEW", preview_text(lines, 10))
        local line = node:input_field("BODY_LINE", 78)
        if line == nil then
            line = node:ask("> ", 78)
        end
        if line == nil then
            status(node, "Cancelled.")
            node:pause()
            node:goto_menu("message_menu")
            return
        end
        if line == "" then
            break
        end
        table.insert(lines, line)
    end

    if #lines == 0 then
        status(node, "Empty message, cancelled.")
        node:pause()
        node:goto_menu("message_menu")
        return
    end

    local body = table.concat(lines, "\n")
    local id, err = msg.post(area_id, subject, body)
    if id then
        status(node, "Message posted! (#" .. tostring(id) .. ")")
    else
        status(node, "Error posting: " .. tostring(err or "unknown"))
    end

    node:pause()
    node:goto_menu("message_menu")
end

return menu

