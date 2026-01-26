package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

const (
	credentialsFile = "credentials.json"
	tokenFile       = "youtube_token.json"
)

type StreamScheduler struct {
	service *youtube.Service
}

// Retrieve a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) (*http.Client, error) {
	tokFile := tokenFile
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok, err = getTokenFromWeb(config)
		if err != nil {
			return nil, err
		}
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok), nil
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	fmt.Println("\n" + string(make([]byte, 80)))
	for i := 0; i < 80; i++ {
		fmt.Print("=")
	}
	fmt.Println()
	fmt.Println("🔐 AUTHORIZATION REQUIRED")
	fmt.Println(string(make([]byte, 80)))
	for i := 0; i < 80; i++ {
		fmt.Print("=")
	}
	fmt.Println("\n")
	fmt.Println("Step 1: Visit this URL in your browser:")
	fmt.Printf("\n%s\n\n", authURL)
	fmt.Println("Step 2: After authorizing, Google will display an authorization code.")
	fmt.Println("Step 3: Copy the code and paste it below.")
	fmt.Println()
	fmt.Print("Enter authorization code: ")

	var authCode string
	fmt.Scanln(&authCode)

	tok, err := config.Exchange(context.Background(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token: %v", err)
	}

	fmt.Println("\n✅ Authentication successful!")
	return tok, nil
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("✅ Saving credential file to: %s\n", path)
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// Open browser based on OS
// Initialize YouTube service
func NewStreamScheduler() (*StreamScheduler, error) {
	ctx := context.Background()

	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file: %v\nPlease follow setup instructions in YOUTUBE_SETUP.md", err)
	}

	config, err := google.ConfigFromJSON(b, youtube.YoutubeScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials file: %v", err)
	}

	client, err := getClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %v", err)
	}

	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create YouTube service: %v", err)
	}

	fmt.Println("✅ Authorized with YouTube API")

	return &StreamScheduler{service: service}, nil
}

