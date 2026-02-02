package main

import (
	"flag"
	"fmt"
	"launcher/internal/release"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const broadcastIDFile = "broadcast_id.txt"
const VERSION = "0.0.1"

func printUsage() {
	fmt.Println("OBS Stream Launcher")
	fmt.Println()
	fmt.Println("Usage: launcher <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  sunrise  Get sunrise time for a location")
	fmt.Println("  sunset   Get sunset time for a location")
	fmt.Println("  stream   Stream management commands")
	fmt.Println("  update   Update the CLI to the latest release")
	fmt.Println()
	fmt.Println("Run 'launcher <command> --help' for more information on a command.")
}

func printStreamUsage() {
	fmt.Println("Stream management commands")
	fmt.Println()
	fmt.Println("Usage: launcher stream <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  schedule  Create YouTube broadcast and schedule start/end tasks")
	fmt.Println("  start     Start OBS and transition broadcast to live")
	fmt.Println("  end       End the current broadcast")
	fmt.Println()
	fmt.Println("Run 'launcher stream <command> --help' for more information.")
}

func printFlagUsage(fs *flag.FlagSet, command string) {
	fmt.Printf("Usage: %s [options]\n\n", command)
	fmt.Println("Options:")
	fs.VisitAll(func(f *flag.Flag) {
		defaultVal := ""
		if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
			defaultVal = fmt.Sprintf(" (default: %s)", f.DefValue)
		}
		fmt.Printf("  --%-14s %s%s\n", f.Name, f.Usage, defaultVal)
	})
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
	case "stream":
		cmdStream(os.Args[2:])
	case "update":
		cmdUpdate(os.Args[2:])
	case "-help", "--help", "help":
		printUsage()
	case "-version", "--version", "version":
		fmt.Printf("OBS Stream Launcher version %s\n", VERSION)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// cmdStream handles the stream subcommand
func cmdStream(args []string) {
	if len(args) < 1 {
		printStreamUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "schedule":
		cmdStreamSchedule(args[1:])
	case "start":
		cmdStreamStart(args[1:])
	case "end":
		cmdStreamEnd(args[1:])
	case "-help", "--help", "help":
		printStreamUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown stream command: %s\n\n", args[0])
		printStreamUsage()
		os.Exit(1)
	}
}

// cmdSunrise handles the sunrise subcommand
func cmdSunrise(args []string) {
	fs := flag.NewFlagSet("sunrise", flag.ExitOnError)
	city := fs.String("city", "", "City for lookup (e.g., 'San Bernardino, CA'). If not specified, uses IP geolocation")
	offset := fs.Int("offset", 0, "Minutes offset from sunrise")
	format := fs.String("format", "human", "Output format: 'human', 'datetime' (ISO format), or 'time' (HH:MM)")
	fs.Usage = func() { printFlagUsage(fs, "launcher sunrise") }
	fs.Parse(args)

	sunTimes, locationName := getSunTimesForLocation(*city)
	resultTime := sunTimes.Sunrise.Add(time.Duration(*offset) * time.Minute)

	switch *format {
	case "datetime":
		fmt.Println(resultTime.Format("2006-01-02T15:04:05"))
	case "time":
		fmt.Println(resultTime.Format("15:04"))
	default:
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
	format := fs.String("format", "human", "Output format: 'human', 'datetime' (ISO format), or 'time' (HH:MM)")
	fs.Usage = func() { printFlagUsage(fs, "launcher sunset") }
	fs.Parse(args)

	sunTimes, locationName := getSunTimesForLocation(*city)
	resultTime := sunTimes.Sunset.Add(time.Duration(*offset) * time.Minute)

	switch *format {
	case "datetime":
		fmt.Println(resultTime.Format("2006-01-02T15:04:05"))
	case "time":
		fmt.Println(resultTime.Format("15:04"))
	default:
		fmt.Printf("Location: %s\n", locationName)
		fmt.Printf("Sunset:   %s\n", sunTimes.Sunset.Format("15:04:05"))
		if *offset != 0 {
			fmt.Printf("Offset:   %+d minutes\n", *offset)
			fmt.Printf("Result:   %s\n", resultTime.Format("15:04:05"))
		}
	}
}

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

func cmdUpdate(_ []string) {
	updater := release.NewUpdater(VERSION)
	latestRelease, err := updater.GetLatestRelease()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	err = updater.Apply(latestRelease)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func cmdStreamSchedule(args []string) {
	fs := flag.NewFlagSet("stream schedule", flag.ExitOnError)

	title := fs.String("title", "", "Stream title (default: 'Marshall WX (MM/DD/YYYY)')")
	description := fs.String("description", "", "Stream description")
	privacy := fs.String("privacy", "public", "Privacy status: public, unlisted, or private")

	city := fs.String("city", "", "City for sunrise/sunset lookup")
	startTimeFlag := fs.String("time", "SUNRISE", "Start time: 'SUNRISE', 'SUNSET', or specific time 'YYYY-MM-DDTHH:MM:SS'")
	startOffset := fs.Int("start-offset", -30, "Minutes offset from sunrise/sunset for start")
	endOffset := fs.Int("end-offset", 30, "Minutes offset from sunset for end")

	fs.Usage = func() { printFlagUsage(fs, "launcher stream schedule") }
	fs.Parse(args)

	fmt.Println("=== Stream Scheduler ===")
	fmt.Println()

	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}
	baseDir := filepath.Dir(execPath)

	// Determine start time
	var startTime time.Time
	var endTime time.Time
	timeUpper := strings.ToUpper(*startTimeFlag)

	if timeUpper == "SUNRISE" || timeUpper == "SUNSET" {
		sunTimes, locationName := getSunTimesForLocation(*city)
		fmt.Printf("Location: %s\n", locationName)
		fmt.Printf("Sunrise:  %s\n", sunTimes.Sunrise.Format("15:04:05"))
		fmt.Printf("Sunset:   %s\n", sunTimes.Sunset.Format("15:04:05"))

		if timeUpper == "SUNRISE" {
			startTime = sunTimes.Sunrise.Add(time.Duration(*startOffset) * time.Minute)
			fmt.Printf("Stream start (sunrise %+d min): %s\n", *startOffset, startTime.Format("15:04:05"))
		} else {
			startTime = sunTimes.Sunset.Add(time.Duration(*startOffset) * time.Minute)
			fmt.Printf("Stream start (sunset %+d min): %s\n", *startOffset, startTime.Format("15:04:05"))
		}

		endTime = sunTimes.Sunset.Add(time.Duration(*endOffset) * time.Minute)
		fmt.Printf("Stream end (sunset %+d min): %s\n", *endOffset, endTime.Format("15:04:05"))
	} else {
		var err error
		startTime, err = time.ParseInLocation("2006-01-02T15:04:05", *startTimeFlag, time.Local)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Invalid time format. Use 'SUNRISE', 'SUNSET', or 'YYYY-MM-DDTHH:MM:SS'\n")
			os.Exit(1)
		}
		fmt.Printf("Stream start: %s\n", startTime.Format("2006-01-02 15:04:05"))

		// Still use sunset for end time
		sunTimes, locationName := getSunTimesForLocation(*city)
		fmt.Printf("Location: %s\n", locationName)
		endTime = sunTimes.Sunset.Add(time.Duration(*endOffset) * time.Minute)
		fmt.Printf("Stream end (sunset %+d min): %s\n", *endOffset, endTime.Format("15:04:05"))
	}
	fmt.Println()

	today := time.Now()
	streamTitle := *title
	if streamTitle == "" {
		streamTitle = fmt.Sprintf("Marshall WX (%s)", today.Format("01/02/2006"))
	}
	fmt.Printf("Title: %s\n", streamTitle)
	fmt.Println()

	scheduler, err := NewStreamScheduler(baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing YouTube scheduler: %v\n", err)
		os.Exit(1)
	}

	broadcast, _, err := scheduler.ScheduleStream(streamTitle, *description, startTime, *privacy)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scheduling stream: %v\n", err)
		os.Exit(1)
	}

	bidFile := filepath.Join(baseDir, broadcastIDFile)
	if err := os.WriteFile(bidFile, []byte(broadcast.Id), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not save broadcast ID to file: %v\n", err)
	} else {
		fmt.Printf("Broadcast ID saved to: %s\n", bidFile)
	}

	startCmd := fmt.Sprintf(`"%s" stream start -id "%s"`, execPath, broadcast.Id)
	if err := createScheduledTask("StartYouTubeStream", startCmd, startTime); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating start task: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Scheduled start task for: %s\n", startTime.Format("15:04"))

	endCmd := fmt.Sprintf(`"%s" stream end -id "%s"`, execPath, broadcast.Id)
	if err := createScheduledTask("EndYouTubeStream", endCmd, endTime); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating end task: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Scheduled end task for: %s\n", endTime.Format("15:04"))

	fmt.Println()
	fmt.Println("=== Schedule Complete ===")
	fmt.Println("The stream will automatically start and end at the scheduled times.")
}

