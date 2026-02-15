-- file_menu.lua - File areas menu with ZMODEM-8K transfer support
local menu = {}

local function set_current_area(node, area_id, area_name)
    node:set_session("current_file_area", area_id)
    node:set_session("current_file_area_name", area_name or "")
end

local function get_current_area(node)
    local area_id = node:get_session("current_file_area")
    if area_id ~= nil then
        return tonumber(area_id)
    end
    return nil
end

local function get_or_default_area(node)
    local area_id = get_current_area(node)
    if area_id ~= nil then
        return area_id
    end

    local areas = files.areas()
    if areas and #areas > 0 then
        set_current_area(node, areas[1].id, areas[1].name)
        return areas[1].id
    end
    return nil
end

local function require_transfer(node)
    if transfer == nil or type(transfer) ~= "table" or transfer.available == nil then
        node:sendln("\r\n  File transfer module is not available.")
        node:pause()
        return false
    end
    return true
end

local function parse_id_list(raw)
    local set = {}
    local order = {}
    if raw == nil or raw == "" then
        return set, order
    end

    for token in string.gmatch(raw, "[^,%s]+") do
        local id = tonumber(token)
        if id and not set[id] then
            set[id] = true
            table.insert(order, id)
        end
    end
    return set, order
end

local function save_id_list(node, key, order)
    if order == nil or #order == 0 then
        node:set_session(key, nil)
        return
    end

    local parts = {}
    for _, id in ipairs(order) do
        table.insert(parts, tostring(id))
    end
    node:set_session(key, table.concat(parts, ","))
end

local function get_marked_ids(node)
    return parse_id_list(node:get_session("download_basket"))
end

function menu.on_load(node)
    node:cls()
end

function menu.on_key(node, key)
    if key == "L" or key == "l" then
        list_areas(node)
    elseif key == "B" or key == "b" then
        browse_and_mark(node)
        node:goto_menu("file_menu")
    elseif key == "D" or key == "d" then
        download_marked(node)
        node:goto_menu("file_menu")
    elseif key == "F" or key == "f" then
        download_file(node)
        node:goto_menu("file_menu")
    elseif key == "U" or key == "u" then
        upload_file(node)
        node:goto_menu("file_menu")
    elseif key == "S" or key == "s" then
        search_files(node)
        node:goto_menu("file_menu")
    elseif key == "Q" or key == "q" then
        node:goto_menu("main_menu")
    end
end

-- -----------------------------------------------------------------------
-- List and select file areas
-- -----------------------------------------------------------------------
function list_areas(node)
    local areas = files.areas()
    if areas == nil or #areas == 0 then
        node:sendln("\r\n  No file areas available.")
        node:pause()
        node:goto_menu("file_menu")
        return
    end

    node:sendln("")
    node:sendln("  #   Name                          Files")
    node:sendln("  --- ------------------------------ -----")
    for i, area in ipairs(areas) do
        local line = string.format("  %-3d %-30s %5d",
            area.id, area.name, area.files)
        node:sendln(line)
    end
    node:sendln("")

    local choice = node:ask("  Enter area # (or Q to cancel): ", 5)
    if choice == nil or choice == "" or string.upper(choice) == "Q" then
        node:goto_menu("file_menu")
        return
    end

    local area_id = tonumber(choice)
    if area_id then
        local area = files.get_area(area_id)
        if area then
            set_current_area(node, area_id, area.name)
            node:sendln("  Current area: " .. area.name)
            node:pause()
        else
            node:sendln("  Area not found.")
            node:pause()
        end
    end
    node:goto_menu("file_menu")
end

