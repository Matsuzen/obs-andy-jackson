obs = obslua

-- Script settings
local wind_source_name = ""
local datetime_source_name = ""
local update_interval = 60 -- seconds
local base_url = "https://www.flymarshall.com/wx/betaTwo/wx"
local url_suffix = ".dat"

local begin_at = 6 * 60 + 30 -- 6:30 AM in minutes
local end_at = 17 * 60 + 30 -- 5:30 PM in minutes

-- Timer flag
local last_update = 0

function script_description()
    return "Fetches weather data from URL and displays it in two text sources.\n\nOne source shows wind data, the other shows time/date.\nThe URL is automatically updated with today's date in YYYYMMDD format."
end

function script_properties()
    local props = obs.obs_properties_create()

    -- Wind data text source selection dropdown
    local p_wind = obs.obs_properties_add_list(props, "wind_source", "Wind Data Text Source",
        obs.OBS_COMBO_TYPE_EDITABLE, obs.OBS_COMBO_FORMAT_STRING)

    -- Date/Time text source selection dropdown
    local p_datetime = obs.obs_properties_add_list(props, "datetime_source", "Date/Time Text Source",
        obs.OBS_COMBO_TYPE_EDITABLE, obs.OBS_COMBO_FORMAT_STRING)

    local sources = obs.obs_enum_sources()
    if sources ~= nil then
        for _, source in ipairs(sources) do
            local source_id = obs.obs_source_get_unversioned_id(source)
            -- Add text sources (GDI+ text on Windows, FreeType 2 on Mac/Linux)
            if source_id == "text_gdiplus" or source_id == "text_ft2_source" or source_id == "text_gdiplus_v2" then
                local name = obs.obs_source_get_name(source)
                obs.obs_property_list_add_string(p_wind, name, name)
                obs.obs_property_list_add_string(p_datetime, name, name)
            end
        end
    end
    obs.source_list_release(sources)

    obs.obs_properties_add_text(props, "base_url", "Base URL", obs.OBS_TEXT_DEFAULT)

    obs.obs_properties_add_text(props, "url_suffix", "URL Suffix", obs.OBS_TEXT_DEFAULT)

    obs.obs_properties_add_int(props, "update_interval", "Update Interval (seconds)", 10, 3600, 1)

    obs.obs_properties_add_button(props, "update_now", "Update Now", update_now_clicked)

    return props
end

function script_defaults(settings)
    obs.obs_data_set_default_string(settings, "base_url", base_url)
    obs.obs_data_set_default_string(settings, "url_suffix", url_suffix)
    obs.obs_data_set_default_int(settings, "update_interval", update_interval)
end

function script_update(settings)
    wind_source_name = obs.obs_data_get_string(settings, "wind_source")
    datetime_source_name = obs.obs_data_get_string(settings, "datetime_source")
    base_url = obs.obs_data_get_string(settings, "base_url")
    url_suffix = obs.obs_data_get_string(settings, "url_suffix")
    update_interval = obs.obs_data_get_int(settings, "update_interval")

    if wind_source_name ~= "" or datetime_source_name ~= "" then
        fetch_weather_data()
    end
end

function script_load(settings)
    obs.timer_add(check_update, 1000) -- Check every second
end

function check_update()
    local current_time = os.time()
    if current_time - last_update >= update_interval then
        fetch_weather_data()
        last_update = current_time
    end
end

function update_now_clicked(props, p)
    fetch_weather_data()
    return true
end

function degrees_to_cardinal(degrees)
    local deg = tonumber(degrees)
    if not deg then
        return "N/A"
    end

    -- Normalize to 0-360
    deg = deg % 360

    local directions = {
        "N", "NNE", "NE", "ENE",
        "E", "ESE", "SE", "SSE",
        "S", "SSW", "SW", "WSW",
        "W", "WNW", "NW", "NNW"
    }

    -- Each direction covers 22.5 degrees (360/16)
    -- Add 11.25 to center the ranges, then divide by 22.5
    local index = math.floor((deg + 11.25) / 22.5) % 16 + 1

    return directions[index]
end

function fetch_weather_data()
    if wind_source_name == "" and datetime_source_name == "" then
        print("No text sources selected")
        return
    end

    local date_str = os.date("%Y%m%d")

    local url = base_url .. date_str .. url_suffix

    print("Fetching data from: " .. url)

    local curl_command = string.format('curl -s "%s"', url)
    local handle = io.popen(curl_command)

    if handle then
        local data = handle:read("*a")
        handle:close()

        if data and data ~= "" then
            local last_line = ""
            for line in data:gmatch("[^\r\n]+") do
                if line ~= "" then
                    last_line = line
                end
            end

            local fields = {}
            for field in last_line:gmatch("[^,]+") do
                table.insert(fields, field)
            end

            local current_minutes = tonumber(os.date("%H")) * 60 + tonumber(os.date("%M"))
            local offline = current_minutes < begin_at or current_minutes > end_at

            local wind_output = ""
            if offline == true then
                wind_output = "-- mph, -- mph, -- "
            else
                local wind_speed = fields[3] or "N/A"
                local wind_gust = fields[4] or "N/A"
                local wind_dir = fields[5] or "N/A"
                wind_output = string.format(
                    "%s mph, %s mph, %s",
                    string.gsub(wind_speed, "^0", ""), string.gsub(wind_gust, "^0", ""), degrees_to_cardinal(wind_dir)
                )
            end

            local datetime_output = os.date("%Y/%m/%d %H:%M")
            if offline == true then
                datetime_output = datetime_output .. " (offline)"
            end

            if wind_source_name ~= "" then
                update_text_source(wind_source_name, wind_output)
            end

            if datetime_source_name ~= "" then
                update_text_source(datetime_source_name, datetime_output)
            end
        else
            local error_msg = "Error: Could not fetch data from " .. url
            if wind_source_name ~= "" then
                update_text_source(wind_source_name, error_msg)
            end
            if datetime_source_name ~= "" then
                update_text_source(datetime_source_name, error_msg)
            end
            print(error_msg)
        end
    else
        local error_msg = "Error: Could not execute curl command"
        if wind_source_name ~= "" then
            update_text_source(wind_source_name, error_msg)
        end
        if datetime_source_name ~= "" then
            update_text_source(datetime_source_name, error_msg)
        end
        print(error_msg)
    end
end

function update_text_source(source_name, text)
    local source = obs.obs_get_source_by_name(source_name)
    if source ~= nil then
        local settings = obs.obs_source_get_settings(source)

        obs.obs_data_set_string(settings, "text", text)

        obs.obs_source_update(source, settings)

        obs.obs_data_release(settings)
        obs.obs_source_release(source)
    else
        print("Text source '" .. source_name .. "' not found")
    end
end

function script_unload()
    obs.timer_remove(check_update)
end
