package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

const (
	credentialsFile = "credentials.json"
	tokenFile       = "youtube_token.json"
	youtubeStreamTitle    = "Marshall Weather Station - Stream" // This differs from the Broadcast title!
)

type StreamScheduler struct {
	service     *youtube.Service
	credentialsDir string
}

func getClient(config *oauth2.Config, credentialsDir string) (*http.Client, error) {
	tokFile := filepath.Join(credentialsDir, tokenFile)
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

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println("AUTHORIZATION REQUIRED")
	fmt.Println("================================================================================")
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

	fmt.Println("\nAuthentication successful!")
	return tok, nil
}

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

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.Create(path)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func NewStreamScheduler(credentialsDir string) (*StreamScheduler, error) {
	ctx := context.Background()

	credPath := filepath.Join(credentialsDir, credentialsFile)
	b, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file (%s): %v\nPlease ensure credentials.json exists", credPath, err)
	}

	config, err := google.ConfigFromJSON(b, youtube.YoutubeScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials file: %v", err)
	}

	client, err := getClient(config, credentialsDir)
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %v", err)
	}

	service, err := youtube.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create YouTube service: %v", err)
	}

	fmt.Println("Authorized with YouTube API")

	return &StreamScheduler{service: service, credentialsDir: credentialsDir}, nil
}

func (s *StreamScheduler) ScheduleStream(title, description string, scheduledTime time.Time, privacy string) (*youtube.LiveBroadcast, *youtube.LiveStream, error) {
	fmt.Println("Scheduling live stream...")
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
			EnableAutoStop:  false,
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

	fmt.Printf("Broadcast created with ID: %s\n", broadcastResponse.Id)

	streamListCall := s.service.LiveStreams.List([]string{"snippet", "cdn"})
	streamListResponse, err := streamListCall.Mine(true).Do()
	if err != nil {
		return nil, nil, fmt.Errorf("error listing streams: %v", err)
	}

	var stream *youtube.LiveStream

	for _, streamItem := range streamListResponse.Items {
		if streamItem.Snippet.Title == youtubeStreamTitle {
			stream = streamItem
			break
		}
	}

	if stream == nil {
		newStream := &youtube.LiveStream{
			Snippet: &youtube.LiveStreamSnippet{
				Title: youtubeStreamTitle,
			},
			Cdn: &youtube.CdnSettings{
				FrameRate:     "variable",
				IngestionType: "rtmp",
				Resolution:    "variable",
			},
		}
		streamCall := s.service.LiveStreams.Insert([]string{"snippet", "cdn"}, newStream)
		streamResponse, err := streamCall.Do()
		if err != nil {
			return nil, nil, fmt.Errorf("error creating new stream: %v", err)
		}
		stream = streamResponse
	}

	bindCall := s.service.LiveBroadcasts.Bind(broadcastResponse.Id, []string{"id", "contentDetails"}).StreamId(stream.Id)
	_, err = bindCall.Do()
	if err != nil {
		return nil, nil, fmt.Errorf("error binding broadcast to stream: %v", err)
	}
	fmt.Printf("Stream bound with ID: %s, Title: %s\n", stream.Id, stream.Snippet.Title)

	// Display stream information
	fmt.Println()
	fmt.Println("Stream Information:")
	fmt.Printf("  Studio URL: https://studio.youtube.com/video/%s/livestreaming\n", broadcastResponse.Id)
	fmt.Printf("  Watch URL: https://youtube.com/watch?v=%s\n", broadcastResponse.Id)
	fmt.Printf("  Stream Key: %s\n", stream.Cdn.IngestionInfo.StreamName)
	fmt.Printf("  RTMP URL: %s/%s\n", stream.Cdn.IngestionInfo.IngestionAddress, stream.Cdn.IngestionInfo.StreamName)
	fmt.Println()

	return broadcastResponse, stream, nil
}

// GoLive transitions the broadcast to live
func (s *StreamScheduler) GoLive(broadcastID string) error {
	fmt.Println("Transitioning broadcast to LIVE...")

	testingCall := s.service.LiveBroadcasts.Transition("testing", broadcastID, []string{"status"})
	_, err := testingCall.Do()
	if err != nil {
		fmt.Println("Broadcast already in testing or live mode")
	} else {
		fmt.Println("Broadcast in testing mode")
		time.Sleep(2 * time.Second)
	}

	liveCall := s.service.LiveBroadcasts.Transition("live", broadcastID, []string{"status"})
	_, err = liveCall.Do()
	if err != nil {
		return fmt.Errorf("error transitioning to live: %v", err)
	}

	fmt.Println("Broadcast is now LIVE!")
	fmt.Printf("  Watch at: https://youtube.com/watch?v=%s\n\n", broadcastID)

	return nil
}

func (s *StreamScheduler) WaitAndGoLive(scheduledTime time.Time, broadcastID string) {
	now := time.Now()
	duration := scheduledTime.Sub(now)

	if duration <= 0 {
		fmt.Println("Scheduled time is in the past. Going live immediately...")
		if err := s.GoLive(broadcastID); err != nil {
			log.Fatalf("Error going live: %v", err)
		}
		return
	}

	minutes := int(duration.Minutes())
	seconds := int(duration.Seconds()) % 60

	fmt.Printf("Waiting %d minutes and %d seconds until scheduled start time...\n", minutes, seconds)
	fmt.Printf("  Will go live at: %s\n\n", scheduledTime.Format("2006-01-02 15:04:05"))

	// Show countdown
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	done := time.After(duration)

	for {
		select {
		case <-done:
			fmt.Println("\nScheduled time reached!")
			if err := s.GoLive(broadcastID); err != nil {
				log.Fatalf("Error going live: %v", err)
			}
			return
		case <-ticker.C:
			remaining := scheduledTime.Sub(time.Now())
			if remaining > 0 {
				mins := int(remaining.Minutes())
				secs := int(remaining.Seconds()) % 60
				fmt.Printf("Time remaining: %d minutes %d seconds\n", mins, secs)
			}
		}
	}
}
