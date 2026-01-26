# OBS Automation Scripts

This repository contains automation scripts for OBS (Open Broadcaster Software).

## Projects

### 1. Weather Data Fetcher (Lua)

A Lua script for OBS that automatically fetches weather data from a URL and displays it in text sources.

**File:** `fetch_weather_data.lua`

**Features:**
- Fetches weather data from Arduino Weather Station
- Displays wind speed, gusts, and direction
- Shows formatted date/time
- Auto-updates at configurable intervals

See the [Weather Data section](#weather-data-details) below for detailed information.

---

### 2. YouTube Stream Scheduler (Go)

**NEW!** A Go application that automates YouTube live streaming by scheduling streams and automatically pressing "Go Live" at the specified time.

**Files:**
- `main.go` - Main application
- `go.mod` - Go dependencies
- `YOUTUBE_SETUP.md` - Complete setup guide

**Features:**
- Schedule YouTube live streams for specific times
- Automatically transition from preview to live
- Single executable - easy to distribute
- OAuth 2.0 authentication with YouTube API
- Command-line interface with flags

**Quick Start:**

1. Follow the setup guide in `YOUTUBE_SETUP.md` to configure YouTube API access
2. Build the executable:
   ```bash
   go build -o youtube-stream-scheduler
   ```
3. Run the scheduler:
   ```bash
   ./youtube-stream-scheduler -title "My Stream" -time "2026-01-25T20:00:00"
   ```

**See `YOUTUBE_SETUP.md` for complete documentation.**

---

## Weather Data Details

### Summary

Created a Lua script for OBS that automatically fetches weather data from a URL and displays it in two separate text sources.

### Solution

Created `fetch_weather_data.lua` - an OBS Lua script that:
1. Automatically generates URLs with today's date in YYYYMMDD format
2. Fetches weather data from `https://www.flymarshall.com/wx/betaTwo/wx{DATE}.dat`
3. Parses the CSV data format from Arduino Weather Station
4. Displays formatted data in two separate text sources

### Features

#### Two Text Sources

**Wind Data Source** displays:
```
{wind_speed} mph, {wind_gust} mph, {cardinal_direction}
```
Example: `13.0 mph, 18 mph, SW`

**Date/Time Source** displays:
```
YYYY/MM/DD HH:MM
```
Example: `2025/11/27 13:13`

#### Data Processing

- Fetches latest line from weather data file
- Parses CSV format based on [Arduino Weather Station data string composition](https://github.com/crestlinesoaring/ArduinoWeatherStation/wiki/Data-String-Composition)
- Extracts fields:
  - Field 1: Time (HH:MM)
  - Field 2: Date (M/D/YYYY)
  - Field 3: Wind Speed (mph)
  - Field 4: Wind Gust Max (5min, mph)
  - Field 5: Wind Direction (degrees)
- Converts wind direction from degrees to 16-point cardinal directions (N, NNE, NE, ENE, E, ESE, SE, SSE, S, SSW, SW, WSW, W, WNW, NW, NNW)
- Strips leading zeros from wind speed values
- Formats date/time to ISO-like format (YYYY/MM/DD HH:MM)

#### Auto-Update

- Configurable update interval (default: 60 seconds)
- Manual "Update Now" button for testing
- Automatic URL date generation using current system date

### Installation

1. Create two text sources in OBS:
   - One for wind data (e.g., "Wind Data")
   - One for date/time (e.g., "Date Time")

2. Add the script to OBS:
   - Go to **Tools → Scripts**
   - Click **+** button
   - Select `/Users/julien.renald/personal/obs/fetch_weather_data.lua`

3. Configure the script:
   - **Wind Data Text Source**: Select your wind data text source
   - **Date/Time Text Source**: Select your date/time text source
   - **Base URL**: `https://www.flymarshall.com/wx/betaTwo/wx` (default)
   - **URL Suffix**: `.dat` (default)
   - **Update Interval**: 60 seconds (configurable)
   - Click **"Update Now"** to test

### Technical Details

#### Current Implementation

- Uses `io.popen()` to execute `curl` command
- Parses CSV data with Lua pattern matching
- Updates OBS text sources directly via `obs_source_update()`
- Timer-based polling for automatic updates

#### Cardinal Direction Conversion

The `degrees_to_cardinal()` function converts degrees to 16-point compass directions:
- Each direction covers 22.5° (360° / 16 directions)
- Uses `+11.25` offset to center each direction range
  - Without offset: N would only cover 0° to 22.5° (asymmetric)
  - With offset: N covers 348.75° to 11.25° (centered at 0°)
- Example: 225° converts to "SW" (Southwest)

### Data Source

Weather data comes from [Crestline Soaring Arduino Weather Station](https://github.com/crestlinesoaring/ArduinoWeatherStation)
- CSV format with comma-separated fields
- Multiple readings per day
- Script uses only the last (most recent) line

---

## Security

The `.gitignore` file is configured to exclude:
- YouTube API credentials (`credentials.json`, `youtube_token.json`)
- Compiled binaries
- System files

**Never commit API credentials to version control.**

## Requirements

- **Weather Script**: OBS Studio with Lua scripting support
- **YouTube Scheduler**: Go 1.21 or later
