package search

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/internal/models"
	"Rail-Ticket-Notifier/internal/notifier"
	"Rail-Ticket-Notifier/utils/constants"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var httpClient *http.Client

func init() {
	// Custom DNS resolver using Google DNS — fixes Android/Termux where /etc/resolv.conf is missing
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}
	dialer := &net.Dialer{
		Timeout:  10 * time.Second,
		Resolver: resolver,
	}
	transport := &http.Transport{
		DialContext: dialer.DialContext,
	}
	httpClient = &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
	}
}

// LoadAuthFromFile reads auth credentials from a JSON file.
// If path is empty, defaults to "auth.json" in the current directory.
func LoadAuthFromFile(path string) (*models.CapturedAuth, error) {
	if path == "" {
		path = "auth.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth file %s: %w", path, err)
	}
	var auth models.CapturedAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, fmt.Errorf("failed to parse auth file: %w", err)
	}
	if auth.Authorization == "" {
		return nil, fmt.Errorf("auth file missing authorization token")
	}
	return &auth, nil
}

// SearchTrainsAPI calls the railway API directly
func SearchTrainsAPI(auth *models.CapturedAuth, from, to, date, seatClass string) (*models.TrainResponse, error) {
	url := fmt.Sprintf(
		"https://railspaapi.shohoz.com/v1.0/web/bookings/search-trips-v2?from_city=%s&to_city=%s&date_of_journey=%s&seat_class=%s",
		from, to, date, seatClass,
	)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", auth.Authorization)
	req.Header.Set("X-Device-Id", auth.DeviceId)
	req.Header.Set("X-Device-Key", auth.DeviceKey)
	req.Header.Set("User-Agent", auth.UserAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://eticket.railway.gov.bd/")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result models.TrainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// FindAvailableSeats checks for seats matching criteria, respecting both seat type AND train priority order
func FindAvailableSeats(trains *models.TrainResponse, targetTrains []string, targetSeatTypes []string, minSeats uint) (string, string, int) {
	for _, targetTrain := range targetTrains {
		for _, targetType := range targetSeatTypes {
			for _, train := range trains.Data.Trains {
				if !strings.Contains(train.TripNumber, targetTrain) {
					continue
				}
				for _, seat := range train.SeatTypes {
					if seat.Type == targetType && seat.SeatCounts.Online >= int(minSeats) {
						return train.TripNumber, seat.Type, seat.SeatCounts.Online
					}
				}
			}
		}
	}
	return "", "", 0
}

// PerformSearch runs the search loop using direct API calls until seats are found.
func PerformSearch(authFilePath string) (string, bool) {
	rand.Seed(time.Now().UnixNano())

	log.Println("Loading auth credentials from file...")
	auth, err := LoadAuthFromFile(authFilePath)
	if err != nil {
		log.Fatal("Failed to load auth:", err)
	}
	log.Println("Auth loaded successfully.")

	searchAltUrl := strings.EqualFold(arguments.FROM, "Dhaka")
	if searchAltUrl {
		log.Println("Alt search (Biman Bandar) enabled")
	}

	attemptNo := 0

	for {
		attemptNo++
		log.Printf("Search attempt %d...", attemptNo)

		currentFrom := arguments.FROM
		if searchAltUrl && attemptNo%2 == 0 {
			currentFrom = "Biman Bandar"
		}

		trains, err := SearchTrainsAPI(auth, currentFrom, arguments.TO, arguments.DATE, arguments.SEAT_TYPE_ARRAY[0])
		if err != nil {
			log.Printf("API error: %v", err)

			if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
				log.Println("Auth token expired! Please update auth.json and restart.")
				return "", false
			}

			time.Sleep(getRandomDelay())
			continue
		}

		log.Printf("Found %d trains", len(trains.Data.Trains))
		for _, t := range trains.Data.Trains {
			for _, s := range t.SeatTypes {
				if s.SeatCounts.Online > 0 {
					log.Printf("  %s - %s: %d seats", t.TripNumber, s.Type, s.SeatCounts.Online)
				}
			}
		}

		trainName, seatType, seatCount := FindAvailableSeats(
			trains,
			arguments.SPECIFIC_TRAIN_ARRAY,
			arguments.SEAT_TYPE_ARRAY,
			arguments.SEAT_COUNT,
		)

		if trainName != "" {
			log.Printf("SEAT FOUND: %s - %s - %d seats!", trainName, seatType, seatCount)
			messageBody := fmt.Sprintf("Train: %s\nClass: %s\nAvailable: %d seats\nFrom: %s\nTo: %s\nDate: %s\n",
				trainName, seatType, seatCount, arguments.FROM, arguments.TO, arguments.DATE)

			notifier.SendEmail(messageBody)
			notifier.MakeCall()

			return messageBody, true
		}

		randomDelay := getRandomDelay()
		log.Printf("No matching seats. Waiting %.0f seconds...", randomDelay.Seconds())
		time.Sleep(randomDelay)
	}
}

func getRandomDelay() time.Duration {
	min := constants.SEARCH_DELAY_MIN_SEC
	max := constants.SEARCH_DELAY_MAX_SEC
	delay := rand.Intn(max-min+1) + min
	return time.Duration(delay) * time.Second
}
