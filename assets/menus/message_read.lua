-- message_read.lua - Read a message
local menu = {}

local function status(node, text)
    node:output_field("STATUS", text or "")
end

function menu.on_load(node)
    node:cls()
end

function menu.on_enter(node)
    -- Clear any stale status text (art placeholder blanking only clears marker length).
    status(node, "")

    local msg_id = tonumber(node:get_session("current_msg_id"))
    if msg_id == nil then
        status(node, "No message selected.")
        node:pause()
        node:goto_menu("message_menu")
        return
    end

    local m = msg.read(msg_id)
    if m == nil then
        status(node, "Message not found.")
        node:pause()
        node:goto_menu("message_menu")
        return
    end

    node:output_field("FROM", m.from or "")
    node:output_field("TO", m.to or "")
    node:output_field("SUBJECT", m.subject or "")
    node:output_field("DATE", m.date or "")
    node:output_field("BODY", m.body or "")

    -- Move cursor to STATUS line so pause prompt doesn't overwrite BODY.
    local row, col = node:field("STATUS")
    if row ~= nil then
        node:goto_xy(row, col)
    end
    node:pause()
    node:goto_menu("message_menu")
end

return menu

