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

function menu.on_load(node)
    node:cls()
end

function menu.on_key(node, key)
    if key == "L" or key == "l" then
        list_areas(node)
    elseif key == "D" or key == "d" then
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
function list_files(node, area_id)
    local file_list = files.list(area_id, 0, 20)
    if file_list == nil or #file_list == 0 then
        node:sendln("\r\n  No files in this area.")
        node:pause()
        return
    end

    node:sendln("")
    node:sendln("  #   Filename             Size      DLs  Date        Description")
    node:sendln("  --- -------------------- --------- ---- ----------  -------------------")
    for i, f in ipairs(file_list) do
        local line = string.format("  %-3d %-20s %9s %4d %10s  %s",
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
