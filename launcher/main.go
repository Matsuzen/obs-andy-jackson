package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func printUsage() {
	fmt.Println("OBS Stream Launcher")
	fmt.Println()
	fmt.Println("Usage: launcher <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  sunrise   Get sunrise time for a location")
	fmt.Println("  sunset    Get sunset time for a location")
	fmt.Println("  schedule  Schedule a YouTube stream and start OBS")
	fmt.Println()
	fmt.Println("Run 'launcher <command> -help' for more information on a command.")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "sunrise":
		cmdSunrise(os.Args[2:])
	case "sunset":
		cmdSunset(os.Args[2:])
	case "schedule":
		cmdSchedule(os.Args[2:])
	case "-help", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// cmdSunrise handles the sunrise subcommand
func cmdSunrise(args []string) {
	fs := flag.NewFlagSet("sunrise", flag.ExitOnError)
	city := fs.String("city", "", "City for lookup (e.g., 'San Bernardino, CA'). If not specified, uses IP geolocation")
	offset := fs.Int("offset", 0, "Minutes offset from sunrise")
	format := fs.String("format", "human", "Output format: 'human' or 'time' (just the time value)")
	fs.Parse(args)

	sunTimes, locationName := getSunTimesForLocation(*city)
	resultTime := sunTimes.Sunrise.Add(time.Duration(*offset) * time.Minute)

	if *format == "time" {
		fmt.Println(resultTime.Format("2006-01-02T15:04:05"))
	} else {
		fmt.Printf("Location: %s\n", locationName)
		fmt.Printf("Sunrise:  %s\n", sunTimes.Sunrise.Format("15:04:05"))
		if *offset != 0 {
			fmt.Printf("Offset:   %+d minutes\n", *offset)
			fmt.Printf("Result:   %s\n", resultTime.Format("15:04:05"))
		}
	}
}

// cmdSunset handles the sunset subcommand
func cmdSunset(args []string) {
	fs := flag.NewFlagSet("sunset", flag.ExitOnError)
	city := fs.String("city", "", "City for lookup (e.g., 'San Bernardino, CA'). If not specified, uses IP geolocation")
	offset := fs.Int("offset", 0, "Minutes offset from sunset")
	format := fs.String("format", "human", "Output format: 'human' or 'time' (just the time value)")
	fs.Parse(args)

	sunTimes, locationName := getSunTimesForLocation(*city)
	resultTime := sunTimes.Sunset.Add(time.Duration(*offset) * time.Minute)

	if *format == "time" {
		fmt.Println(resultTime.Format("2006-01-02T15:04:05"))
	} else {
		fmt.Printf("Location: %s\n", locationName)
		fmt.Printf("Sunset:   %s\n", sunTimes.Sunset.Format("15:04:05"))
		if *offset != 0 {
			fmt.Printf("Offset:   %+d minutes\n", *offset)
			fmt.Printf("Result:   %s\n", resultTime.Format("15:04:05"))
		}
	}
}

// getSunTimesForLocation is a helper that gets sun times for a city or IP location
func getSunTimesForLocation(city string) (*SunTimes, string) {
	var lat, lng float64
	var locationName string
	var err error

	if city != "" {
		lat, lng, err = getLocationFromCity(city)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting location for city: %v\n", err)
			os.Exit(1)
		}
		locationName = city
	} else {
		lat, lng, locationName, err = getLocationFromIP()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting location from IP: %v\n", err)
			os.Exit(1)
		}
	}

	sunTimes, err := getSunTimes(lat, lng, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting sun times: %v\n", err)
		os.Exit(1)
	}

	return sunTimes, locationName
}