func cmdStreamStart(args []string) {
	fs := flag.NewFlagSet("stream start", flag.ExitOnError)

	broadcastID := fs.String("id", "", "Broadcast ID to start (default: read from broadcast_id.txt)")
	obsPath := fs.String("obs-path", "", "Custom path to OBS executable")
	skipOBS := fs.Bool("skip-obs", false, "Skip starting OBS")

	fs.Usage = func() { printFlagUsage(fs, "launcher stream start") }
	fs.Parse(args)

	fmt.Println("=== Starting Stream ===")
	fmt.Println()

	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}
	baseDir := filepath.Dir(execPath)

	bid := *broadcastID
	if bid == "" {
		bidFile := filepath.Join(baseDir, broadcastIDFile)
		data, err := os.ReadFile(bidFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: No broadcast ID provided and could not read %s: %v\n", bidFile, err)
			os.Exit(1)
		}
		bid = strings.TrimSpace(string(data))
	}

	if bid == "" {
		fmt.Fprintf(os.Stderr, "Error: Broadcast ID is empty\n")
		os.Exit(1)
	}

	fmt.Printf("Broadcast ID: %s\n", bid)

	if !*skipOBS {
		obsExe := *obsPath
		if obsExe == "" {
			obsExe = getOBSPath()
		}

		fmt.Printf("Starting OBS in directory: %s\n", obsExe)

		obsCmd := exec.Command(obsExe, "--startstreaming")
		obsCmd.Dir = filepath.Dir(obsExe)
		if err := obsCmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting OBS: %v\n", err)
		} else {
			fmt.Println("OBS started with streaming enabled")
			// This sleep time here makes sure that OBS has enough time to initialize before transitioning the stream to live.
			time.Sleep(30 * time.Second)
		}
	}

	scheduler, err := NewStreamScheduler(baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing YouTube scheduler: %v\n", err)
		os.Exit(1)
	}

	if err := scheduler.GoLive(bid); err != nil {
		fmt.Fprintf(os.Stderr, "Error transitioning to live: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("=== Stream is Live ===")
}

func cmdStreamEnd(args []string) {
	fs := flag.NewFlagSet("stream end", flag.ExitOnError)
	broadcastID := fs.String("id", "", "Broadcast ID to end (default: read from broadcast_id.txt)")
	fs.Usage = func() { printFlagUsage(fs, "launcher stream end") }
	fs.Parse(args)

	fmt.Println("=== Ending Stream ===")
	fmt.Println()

	execPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting executable path: %v\n", err)
		os.Exit(1)
	}
	baseDir := filepath.Dir(execPath)

	bid := *broadcastID
	if bid == "" {
		bidFile := filepath.Join(baseDir, broadcastIDFile)
		data, err := os.ReadFile(bidFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: No broadcast ID provided and could not read %s: %v\n", bidFile, err)
			os.Exit(1)
		}
		bid = strings.TrimSpace(string(data))
	}

	if bid == "" {
		fmt.Fprintf(os.Stderr, "Error: Broadcast ID is empty\n")
		os.Exit(1)
	}

	fmt.Printf("Broadcast ID: %s\n", bid)

	scheduler, err := NewStreamScheduler(baseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing YouTube scheduler: %v\n", err)
		os.Exit(1)
	}

	if err := scheduler.EndStream(bid); err != nil {
		fmt.Fprintf(os.Stderr, "Error ending stream: %v\n", err)
		os.Exit(1)
	}
}

