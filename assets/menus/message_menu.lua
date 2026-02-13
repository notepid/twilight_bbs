-- message_menu.lua - Message bases menu
local menu = {}
local current_area = nil

function menu.on_key(node, key)
    if key == "L" or key == "l" then
        list_areas(node)
    elseif key == "R" or key == "r" then
        if current_area then
            read_messages(node, current_area)
        else
            node:sendln("\r\n  Select an area first (press L).")
            node:pause()
        end
        node:goto_menu("message_menu")
    elseif key == "P" or key == "p" then
        if current_area then
            post_message(node, current_area)
        else
            node:sendln("\r\n  Select an area first (press L).")
            node:pause()
        end
        node:goto_menu("message_menu")
    elseif key == "S" or key == "s" then
        scan_new(node)
        node:goto_menu("message_menu")
    elseif key == "Q" or key == "q" then
        node:goto_menu("main_menu")
    end
end

function list_areas(node)
    local areas = msg.areas()
    if areas == nil or #areas == 0 then
        node:sendln("\r\n  No message areas available.")
        node:pause()
        node:goto_menu("message_menu")
        return
    end

    node:sendln("")
    node:sendln("  #   Name                          Total  New")
    node:sendln("  --- ------------------------------ ----- -----")
    for i, area in ipairs(areas) do
        local line = string.format("  %-3d %-30s %5d %5d",
            area.id, area.name, area.total, area.new)
        node:sendln(line)
    end
    node:sendln("")

    local choice = node:ask("  Enter area # (or Q to cancel): ", 5)
    if choice == nil or choice == "" or string.upper(choice) == "Q" then
        node:goto_menu("message_menu")
        return
    end

    local area_id = tonumber(choice)
    if area_id then
        local area = msg.get_area(area_id)
        if area then
            current_area = area_id
            node:sendln("  Current area: " .. area.name)
            node:pause()
        else
            node:sendln("  Area not found.")
            node:pause()
        end
    end
    node:goto_menu("message_menu")
end

function read_messages(node, area_id)
    local messages = msg.list(area_id, 0, 20)
    if messages == nil or #messages == 0 then
        node:sendln("\r\n  No messages in this area.")
        node:pause()
        return
    end

    node:sendln("")
    node:sendln("  #    From             Subject                    Date")
    node:sendln("  ---- ---------------- -------------------------- ----------------")
    for _, m in ipairs(messages) do
        local line = string.format("  %-4d %-16s %-26s %s",
            m.id, string.sub(m.from, 1, 16), string.sub(m.subject, 1, 26), m.date)
        node:sendln(line)
    end
    node:sendln("")

    local choice = node:ask("  Read message # (or Q): ", 5)
    if choice and choice ~= "" and string.upper(choice) ~= "Q" then
        local msg_id = tonumber(choice)
        if msg_id then
            local m = msg.read(msg_id)
            if m then
                node:cls()
                node:sendln("  From:    " .. m.from)
                if m.to and m.to ~= "" then
                    node:sendln("  To:      " .. m.to)
                end
                node:sendln("  Subject: " .. m.subject)
                node:sendln("  Date:    " .. m.date)
                node:sendln("  ─────────────────────────────────────────────")
                node:sendln("")
                node:sendln(m.body)
                node:sendln("")
                node:pause()
            else
                node:sendln("  Message not found.")
                node:pause()
            end
        end
    end
end

function post_message(node, area_id)
    node:sendln("")
    node:sendln("  -- Post New Message --")
    node:sendln("")
    local subject = node:ask("  Subject: ", 60)
    if subject == nil or subject == "" then
        node:sendln("  Cancelled.")
        node:pause()
        return
    end

    node:sendln("  Enter message (blank line to finish):")
    node:sendln("")

    local lines = {}
    while true do
        local line = node:ask("  > ", 78)
        if line == nil or line == "" then
            break
        end
        table.insert(lines, line)
    end

    if #lines == 0 then
        node:sendln("  Empty message, cancelled.")
        node:pause()
        return
    end

    local body = table.concat(lines, "\n")
    local id, err = msg.post(area_id, subject, body)
    if id then
        node:sendln("  Message posted! (#" .. id .. ")")
    else
        node:sendln("  Error posting: " .. (err or "unknown"))
    end
    node:pause()
end

function scan_new(node)
    local areas = msg.areas()
    if areas == nil then
        node:sendln("\r\n  No areas available.")
        node:pause()
        return
    end

    node:sendln("")
    node:sendln("  -- New Message Scan --")
    node:sendln("")

    local total_new = 0
    for _, area in ipairs(areas) do
        if area.new > 0 then
            node:sendln(string.format("  %s: %d new message(s)", area.name, area.new))
            total_new = total_new + area.new
        end
    end

    if total_new == 0 then
        node:sendln("  No new messages.")
    else
        node:sendln(string.format("\r\n  Total: %d new message(s)", total_new))
    end
    node:sendln("")
    node:pause()
end

return menu
