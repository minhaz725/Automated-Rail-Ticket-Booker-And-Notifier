package search

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/internal/notifier"
	"Rail-Ticket-Notifier/utils/constants"
	"context"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type CapturedAuth struct {
	Authorization string
	DeviceId      string
	DeviceKey     string
	UserAgent     string
}

type TrainResponse struct {
	Data struct {
		Trains []Train `json:"trains"`
	} `json:"data"`
}

type Train struct {
	TripNumber    string     `json:"trip_number"`
	DepartureTime string     `json:"departure_date_time"`
	ArrivalTime   string     `json:"arrival_date_time"`
	SeatTypes     []SeatType `json:"seat_types"`
}

type SeatType struct {
	Type       string `json:"type"`
	Fare       string `json:"fare"`
	SeatCounts struct {
		Online  int `json:"online"`
		Offline int `json:"offline"`
	} `json:"seat_counts"`
}

func CaptureAuthFromBrowser(searchUrl string) (*CapturedAuth, error) {
	allocCtx, cancel := chromedp.NewRemoteAllocator(context.Background(), "http://127.0.0.1:9222")
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	auth := &CapturedAuth{}

	chromedp.ListenTarget(ctx, func(ev interface{}) {
		if e, ok := ev.(*network.EventRequestWillBeSent); ok {
			if strings.Contains(e.Request.URL, "railspaapi.shohoz.com") {
				for k, v := range e.Request.Headers {
					if str, ok := v.(string); ok {
						switch k {
						case "Authorization":
							auth.Authorization = str
						case "X-Device-Id":
							auth.DeviceId = str
						case "X-Device-Key":
							auth.DeviceKey = str
						case "User-Agent":
							auth.UserAgent = str
						}
					}
				}
			}
		}
	})

	err := chromedp.Run(ctx, network.Enable())
	if err != nil {
		return nil, err
	}

	// Trigger one API call to capture headers
	err = chromedp.Run(ctx,
		chromedp.Navigate(searchUrl),
		chromedp.WaitReady("body"),
		chromedp.Sleep(10*time.Second),
	)
	if err != nil {
		return nil, err
	}

	if auth.Authorization == "" {
		return nil, fmt.Errorf("failed to capture auth token")
	}
	log.Println("Auth captured! Navigating to home...")
	chromedp.Run(ctx, chromedp.Navigate("https://eticket.railway.gov.bd"))
	return auth, nil
}

// SearchTrainsAPI calls the API directly
func SearchTrainsAPI(auth *CapturedAuth, from, to, date, seatClass string) (*TrainResponse, error) {
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

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result TrainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// FindAvailableSeats checks for seats matching criteria
func FindAvailableSeats(trains *TrainResponse, targetTrains []string, targetSeatTypes []string, minSeats uint) (string, string, int) {
	for _, train := range trains.Data.Trains {
		for _, targetTrain := range targetTrains {
			if !strings.Contains(train.TripNumber, targetTrain) {
				continue
			}
			for _, seat := range train.SeatTypes {
				for _, targetType := range targetSeatTypes {
					if seat.Type == targetType && seat.SeatCounts.Online >= int(minSeats) {
						return train.TripNumber, seat.Type, seat.SeatCounts.Online
					}
				}
			}
		}
	}
	return "", "", 0
}

func bookSeatsInExistingTab(ctx context.Context, trainName, seatClass, searchUrl string, messageBody *string) bool {
	log.Println("SEATS FOUND!")

	// Send notifications FIRST
	*messageBody += fmt.Sprintf("\nURL: %s\n", searchUrl)
	notifier.SendEmail(*messageBody)
	notifier.MakeCall()

	log.Println("Opening booking URL in a fresh browser...")

	// Open in default browser (NOT controlled by CDP)
	var cmd *exec.Cmd
	fmt.Println(searchUrl)
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", searchUrl)
	case "darwin":
		cmd = exec.Command("open", searchUrl)
	case "linux":
		cmd = exec.Command("xdg-open", searchUrl)
	}
	err := cmd.Start()
	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}

	log.Println("==========================================")
	log.Println("BOOK NOW! URL opened in your browser.")
	log.Println("==========================================")

	// Keep searching in case this one fails
	return false // Return false to continue searching
}

// ========== MAIN SEARCH ==========