// Schedule a live stream
func (s *StreamScheduler) ScheduleStream(title, description string, scheduledTime time.Time, privacy string) (*youtube.LiveBroadcast, *youtube.LiveStream, error) {
	fmt.Println("📅 Scheduling live stream...")
	fmt.Printf("   Title: %s\n", title)
	fmt.Printf("   Scheduled for: %s\n", scheduledTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("   Privacy: %s\n\n", privacy)

	// Create the live broadcast
	broadcast := &youtube.LiveBroadcast{
		Snippet: &youtube.LiveBroadcastSnippet{
			Title:              title,
			Description:        description,
			ScheduledStartTime: scheduledTime.Format(time.RFC3339),
		},
		ContentDetails: &youtube.LiveBroadcastContentDetails{
			EnableAutoStart: false,
			// Double check if this needs to be true or not
			// If this is false, the stream won't stop when there is internet problems and the stream cuts
			EnableAutoStop: false,
		},
		Status: &youtube.LiveBroadcastStatus{
			PrivacyStatus:           privacy,
			SelfDeclaredMadeForKids: false,
		},
	}

	broadcastCall := s.service.LiveBroadcasts.Insert([]string{"snippet", "contentDetails", "status"}, broadcast)
	broadcastResponse, err := broadcastCall.Do()
	if err != nil {
		return nil, nil, fmt.Errorf("error creating broadcast: %v", err)
	}

	fmt.Printf("✅ Broadcast created with ID: %s\n", broadcastResponse.Id)

	stream := &youtube.LiveStream{
		Snippet: &youtube.LiveStreamSnippet{
			Title: fmt.Sprintf("%s - Stream", title),
		},
		Cdn: &youtube.CdnSettings{
			FrameRate:     "variable",
			IngestionType: "rtmp",
			Resolution:    "variable",
		},
	}

	streamCall := s.service.LiveStreams.Insert([]string{"snippet", "cdn"}, stream)
	streamResponse, err := streamCall.Do()
	if err != nil {
		return nil, nil, fmt.Errorf("error creating stream: %v", err)
	}

	fmt.Printf("✅ Stream created with ID: %s\n", streamResponse.Id)

	bindCall := s.service.LiveBroadcasts.Bind(broadcastResponse.Id, []string{"id", "contentDetails"})
	bindCall.StreamId(streamResponse.Id)
	_, err = bindCall.Do()
	if err != nil {
		return nil, nil, fmt.Errorf("error binding broadcast to stream: %v", err)
	}

	fmt.Println("✅ Broadcast bound to stream")

	// Display stream information
	fmt.Println("Stream Information:")
	fmt.Printf("Studio URL: https://studio.youtube.com/video/%s/livestreaming\n", broadcastResponse.Id)
	fmt.Printf("Watch URL: https://youtube.com/watch?v=%s\n", broadcastResponse.Id)
	fmt.Printf("Stream Key: %s\n", streamResponse.Cdn.IngestionInfo.StreamName)
	fmt.Printf("RTMP URL: %s/%s\n\n", streamResponse.Cdn.IngestionInfo.IngestionAddress, streamResponse.Cdn.IngestionInfo.StreamName)

	return broadcastResponse, streamResponse, nil
}

// Transition broadcast to live
func (s *StreamScheduler) GoLive(broadcastID string) error {
	fmt.Println("Transitioning broadcast to LIVE...")

	testingCall := s.service.LiveBroadcasts.Transition("testing", broadcastID, []string{"status"})
	_, err := testingCall.Do()
	if err != nil {
		fmt.Println("ℹ️ Broadcast already in testing or live mode")
	} else {
		fmt.Println("✅ Broadcast in testing mode")
		time.Sleep(2 * time.Second)
	}

	liveCall := s.service.LiveBroadcasts.Transition("live", broadcastID, []string{"status"})
	_, err = liveCall.Do()
	if err != nil {
		return fmt.Errorf("error transitioning to live: %v", err)
	}

	fmt.Println("✅ Broadcast is now LIVE!")
	fmt.Printf("   Watch at: https://youtube.com/watch?v=%s\n\n", broadcastID)

	return nil
}

func (s *StreamScheduler) WaitAndGoLive(scheduledTime time.Time, broadcastID string) {
	now := time.Now()
	duration := scheduledTime.Sub(now)

	if duration <= 0 {
		fmt.Println("⚠️  Scheduled time is in the past. Going live immediately...")
		if err := s.GoLive(broadcastID); err != nil {
			log.Fatalf("Error going live: %v", err)
		}
		return
	}

	minutes := int(duration.Minutes())
	seconds := int(duration.Seconds()) % 60

	fmt.Printf("⏰ Waiting %d minutes and %d seconds until scheduled start time...\n", minutes, seconds)
	fmt.Printf("   Will go live at: %s\n\n", scheduledTime.Format("2006-01-02 15:04:05"))

	// Show countdown
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	done := time.After(duration)

	for {
		select {
		case <-done:
			fmt.Println("\n Scheduled time reached")
			if err := s.GoLive(broadcastID); err != nil {
				log.Fatalf("Error going live: %v", err)
			}
			return
		case <-ticker.C:
			remaining := scheduledTime.Sub(time.Now())
			if remaining > 0 {
				mins := int(remaining.Minutes())
				secs := int(remaining.Seconds()) % 60
				fmt.Printf("⏱️  Time remaining: %d minutes %d seconds\n", mins, secs)
			}
		}
	}
}

func main() {
	// Command line flags
	title := flag.String("title", "", "Stream title (required)")
	scheduledTime := flag.String("time", "", "Scheduled start time in format '2006-01-02T15:04:05' (required)")
	description := flag.String("description", "", "Stream description (optional)")
	privacy := flag.String("privacy", "public", "Privacy status: public, unlisted, or private")

	flag.Parse()

	// Print header
	fmt.Println("🎥 YouTube Stream Scheduler")
	fmt.Println(string(make([]byte, 50)))
	for i := 0; i < 50; i++ {
		fmt.Print("=")
	}
	fmt.Println("")

	// Validate required flags
	if *title == "" || *scheduledTime == "" {
		fmt.Println("Usage: youtube-stream-scheduler -title \"<title>\" -time \"<scheduled_time>\" [-description \"<desc>\"] [-privacy <public|unlisted|private>]")
		fmt.Println("\nExamples:")
		fmt.Println("  ./youtube-stream-scheduler -title \"My Live Stream\" -time \"2026-01-25T20:00:00\"")
		fmt.Println("  ./youtube-stream-scheduler -title \"Gaming Session\" -time \"2026-01-26T15:30:00\" -description \"Playing Elden Ring\" -privacy \"unlisted\"")
		fmt.Println("\nScheduled time format: YYYY-MM-DDTHH:MM:SS (24-hour format, local timezone)")
		fmt.Println("Privacy options: public, unlisted, private (default: public)")
		os.Exit(1)
	}

	// Parse scheduled time
	parsedTime, err := time.ParseInLocation("2006-01-02T15:04:05", *scheduledTime, time.Local)
	if err != nil {
		log.Fatalf("❌ Error: Invalid time format. Use YYYY-MM-DDTHH:MM:SS (example: 2026-01-25T20:00:00)\n")
	}

	// Initialize scheduler
	scheduler, err := NewStreamScheduler()
	if err != nil {
		log.Fatalf("❌ Error initializing scheduler: %v\n", err)
	}

	// Schedule the stream
	broadcast, _, err := scheduler.ScheduleStream(*title, *description, parsedTime, *privacy)
	if err != nil {
		log.Fatalf("❌ Error scheduling stream: %v\n", err)
	}

	// Wait and go live
	fmt.Println("✅ Script will continue running until stream goes live...")
	fmt.Println("   Press Ctrl+C to cancel")

	scheduler.WaitAndGoLive(parsedTime, broadcast.Id)

	fmt.Println("✅ Stream is now live! You can close this program.")
}