// cmdSchedule handles the schedule subcommand
func cmdSchedule(args []string) {
	fs := flag.NewFlagSet("schedule", flag.ExitOnError)

	// Stream scheduling flags
	title := fs.String("title", "", "Stream title (default: 'Marshall WX (MM/DD/YYYY)')")
	scheduledTime := fs.String("time", "", "Scheduled time: 'SUNRISE', 'SUNSET', or specific time '2006-01-02T15:04:05'")
	description := fs.String("description", "", "Stream description")
	privacy := fs.String("privacy", "public", "Privacy status: public, unlisted, or private")

	// Sunrise/sunset related flags
	city := fs.String("city", "", "City for sunrise/sunset lookup (e.g., 'San Bernardino, CA')")
	offset := fs.Int("offset", -30, "Minutes offset from sunrise/sunset (default: -30)")

	// OBS flags
	obsPath := fs.String("obs-path", "", "Custom path to OBS executable")
	skipOBS := fs.Bool("skip-obs", false, "Skip starting OBS")

	// Credentials flag
	credentialsDir := fs.String("credentials-dir", "", "Directory containing credentials.json")

	fs.Parse(args)

	// Validate required flags
	if *scheduledTime == "" {
		fmt.Println("Usage: launcher schedule -time <SUNRISE|SUNSET|TIME> [options]")
		fmt.Println()
		fmt.Println("Required:")
		fmt.Println("  -time          'SUNRISE', 'SUNSET', or specific time '2006-01-02T15:04:05'")
		fmt.Println()
		fmt.Println("Optional:")
		fmt.Println("  -title         Stream title (default: 'Marshall WX (MM/DD/YYYY)')")
		fmt.Println("  -description   Stream description")
		fmt.Println("  -privacy       Privacy: public, unlisted, private (default: public)")
		fmt.Println()
		fmt.Println("Sun time options (when -time SUNRISE or SUNSET):")
		fmt.Println("  -city          City for lookup (default: auto-detect via IP)")
		fmt.Println("  -offset        Minutes offset (default: -30)")
		fmt.Println()
		fmt.Println("Other options:")
		fmt.Println("  -obs-path         Custom path to OBS executable")
		fmt.Println("  -skip-obs         Skip starting OBS")
		fmt.Println("  -credentials-dir  Directory containing YouTube credentials")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  launcher schedule -time SUNRISE")
		fmt.Println("  launcher schedule -time SUNRISE -city \"San Bernardino, CA\"")
		fmt.Println("  launcher schedule -time SUNSET -offset -45")
		fmt.Println("  launcher schedule -time 2026-01-25T07:00:00 -title \"My Stream\"")
		os.Exit(1)
	}

	fmt.Println("=== OBS Stream Scheduler ===")
	fmt.Println()

	// Get the executable's directory for credentials
	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}
	baseDir := filepath.Dir(execPath)

	// Set credentials directory
	credDir := *credentialsDir
	if credDir == "" {
		credDir = baseDir
	}

	// Determine the stream time
	var streamTime time.Time
	today := time.Now()
	timeUpper := strings.ToUpper(*scheduledTime)

	if timeUpper == "SUNRISE" || timeUpper == "SUNSET" {
		sunTimes, locationName := getSunTimesForLocation(*city)
		fmt.Printf("Location: %s\n", locationName)

		if timeUpper == "SUNRISE" {
			fmt.Printf("Sunrise: %s\n", sunTimes.Sunrise.Format("15:04:05"))
			streamTime = sunTimes.Sunrise.Add(time.Duration(*offset) * time.Minute)
		} else {
			fmt.Printf("Sunset: %s\n", sunTimes.Sunset.Format("15:04:05"))
			streamTime = sunTimes.Sunset.Add(time.Duration(*offset) * time.Minute)
		}
		fmt.Printf("Stream time (%s %+d min): %s\n", strings.ToLower(timeUpper), *offset, streamTime.Format("15:04:05"))
	} else {
		// Parse specific time
		streamTime, err = time.ParseInLocation("2006-01-02T15:04:05", *scheduledTime, time.Local)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid time format. Use 'SUNRISE', 'SUNSET', or 'YYYY-MM-DDTHH:MM:SS'\n")
			os.Exit(1)
		}
		fmt.Printf("Stream time: %s\n", streamTime.Format("2006-01-02 15:04:05"))
	}

	// Generate title if not provided
	streamTitle := *title
	if streamTitle == "" {
		streamTitle = fmt.Sprintf("Marshall WX (%s)", today.Format("01/02/2006"))
	}
	fmt.Printf("Title: %s\n", streamTitle)
	fmt.Println()

	// Initialize YouTube scheduler
	scheduler, err := NewStreamScheduler(credDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing YouTube scheduler: %v\n", err)
		os.Exit(1)
	}

	// Schedule the stream
	broadcast, _, err := scheduler.ScheduleStream(streamTitle, *description, streamTime, *privacy)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scheduling stream: %v\n", err)
		os.Exit(1)
	}

	// Start OBS
	if !*skipOBS {
		obsExe := *obsPath
		if obsExe == "" {
			obsExe = getOBSPath()
		}

		fmt.Printf("Starting OBS: %s\n", obsExe)

		obsCmd := exec.Command(obsExe, "--startstreaming")
		if err := obsCmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting OBS: %v\n", err)
			fmt.Fprintf(os.Stderr, "You may need to specify the OBS path with -obs-path\n")
			// Don't exit - continue with scheduler
		} else {
			fmt.Println("OBS started with streaming enabled")
		}
	}

	fmt.Println()
	fmt.Println("=== Setup Complete ===")
	fmt.Printf("Stream will go live at: %s\n", streamTime.Format("15:04:05"))
	fmt.Println("Press Ctrl+C to cancel")
	fmt.Println()

	// Wait and go live
	scheduler.WaitAndGoLive(streamTime, broadcast.Id)

	fmt.Println("Stream is now live!")
}

// getOBSPath returns the default OBS installation path for the current OS
func getOBSPath() string {
	switch runtime.GOOS {
	case "windows":
		return `C:\Program Files\obs-studio\bin\64bit\obs64.exe`
	case "darwin":
		return "/Applications/OBS.app/Contents/MacOS/OBS"
	default: // linux
		return "obs"
	}
}