func PerformSearch(originalUrl string, seatBookerFunction string) (string, bool) {
	_ = seatBookerFunction
	rand.Seed(time.Now().UnixNano())

	messageBody := ""

	// Setup browser context - KEEP THIS OPEN FOR THE ENTIRE SESSION
	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), constants.DEBUG_CHROME_URL)
	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)

	// Cleanup handler
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Received exit signal. Cleaning up...")
		chromedp.Run(browserCtx, chromedp.Evaluate(`
			if (typeof window.stopSeatHolding === 'function') {
				window.stopSeatHolding();
			}
		`, nil))
		cancelBrowser()
		cancelAlloc()
		os.Exit(0)
	}()

	// Step 1: Capture auth from browser
	log.Println("Capturing auth from browser...")
	auth, err := CaptureAuthFromBrowser(originalUrl)
	if err != nil {
		log.Fatal("Failed to capture auth:", err)
	}

	// Generate alt URL if needed
	altUrl := ""
	searchAltUrl := false
	if strings.EqualFold(arguments.FROM, "Dhaka") {
		altUrl = arguments.GenerateAltURL()
		searchAltUrl = true
		log.Println("Alt search (Biman Bandar) enabled")
	}

	attemptNo := 0

	// Step 2: Search loop using direct API
	for {
		log.Printf("Search attempt %d...", attemptNo+1)

		currentFrom := arguments.FROM
		if searchAltUrl && attemptNo%2 == 1 {
			currentFrom = "Biman Bandar"
			_ = altUrl
		}

		trains, err := SearchTrainsAPI(auth, currentFrom, arguments.TO, arguments.DATE, arguments.SEAT_TYPE_ARRAY[0])
		if err != nil {
			log.Printf("API error: %v", err)

			if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
				log.Println("Token expired, recapturing...")
				auth, err = CaptureAuthFromBrowser(originalUrl)
				if err != nil {
					log.Printf("Failed to recapture auth: %v", err)
				}
			}

			attemptNo++
			time.Sleep(getRandomDelay())
			continue
		}

		// Log available trains
		log.Printf("Found %d trains", len(trains.Data.Trains))
		for _, t := range trains.Data.Trains {
			for _, s := range t.SeatTypes {
				if s.SeatCounts.Online > 0 {
					log.Printf("  %s - %s: %d seats", t.TripNumber, s.Type, s.SeatCounts.Online)
				}
			}
		}

		// Check for matching seats
		trainName, seatType, seatCount := FindAvailableSeats(
			trains,
			arguments.SPECIFIC_TRAIN_ARRAY,
			arguments.SEAT_TYPE_ARRAY,
			arguments.SEAT_COUNT,
		)

		if trainName != "" {
			log.Printf("SEAT FOUND: %s - %s - %d seats!", trainName, seatType, seatCount)
			messageBody = fmt.Sprintf("Train: %s\nClass: %s\nAvailable: %d seats\n", trainName, seatType, seatCount)

			// Step 3: Use the SAME browser context to book
			success := bookSeatsInExistingTab(browserCtx, trainName, seatType, originalUrl, &messageBody)
			if success {
				return messageBody, true
			}
			log.Println("Booking failed, continuing search...")
		}

		attemptNo++
		randomDelay := getRandomDelay()
		log.Printf("Waiting %.0f seconds...", randomDelay.Seconds())
		time.Sleep(randomDelay)
	}
}