-- -----------------------------------------------------------------------
-- List files in the current area
-- -----------------------------------------------------------------------
function list_files(node, area_id, offset, limit, marked_set)
    offset = offset or 0
    limit = limit or 20
    local file_list = files.list(area_id, offset, limit)
    if file_list == nil or #file_list == 0 then
        node:sendln("\r\n  No files in this area.")
        node:pause()
        return
    end

    node:sendln("")
    node:sendln(string.format("  Files %d-%d", offset + 1, offset + #file_list))
    node:sendln("  M  #   Filename             Size      DLs  Date        Description")
    node:sendln("  -- --- -------------------- --------- ---- ----------  -------------------")
    for i, f in ipairs(file_list) do
        local mark = " "
        if marked_set and marked_set[f.id] then
            mark = "*"
        end
        local line = string.format("  %s  %-3d %-20s %9s %4d %10s  %s",
            mark,
            i,
            string.sub(f.filename, 1, 20),
            f.size_str,
            f.downloads,
            f.date,
            string.sub(f.description, 1, 19))
        node:sendln(line)
    end
    node:sendln("")
    return file_list
end

local function parse_selection(input, max)
    local picks = {}
    for token in string.gmatch(input, "[^,%s]+") do
        local dash = string.find(token, "-")
        if dash then
            local a = tonumber(string.sub(token, 1, dash - 1))
            local b = tonumber(string.sub(token, dash + 1))
            if a and b then
                if a > b then
                    a, b = b, a
                end
                for i = a, b do
                    if i >= 1 and i <= max then
                        picks[i] = true
                    end
                end
            end
        else
            local n = tonumber(token)
            if n and n >= 1 and n <= max then
                picks[n] = true
            end
        end
    end
    return picks
end

function browse_and_mark(node)
    local area_id = get_or_default_area(node)
    if not area_id then
        node:sendln("\r\n  No file areas available.")
        node:pause()
        return
    end

    local offset = 0
    local limit = 20

    while true do
        node:cls()
        local marked_set, marked_order = get_marked_ids(node)
        local file_list = list_files(node, area_id, offset, limit, marked_set)
        if file_list == nil or #file_list == 0 then
            return
        end

        node:sendln("")
        node:sendln("  Marked files: " .. tostring(#marked_order))
        node:sendln("  [#] Toggle  [N]ext  [P]rev  [D]ownload marked  [C]lear  [Q]uit")

        local choice = node:ask("  Selection: ", 20)
        if choice == nil or choice == "" then
            return
        end

        local upper = string.upper(choice)
        if upper == "Q" then
            return
        elseif upper == "N" then
            if #file_list == limit then
                offset = offset + limit
            else
                node:sendln("  End of list.")
                node:pause()
            end
        elseif upper == "P" then
            if offset >= limit then
                offset = offset - limit
            else
                offset = 0
            end
        elseif upper == "D" then
            download_marked(node)
        elseif upper == "C" then
            save_id_list(node, "download_basket", {})
            node:sendln("  Cleared marked files.")
            node:pause()
        else
            local picks = parse_selection(choice, #file_list)
            if next(picks) == nil then
                node:sendln("  Invalid selection.")
                node:pause()
            else
                for idx, _ in pairs(picks) do
                    local f = file_list[idx]
                    if f then
                        if marked_set[f.id] then
                            marked_set[f.id] = nil
                            for i, id in ipairs(marked_order) do
                                if id == f.id then
                                    table.remove(marked_order, i)
                                    break
                                end
                            end
                        else
                            marked_set[f.id] = true
                            table.insert(marked_order, f.id)
                        end
                    end
                end
                save_id_list(node, "download_basket", marked_order)
            end
        end
    end
end

-- -----------------------------------------------------------------------
-- Download a file via ZMODEM-8K
-- -----------------------------------------------------------------------
function download_file(node)
    local area_id = get_or_default_area(node)
    if not area_id then
        node:sendln("\r\n  No file areas available.")
        node:pause()
        return
    end

    if not require_transfer(node) then
        return
    end

    if not transfer.available() then
        node:sendln("\r\n  File transfer is not available (SEXYZ not found).")
        node:pause()
        return
    end

    -- Show file listing and let user pick one
    local file_list = list_files(node, area_id)
    if file_list == nil or #file_list == 0 then
        return
    end

    local choice = node:ask("  Enter file # to download (or Q to cancel): ", 5)
    if choice == nil or choice == "" or string.upper(choice) == "Q" then
        return
    end

    local idx = tonumber(choice)
    if not idx or idx < 1 or idx > #file_list then
        node:sendln("  Invalid selection.")
        node:pause()
        return
    end

    local f = file_list[idx]
    local area = files.get_area(area_id)
    if not area then
        node:sendln("  Error: could not load area information.")
        node:pause()
        return
    end

    -- Build the full path to the file on disk
    local filepath = area.path .. "/" .. f.filename

    node:sendln("")
    node:sendln("  Sending: " .. f.filename .. " (" .. f.size_str .. ")")
    node:sendln("  Protocol: ZMODEM-8K")
    node:sendln("")
    node:sendln("  Start your ZMODEM download now...")
    node:sendln("")

    local ok, err = transfer.send(filepath)
    if ok then
        node:sendln("\r\n  Transfer complete!")
        files.increment_download(f.id)
    else
        node:sendln("\r\n  Transfer failed: " .. (err or "unknown error"))
    end
    node:pause()
end

function download_marked(node)
    local marked_set, marked_order = get_marked_ids(node)
    if marked_order == nil or #marked_order == 0 then
        node:sendln("\r\n  No files marked for download.")
        node:pause()
        return
    end

    if not require_transfer(node) then
        return
    end

    if not transfer.available() then
        node:sendln("\r\n  File transfer is not available (SEXYZ not found).")
        node:pause()
        return
    end

    local paths = {}
    local entries = {}
    for _, file_id in ipairs(marked_order) do
        local f = files.get_file(file_id)
        if f then
            local area = files.get_area(f.area_id)
            if area and area.path then
                table.insert(paths, area.path .. "/" .. f.filename)
                table.insert(entries, f)
            end
        end
    end

    if #paths == 0 then
        node:sendln("\r\n  No valid files found for download.")
        node:pause()
        return
    end

    node:sendln("")
    node:sendln("  Preparing to send " .. tostring(#paths) .. " file(s):")
    for _, f in ipairs(entries) do
        node:sendln("  " .. f.filename .. " (" .. f.size_str .. ")")
    end
    node:sendln("")
    node:sendln("  Protocol: ZMODEM-8K")
    node:sendln("  Start your ZMODEM download now...")
    node:sendln("")

    local ok, err = transfer.send(paths)
    if ok then
        node:sendln("\r\n  Transfer complete!")
        for _, f in ipairs(entries) do
            files.increment_download(f.id)
        end
        save_id_list(node, "download_basket", {})
    else
        node:sendln("\r\n  Transfer failed: " .. (err or "unknown error"))
    end
    node:pause()
end

-- -----------------------------------------------------------------------
-- Upload a file via ZMODEM-8K
-- -----------------------------------------------------------------------
function upload_file(node)
    local area_id = get_or_default_area(node)
    if not area_id then
        node:sendln("\r\n  No file areas available.")
        node:pause()
        return
    end

    if not require_transfer(node) then
        return
    end

    if not transfer.available() then
        node:sendln("\r\n  File transfer is not available (SEXYZ not found).")
        node:pause()
        return
    end

    local area = files.get_area(area_id)
    if not area then
        node:sendln("\r\n  Error: could not load area information.")
        node:pause()
        return
    end

    node:sendln("")
    node:sendln("  Upload to: " .. area.name)
    node:sendln("  Protocol: ZMODEM-8K")
    node:sendln("")
    node:sendln("  Start your ZMODEM upload now...")
    node:sendln("")

    local received, err = transfer.receive(area.path)
    if not received or #received == 0 then
        node:sendln("\r\n  No files received." .. (err and (" (" .. err .. ")") or ""))
        node:pause()
        return
    end

    -- Catalog each received file
    node:sendln("\r\n  Received " .. #received .. " file(s):")
    node:sendln("")
    for _, rf in ipairs(received) do
        node:sendln("  " .. rf.name .. " (" .. rf.size .. " bytes)")
        local desc = node:ask("  Description: ", 60)
        if desc == nil then desc = "" end

        local id, add_err = files.add_entry(area_id, rf.name, desc, rf.size)
        if id then
            node:sendln("  Cataloged successfully.")
        else
            node:sendln("  Error cataloging: " .. (add_err or "unknown"))
        end
        node:sendln("")
    end

    node:sendln("  Upload complete!")
    node:pause()
end

-- -----------------------------------------------------------------------
-- Search files across all areas
-- -----------------------------------------------------------------------
function search_files(node)
    node:sendln("")
    local pattern = node:ask("  Search for: ", 40)
    if pattern == nil or pattern == "" then
        return
    end

    local results = files.search(pattern)
    if results == nil or #results == 0 then
        node:sendln("  No files found matching '" .. pattern .. "'.")
        node:pause()
        return
    end

    node:sendln("")
    node:sendln("  Found " .. #results .. " file(s):")
    node:sendln("")
    node:sendln("  Filename             Size      Description")
    node:sendln("  -------------------- --------- ----------------------------")
    for _, f in ipairs(results) do
        local line = string.format("  %-20s %9s %s",
            string.sub(f.filename, 1, 20),
            f.size_str,
            string.sub(f.description, 1, 28))
        node:sendln(line)
    end
    node:sendln("")
    node:pause()
end

return menu