func createScheduledTask(taskName, command string, runTime time.Time) error {
	switch runtime.GOOS {
	case "windows":
		return createWindowsTask(taskName, command, runTime)
	default:
		return createUnixTask(taskName, command, runTime)
	}
}

func createWindowsTask(taskName, command string, runTime time.Time) error {
	timeStr := runTime.Format("15:04")

	checkCmd := exec.Command("schtasks", "/query", "/tn", taskName)
	if err := checkCmd.Run(); err == nil {
		deleteCmd := exec.Command("schtasks", "/delete", "/tn", taskName, "/f")
		if err := deleteCmd.Run(); err != nil {
			return fmt.Errorf("failed to delete task: %v", err)
		}
	}
	createCmd := exec.Command("schtasks", "/create",
		"/tn", taskName,
		"/tr", command,
		"/sc", "once",
		"/st", timeStr,
		"/f",
	)
	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create task: %v", err)
	}
	return nil
}

func createUnixTask(taskName, command string, runTime time.Time) error {
	minute := runTime.Minute()
	hour := runTime.Hour()
	day := runTime.Day()
	month := int(runTime.Month())
	cronEntry := fmt.Sprintf("%d %d %d %d * %s # TASK:%s", minute, hour, day, month, command, taskName)

	getCurrentCmd := exec.Command("crontab", "-l")
	currentCrontab, _ := getCurrentCmd.Output()

	var newLines []string
	for _, line := range strings.Split(string(currentCrontab), "\n") {
		if !strings.Contains(line, fmt.Sprintf("# TASK:%s", taskName)) && line != "" {
			newLines = append(newLines, line)
		}
	}
	newLines = append(newLines, cronEntry)

	newCrontab := strings.Join(newLines, "\n") + "\n"
	setCrontabCmd := exec.Command("crontab", "-")
	setCrontabCmd.Stdin = strings.NewReader(newCrontab)
	if err := setCrontabCmd.Run(); err != nil {
		return fmt.Errorf("failed to update crontab: %v", err)
	}

	return nil
}

// Returns the path then the actual program
// Windows will throw some errors if the program is launched outside of the executable's directory
func getOBSPath() string {
	switch runtime.GOOS {
	case "windows":
		return `C:\Program Files\obs-studio\bin\64bit\obs64.exe`
	case "darwin":
		return "/Applications/OBS.app/Contents/MacOS/OBS"
	default:
		return "obs"
	}
}