// openBrowserAndBook handles the actual booking via ChromeDP
func openBrowserAndBook(trainName, seatClass, url string, messageBody *string) bool {
	log.Println("Opening browser for booking...")

	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), constants.DEBUG_CHROME_URL)
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Navigate to search page
	err := chromedp.Run(ctx,
		emulation.SetUserAgentOverride(getUserAgent()),
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		log.Printf("Navigate error: %v", err)
		return false
	}

	// Wait for page to load results
	log.Println("Waiting for page results...")
	time.Sleep(10 * time.Second)

	// Run seat holding JS
	holdDuration := 1
	jsCode := buildSeatHoldingJS(trainName, seatClass, holdDuration)

	var success bool
	err = chromedp.Run(ctx,
		chromedp.Evaluate(jsCode, &success),
		chromedp.Sleep(2*time.Second),
	)
	if err != nil {
		log.Printf("Seat holding JS error: %v", err)
		return false
	}

	log.Println("Seat holding initiated!")

	// Wait for seat selection
	time.Sleep(5 * time.Second)

	// Get selected seat details
	var seatDetails struct {
		Coach     string   `json:"coach"`
		SeatClass string   `json:"seatClass"`
		Seats     []string `json:"seats"`
	}
	chromedp.Run(ctx, chromedp.Evaluate(`window.selectedSeatDetails || {coach: '', seatClass: '', seats: []}`, &seatDetails))

	if len(seatDetails.Seats) < int(arguments.SEAT_COUNT) {
		log.Printf("Only %d seats selected, need %d. Aborting.", len(seatDetails.Seats), arguments.SEAT_COUNT)
		return false
	}

	*messageBody += fmt.Sprintf("Coach: %s\nSeats: %s\n", seatDetails.Coach, strings.Join(seatDetails.Seats, ", "))

	// Send notification
	log.Println("Sending notification...")
	notifier.SendEmail(*messageBody)
	notifier.MakeCall()

	// Keep browser open for manual completion
	log.Println("Seat holding active. Complete booking manually.")
	log.Println("Press Ctrl+C to exit.")

	// Wait indefinitely
	select {}
}

// ========== HELPERS ==========

func getUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
	return userAgents[rand.Intn(len(userAgents))]
}

func getRandomDelay() time.Duration {
	min := constants.SEARCH_DELAY_MIN_SEC
	max := constants.SEARCH_DELAY_MAX_SEC
	delay := rand.Intn(max-min+1) + min
	return time.Duration(delay) * time.Second
}

