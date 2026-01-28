# YouTube Stream Scheduler Setup Guide

This guide will walk you through setting up the YouTube Data API and running the automated stream scheduler.

## Prerequisites

- Go 1.21 or later
- A Google account with a YouTube channel
- OBS or another streaming software configured to stream to YouTube

## Step 1: Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click **Select a project** → **New Project**
3. Name your project (e.g., "YouTube Stream Automation")
4. Click **Create**

## Step 2: Enable YouTube Data API v3

1. In your new project, go to **APIs & Services** → **Library**
2. Search for "YouTube Data API v3"
3. Click on it and click **Enable**

## Step 3: Configure OAuth Consent Screen

1. Go to **APIs & Services** → **OAuth consent screen**
2. Select **External** as the user type
3. Click **Create**
4. Fill in the required fields:
   - **App name**: YouTube Stream Scheduler
   - **User support email**: Your email
   - **Developer contact information**: Your email
5. Click **Save and Continue**
6. On the **Scopes** page, click **Add or Remove Scopes**
7. Filter for "YouTube Data API v3" and select:
   - `.../auth/youtube` (Manage your YouTube account)
8. Click **Update** → **Save and Continue**
9. On **Test users**, click **Add Users** and add your Google account email
10. Click **Save and Continue** → **Back to Dashboard**

## Step 4: Create OAuth 2.0 Credentials

1. Go to **APIs & Services** → **Credentials**
2. Click **Create Credentials** → **OAuth client ID**
3. Select **Desktop app** as the application type
4. Name it "YouTube Stream Scheduler Client"
5. Click **Create**
6. Click **Download JSON** on the popup (or click the download icon next to your credential)
7. Save the downloaded file as `credentials.json` in this directory

**Important:** Make sure the file is named exactly `credentials.json` and is in the same directory as the executable.

## Step 5: Build the Executable

```bash
# Download dependencies
go mod download

# Build for your current platform
go build -o youtube-stream-scheduler

# Or build for specific platforms:

# macOS (Intel)
GOOS=darwin GOARCH=amd64 go build -o youtube-stream-scheduler-mac-intel

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o youtube-stream-scheduler-mac-arm

# Windows
GOOS=windows GOARCH=amd64 go build -o youtube-stream-scheduler.exe

# Linux
GOOS=linux GOARCH=amd64 go build -o youtube-stream-scheduler-linux
```

## Step 6: Run the Scheduler

### Basic Usage

```bash
./youtube-stream-scheduler -title "<title>" -time "<scheduled_time>"
```

### Examples

**Schedule a stream for 8 PM today:**
```bash
./youtube-stream-scheduler -title "My Live Stream" -time "2026-01-25T20:00:00"
```

**Schedule a stream with description and privacy setting:**
```bash
./youtube-stream-scheduler -title "Gaming Session" -time "2026-01-26T15:30:00" -description "Playing Elden Ring" -privacy "unlisted"
```

**On Windows:**
```cmd
youtube-stream-scheduler.exe -title "My Live Stream" -time "2026-01-25T20:00:00"
```

### Command-Line Flags

- **-title** (required): The title of your stream
- **-time** (required): When to start the stream in format `YYYY-MM-DDTHH:MM:SS`
  - Uses 24-hour format
  - Uses your local timezone
  - Example: `2026-01-25T20:00:00` for 8 PM on January 25, 2026
- **-description** (optional): Stream description
- **-privacy** (optional): `public`, `unlisted`, or `private` (default: `public`)

## Step 7: First-Time Authentication

When you run the executable for the first time:

1. The program will display an authorization URL in the terminal
2. Copy the URL and paste it into your web browser
3. Log in with your Google account
4. Review the permissions and click **Continue**
5. You may see a warning that the app isn't verified - click **Continue** (this is safe because it's your own app)
6. Click **Allow** to grant permissions
7. Google will display an authorization code on the page
8. Copy the authorization code and paste it back into the terminal when prompted
9. The program will save your credentials to `youtube_token.json` for future use

**Note:** The entire OAuth flow happens in the terminal - no local web server needed. After the first authentication, you won't need to authenticate again unless the token expires.

## How It Works

The program performs these steps automatically:

1. **Authenticates** with YouTube API using OAuth 2.0
2. **Creates a live broadcast** with your specified title, description, and scheduled time
3. **Creates a live stream** and binds it to the broadcast
4. **Displays stream information** including:
   - YouTube Studio URL to configure your stream
   - RTMP stream key for OBS
   - RTMP URL for streaming
5. **Waits** until the scheduled time (shows countdown every 30 seconds)
6. **Automatically transitions** the broadcast from "testing" to "live" status at the scheduled time

## Configure OBS

After running the program, you'll receive RTMP streaming information. Configure OBS:

1. Go to **Settings** → **Stream**
2. Service: **Custom**
3. Server: Use the RTMP URL from the program output (everything before the stream key)
4. Stream Key: Use the stream key from the program output
5. Click **OK**
6. Start streaming in OBS before the scheduled time (the stream will be in preview mode)
7. The program will automatically press "Go Live" at the scheduled time

## Important Notes

- **Keep the program running**: The executable must remain running until the scheduled time to automatically go live
- **Start OBS early**: Begin streaming in OBS a few minutes before the scheduled time so YouTube can process the preview
- **Testing mode**: The stream will be in testing/preview mode until the scheduled time
- **Token security**: Keep `credentials.json` and `youtube_token.json` private - they are in `.gitignore`
- **Single executable**: You can copy the compiled executable to any computer - just make sure `credentials.json` is in the same directory

## Troubleshooting

### "unable to read credentials file"
- Make sure you downloaded the OAuth credentials and saved them as `credentials.json` in the same directory as the executable

### "Invalid grant" or authentication errors
- Delete `youtube_token.json` and run the program again to re-authenticate

### Stream doesn't go live
- Ensure OBS is actively streaming before the scheduled time
- Check YouTube Studio to see the broadcast status
- The stream must be receiving data from OBS for the transition to work

### API quota exceeded
- YouTube API has daily quotas. If exceeded, wait 24 hours or request a quota increase in Google Cloud Console

### "Broadcast already in testing or live mode"
- This is normal if you run the go-live command multiple times
- The program will still transition the stream to live

## Distribution

You can distribute the compiled executable to other computers. Just make sure:

1. The executable and `credentials.json` are in the same directory
2. The user runs the authentication flow once on their machine
3. After authentication, `youtube_token.json` will be created automatically

## Security Best Practices

The `.gitignore` file is already configured to exclude:
- `credentials.json`
- `youtube_token.json`
- All compiled binaries

Never commit your API credentials to version control.

## Additional Resources

- [YouTube Data API Documentation](https://developers.google.com/youtube/v3/live/docs)
- [YouTube Live Streaming API Guide](https://developers.google.com/youtube/v3/live/getting-started)
- [Google API Go Client](https://github.com/googleapis/google-api-go-client)
- [Google Cloud Console](https://console.cloud.google.com/)

## Quick Reference

**Build:**
```bash
go build -o youtube-stream-scheduler
```

**Run:**
```bash
./youtube-stream-scheduler -title "Stream Title" -time "2026-01-25T20:00:00"
```

**Full example:**
```bash
./youtube-stream-scheduler \
  -title "Live Coding Session" \
  -time "2026-01-26T19:00:00" \
  -description "Building a Go application" \
  -privacy "public"
```
