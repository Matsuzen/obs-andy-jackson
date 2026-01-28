package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// SunriseSunsetResponse represents the response from sunrise-sunset.org API
type SunriseSunsetResponse struct {
	Results struct {
		Sunrise string `json:"sunrise"`
		Sunset  string `json:"sunset"`
	} `json:"results"`
	Status string `json:"status"`
}

// IPLocationResponse represents the response from ip-api.com
type IPLocationResponse struct {
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	City    string  `json:"city"`
	Region  string  `json:"regionName"`
	Country string  `json:"country"`
	Status  string  `json:"status"`
}

// NominatimResponse represents the response from OpenStreetMap Nominatim API
type NominatimResponse struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

// getLocationFromIP returns lat/lng based on the computer's IP address
func getLocationFromIP() (float64, float64, string, error) {
	resp, err := http.Get("http://ip-api.com/json/")
	if err != nil {
		return 0, 0, "", fmt.Errorf("failed to get IP location: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, "", fmt.Errorf("failed to read IP location response: %v", err)
	}

	var location IPLocationResponse
	if err := json.Unmarshal(body, &location); err != nil {
		return 0, 0, "", fmt.Errorf("failed to parse IP location: %v", err)
	}

	if location.Status != "success" {
		return 0, 0, "", fmt.Errorf("IP location lookup failed")
	}

	locationName := fmt.Sprintf("%s, %s", location.City, location.Region)
	return location.Lat, location.Lon, locationName, nil
}

// getLocationFromCity returns lat/lng for a given city name using OpenStreetMap Nominatim
func getLocationFromCity(city string) (float64, float64, error) {
	encodedCity := url.QueryEscape(city)
	apiURL := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json&limit=1", encodedCity)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "OBSLauncher/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to geocode city: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read geocode response: %v", err)
	}

	var results []NominatimResponse
	if err := json.Unmarshal(body, &results); err != nil {
		return 0, 0, fmt.Errorf("failed to parse geocode response: %v", err)
	}

	if len(results) == 0 {
		return 0, 0, fmt.Errorf("city not found: %s", city)
	}

	var lat, lng float64
	fmt.Sscanf(results[0].Lat, "%f", &lat)
	fmt.Sscanf(results[0].Lon, "%f", &lng)

	return lat, lng, nil
}

// SunTimes holds both sunrise and sunset times
type SunTimes struct {
	Sunrise time.Time
	Sunset  time.Time
}

// getSunTimes fetches both sunrise and sunset times for a given location and date
func getSunTimes(lat, lng float64, date time.Time) (*SunTimes, error) {
	dateStr := date.Format("2006-01-02")
	apiURL := fmt.Sprintf("https://api.sunrise-sunset.org/json?lat=%f&lng=%f&date=%s&formatted=0", lat, lng, dateStr)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sun times: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var sunResp SunriseSunsetResponse
	if err := json.Unmarshal(body, &sunResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if sunResp.Status != "OK" {
		return nil, fmt.Errorf("API returned status: %s", sunResp.Status)
	}

	sunriseUTC, err := time.Parse(time.RFC3339, sunResp.Results.Sunrise)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sunrise time: %v", err)
	}

	sunsetUTC, err := time.Parse(time.RFC3339, sunResp.Results.Sunset)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sunset time: %v", err)
	}

	return &SunTimes{
		Sunrise: sunriseUTC.Local(),
		Sunset:  sunsetUTC.Local(),
	}, nil
}