func buildSeatHoldingJS(trainName, selectedClass string, holdDurationMinutes int) string {
	return `(() => {
   	console.log("Starting seat holding process...");
   	let heldSeats = [];
   	let holdInterval;
   	let isHolding = false;

   	// Function to find and track selected seats
   	function trackSelectedSeats() {
   		console.log("Tracking selected seats...");
   		const selectedSeatElements = document.querySelectorAll('.btn-seat.seat-selected, [class*="seat"][class*="selected"]');
   		heldSeats = [];
   		
   		selectedSeatElements.forEach(seat => {
   			const seatId = seat.getAttribute('title') || seat.getAttribute('data-seat') || seat.textContent;
   			if (seatId) {
   				heldSeats.push({
   					element: seat,
   					id: seatId,
   					selector: generateSelectorForSeat(seat)
   				});
   			}
   		});
   		
   		console.log("Tracked seats:", heldSeats.map(s => s.id));
   		return heldSeats.length > 0;
   	}

   	// Generate a reliable selector for a seat element
   	function generateSelectorForSeat(seatElement) {
   		const title = seatElement.getAttribute('title');
   		const className = seatElement.className;
   		
   		if (title) {
   			return '[title="' + title + '"]';
   		} else if (className) {
   			return '.' + className.split(' ').join('.');
   		} else {
   			return null;
   		}
   	}

   	// Function to click seats (hold or unhold)
   	function toggleSeats(hold = true) {
   		let successCount = 0;
   		console.log((hold ? "Holding" : "Unholding") + " seats...");
   		
   		heldSeats.forEach(seatInfo => {
   			try {
   				let seatElement = document.querySelector(seatInfo.selector);
   				
   				if (seatElement && !seatElement.disabled) {
   					seatElement.click();
   					successCount++;
   					console.log((hold ? 'Held' : 'Unheld') + ' seat: ' + seatInfo.id);
   				} else {
   					console.warn('Seat not found or disabled: ' + seatInfo.id);
   				}
   			} catch (e) {
   				console.error('Error toggling seat ' + seatInfo.id + ':', e);
   			}
   		});
   		
   		console.log("Successfully toggled " + successCount + " seats");
   		return successCount;
   	}

   	// Main seat holding cycle
   	function startHoldCycle() {
   		if (isHolding) {
   			console.log("Holding cycle already active");
   			return;
   		}
   		
   		isHolding = true;
   		console.log('Starting seat hold cycle. Hold duration: ' + ` + fmt.Sprintf("%d", holdDurationMinutes) + ` + ' minutes');
   		
   		const cycleInterval = ` + fmt.Sprintf("%d", holdDurationMinutes) + ` * 60 * 1000; // Convert to milliseconds
   		console.log("Cycle interval:", cycleInterval, "ms");
   		
   		holdInterval = setInterval(() => {
   			console.log("=== SEAT CYCLE START ===");
   			console.log("Step 1: Unholding seats...");
   			toggleSeats(false); // Unhold seats
   			
   			setTimeout(() => {
   				console.log("Step 2: Reholding seats...");
   				toggleSeats(true); // Re-hold seats immediately
   				console.log("=== SEAT CYCLE COMPLETE ===");
   			}, 1000); // 1000ms delay between unhold and rehold
   			
   		}, cycleInterval);
   		
   		console.log("Hold interval ID:", holdInterval);
   		
   		// Set up stop mechanism
   		window.stopSeatHolding = () => {
   			console.log("Stopping seat holding...");
   			if (holdInterval) {
   				clearInterval(holdInterval);
   				console.log("Interval cleared");
   			}
   			isHolding = false;
   			
   			// Final unhold
   			toggleSeats(false);
   			console.log("All seats released. Seat holding stopped.");
   		};
   		
   		console.log("Seat holding started. Call window.stopSeatHolding() to stop.");
   	}

   	// Original seat selection logic
   	try {
   		console.log("Looking for train:", "` + trainName + `");
   		const headers = Array.from(document.querySelectorAll("h2"));
   		const header = headers.find((h) => h.innerText.includes("` + trainName + `"));
   		
   		if (!header) throw new Error("Header not found");
   		console.log("Found train header");
   		
   		const appSingleTrip = header.closest("app-single-trip");
   		if (!appSingleTrip) throw new Error("Parent component not found");
   		console.log("Found parent component");

   		const seatClassDivs = Array.from(appSingleTrip.querySelectorAll(".single-seat-class"));
   		let seatDiv = seatClassDivs.find((div) => {
   			let seatNameSpan = div.querySelector(".seat-class-name");
   			return seatNameSpan && seatNameSpan.innerText.trim() === "` + selectedClass + `";
   		});

   		if (!seatDiv) throw new Error("Seat class div not found for: " + "` + selectedClass + `");
   		console.log("Found seat class div for:", "` + selectedClass + `");

   		const bookNowBtn = seatDiv.querySelector(".book-now-btn-wrapper .book-now-btn");
   		if (!bookNowBtn) throw new Error("Book now button not found");

   		console.log("Clicking book now button...");
   		bookNowBtn.click();

   		// Wait for bogie selection
   		const waitForInterface = new Promise((resolve) => {
   			setTimeout(() => {
   				console.log("Looking for bogie selection...");
   				const bogieSelection = document.getElementById("select-bogie");
   				if (bogieSelection) {
   					console.log("Found bogie selection");
   					const extractNumber = (text) => {
   						// Extract the number between " - " and " Seat(s)"
   						const match = text.match(/ - (\d+) Seat\(s\)/);
   						return match ? parseInt(match[1]) : 0;
   					};

   					const options = Array.from(bogieSelection.options);
   					const highestOption = options.reduce((highest, current) => {
   						const highestNumber = extractNumber(highest.text);
   						const currentNumber = extractNumber(current.text);
   						return currentNumber > highestNumber ? current : highest;
   					}, options[0]);

   					const coachWithHighestSeat = highestOption.text.split(" - ")[0];
   					console.log("Selected coach:", coachWithHighestSeat);
   					
   					const coachOption = Array.from(bogieSelection.options).find((option) =>
   						option.text.includes(coachWithHighestSeat)
   					);

   					bogieSelection.value = coachOption.value;
   					bogieSelection.dispatchEvent(new Event("change", { bubbles: true }));
   					// Handle XTR coach popup
   					if (coachWithHighestSeat && coachWithHighestSeat.startsWith("XTR")) {
   						setTimeout(() => {
   							// Try to find and click the OKAY button in the popup
   							const okBtn = Array.from(document.querySelectorAll("button, input[type='button']"))
   								.find(el => el.textContent.trim().toUpperCase() === "OKAY");
   							if (okBtn) {
   								okBtn.click();
   								console.log("XTR coach popup OKAY clicked");
   							} else {
   								console.warn("XTR coach popup OKAY button not found");
   							}
   						}, 400); // Adjust delay if needed
   					}
   					// Check for available seat buttons before proceeding
   					setTimeout(() => {
   						const availableSeats = document.querySelectorAll('.btn-seat.seat-available');
   						if (!availableSeats || availableSeats.length === 0) {
   							console.error("No available seats found after coach selection. Aborting seat selection.");
   							window.seatSelectionFailed = true;
   							resolve(false);
   							return;
   						}
   						resolve(coachWithHighestSeat);
   					}, 500);
   				} else {
   					console.log("Bogie selection not found, continuing anyway...");
   					resolve(null);
   				}
   			}, 500);
   		});

   		waitForInterface.then((coachWithHighestSeat) => {
   			if (coachWithHighestSeat === false || window.seatSelectionFailed) {
   				console.error("Seat selection failed due to no available seats.");
   				return false;
   			}
   			setTimeout(() => {
   				console.log("Starting seat selection...");
   				const clickSeatButton = (seatNumber) => {
   					let selector;
   					if (coachWithHighestSeat) {
   						selector = '.btn-seat.seat-available[title^="' + coachWithHighestSeat + '-"][title$="-' + seatNumber + '"]';
   					} else {
   						selector = '.btn-seat.seat-available[title$="-' + seatNumber + '"]';
   					}
   					console.log("Trying selector for seat " + seatNumber + ":", selector);
   					const seatButton = document.querySelector(selector);
   					if (seatButton) {
   						seatButton.click();
   						console.log("Clicked seat:", seatNumber);
   						return true;
   					}
   					return false;
   				};
   				
   				let seatNumber = "` + arguments.SEAT_FACE + `".includes("Towards") ? 1 : 100;
   				let seatCount = parseInt("` + strconv.Itoa(int(arguments.SEAT_COUNT)) + `");
   				let increment = "` + arguments.SEAT_FACE + `".includes("Towards") ? 1 : -1;
   				console.log("Starting seat selection from:", seatNumber, "increment:", increment, "need:", seatCount);

   				while (seatCount > 0 && seatNumber > 0 && seatNumber <= 100) {
   					if (clickSeatButton(seatNumber)) {
   						seatCount--;
   						console.log("Seats remaining:", seatCount);
   					}
   					seatNumber += increment;
   				}

   				// After seats are selected, track them and start holding cycle
   				setTimeout(() => {
   					console.log("Tracking seats for holding...");
   					if (trackSelectedSeats()) {
   						// Store seat details in window for Go to access
   						window.selectedSeatDetails = {
   							coach: coachWithHighestSeat || 'Unknown',
   							seatClass: '` + selectedClass + `',
   							seats: heldSeats.map(s => s.id)
   						};
   						console.log("Starting hold cycle...");
   						startHoldCycle();
   					} else {
   						console.error("No seats were tracked for holding - checking for any selected seats...");
   						// Fallback: look for any selected seats
   						const anySelected = document.querySelectorAll('[class*="selected"], .seat-selected');
   						console.log("Found selected elements:", anySelected.length);
   						anySelected.forEach((el, i) => console.log("Selected element " + i + ":", el));
   					}
   				}, 1000);
   				
   			}, 100);
   		});

   		return true;
   		
   	} catch (error) {
   		console.error("Seat holding setup error:", error);
   		return false;
   	}
   })();`
}

func createAltUrl(originalUrl string, searchAltUrl bool, attemptNo int, url string, altUrl string) string {
	if searchAltUrl {
		if attemptNo%2 == 0 {
			url = originalUrl
		} else {
			url = altUrl
		}
	}
	return url
}

func updateMessageBody(messageBodyUpdated bool, messageBody string, selectedSpecificTrain string, selectedClass string) (string, bool) {
	if !messageBodyUpdated {
		messageBody += "Train Name: " + selectedSpecificTrain + "\n"
		messageBody += "Seat Class: " + selectedClass + "\n"
		messageBodyUpdated = true
	}
	return messageBody, messageBodyUpdated
}

func generateHtmlFile(err error, renderedHTML string) {
	filename := "parsed-page.html"
	file, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	_, err = file.WriteString(renderedHTML)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("HTML file generated:", filename)
}

func printHtml(err error, doc *goquery.Document) string {
	renderedHTML, err := doc.Html()
	if err != nil {
		log.Fatal(err)
	}
	log.Println(renderedHTML)
	return renderedHTML
}
