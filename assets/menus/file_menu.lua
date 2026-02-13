-- file_menu.lua - File areas menu
local menu = {}
local current_area = nil

function menu.on_key(node, key)
    if key == "L" or key == "l" then
        list_areas(node)
    elseif key == "D" or key == "d" then
        if current_area then
            list_files(node, current_area)
        else
            node:sendln("\r\n  Select an area first (press L).")
            node:pause()
        end
        node:goto_menu("file_menu")
    elseif key == "U" or key == "u" then
        node:sendln("\r\n  Upload functionality requires ZMODEM support.")
        node:sendln("  This will be available in a future update.")
        node:pause()
        node:goto_menu("file_menu")
    elseif key == "S" or key == "s" then
        search_files(node)
        node:goto_menu("file_menu")
    elseif key == "Q" or key == "q" then
        node:goto_menu("main_menu")
    end
end

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
            current_area = area_id
            node:sendln("  Current area: " .. area.name)
            node:pause()
        else
            node:sendln("  Area not found.")
            node:pause()
        end
    end
    node:goto_menu("file_menu")
end

function list_files(node, area_id)
    local file_list = files.list(area_id, 0, 20)
    if file_list == nil or #file_list == 0 then
        node:sendln("\r\n  No files in this area.")
        node:pause()
        return
    end

    node:sendln("")
    node:sendln("  Filename             Size      DLs  Date        Description")
    node:sendln("  -------------------- --------- ---- ----------  -------------------")
    for _, f in ipairs(file_list) do
        local line = string.format("  %-20s %9s %4d %10s  %s",
            string.sub(f.filename, 1, 20),
            f.size_str,
            f.downloads,
            f.date,
            string.sub(f.description, 1, 19))
        node:sendln(line)
    end
    node:sendln("")
    node:pause()
end

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
