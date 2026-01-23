package search

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/internal/notifier"
	"Rail-Ticket-Notifier/utils/constants"
	"context"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
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

func bookSeatsInExistingTab(ctx context.Context, trainName, seatClass, searchUrl string, messageBody *string) bool {
	log.Println("SEATS FOUND!")

	*messageBody += fmt.Sprintf("\nURL: %s\n", searchUrl)
	notifier.SendEmail(*messageBody)
	notifier.MakeCall()

	log.Println("Opening NEW TAB in debug Chrome...")

	// Create new tab in debug Chrome via CDP
	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), constants.DEBUG_CHROME_URL)
	bookingCtx, cancelBooking := chromedp.NewContext(allocCtx) // This creates a new tab

	// Navigate via JS (not CDP navigate)
	err := chromedp.Run(bookingCtx,
		chromedp.Navigate("about:blank"), // Open blank first
	)
	if err != nil {
		log.Printf("Failed to create tab: %v", err)
		cancelBooking()
		cancelAlloc()
		return false
	}

	// Now navigate via JavaScript
	log.Println("Navigating via JS...")
	chromedp.Run(bookingCtx,
		chromedp.Evaluate(fmt.Sprintf(`window.location.href = "%s"`, searchUrl), nil),
	)

	// Wait for page to load
	log.Println("Waiting for page to load...")
	time.Sleep(15 * time.Second)

	// Check if results loaded
	var hasResults bool
	chromedp.Run(bookingCtx, chromedp.Evaluate(`document.querySelectorAll('.single-trip-wrapper').length > 0`, &hasResults))

	if !hasResults {
		log.Println("Page didn't load. Manual booking required.")
		log.Println("URL:", searchUrl)
		// Don't close tab - user can try manually
		return false
	}

	log.Println("PAGE LOADED! Running seat selection JS...")

	// Execute seat holding JS
	jsCode := buildSeatHoldingJS(trainName, seatClass, 1)
	chromedp.Run(bookingCtx, chromedp.Evaluate(jsCode, nil))

	time.Sleep(15 * time.Second)

	var seatDetails struct {
		Coach     string   `json:"coach"`
		SeatClass string   `json:"seatClass"`
		Seats     []string `json:"seats"`
	}
	chromedp.Run(bookingCtx, chromedp.Evaluate(`window.selectedSeatDetails || {coach: '', seatClass: '', seats: []}`, &seatDetails))

	if len(seatDetails.Seats) < int(arguments.SEAT_COUNT) {
		log.Printf("Only %d seats selected, need %d", len(seatDetails.Seats), arguments.SEAT_COUNT)
		return false
	}

	*messageBody += fmt.Sprintf("Coach: %s\nSeats: %s\n", seatDetails.Coach, strings.Join(seatDetails.Seats, ", "))

	log.Println("==========================================")
	log.Println("SEATS SELECTED! Complete payment manually.")
	log.Println("==========================================")

	select {} // Keep alive
}

func getChromePath() string {
	switch runtime.GOOS {
	case "windows":
		paths := []string{
			os.Getenv("ProgramFiles") + "\\Google\\Chrome\\Application\\chrome.exe",
			os.Getenv("ProgramFiles(x86)") + "\\Google\\Chrome\\Application\\chrome.exe",
			os.Getenv("LocalAppData") + "\\Google\\Chrome\\Application\\chrome.exe",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	case "darwin":
		return "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	case "linux":
		return "google-chrome"
	}
	return ""
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

   	function startHoldCycle() {
   		if (isHolding) {
   			console.log("Holding cycle already active");
   			return;
   		}
   		
   		isHolding = true;
   		console.log('Starting seat hold cycle. Hold duration: ' + ` + fmt.Sprintf("%d", holdDurationMinutes) + ` + ' minutes');
   		
   		const cycleInterval = ` + fmt.Sprintf("%d", holdDurationMinutes) + ` * 60 * 1000;
   		
   		holdInterval = setInterval(() => {
   			console.log("=== SEAT CYCLE START ===");
   			toggleSeats(false);
   			
   			setTimeout(() => {
   				toggleSeats(true);
   				console.log("=== SEAT CYCLE COMPLETE ===");
   			}, 1000);
   			
   		}, cycleInterval);
   		
   		window.stopSeatHolding = () => {
   			console.log("Stopping seat holding...");
   			if (holdInterval) {
   				clearInterval(holdInterval);
   			}
   			isHolding = false;
   			toggleSeats(false);
   			console.log("All seats released.");
   		};
   		
   		console.log("Seat holding started. Call window.stopSeatHolding() to stop.");
   	}

   	try {
   		console.log("Looking for train:", "` + trainName + `");
   		const headers = Array.from(document.querySelectorAll("h2"));
   		const header = headers.find((h) => h.innerText.includes("` + trainName + `"));
   		
   		if (!header) throw new Error("Header not found");
   		console.log("Found train header");
   		
   		const appSingleTrip = header.closest("app-single-trip");
   		if (!appSingleTrip) throw new Error("Parent component not found");

   		const seatClassDivs = Array.from(appSingleTrip.querySelectorAll(".single-seat-class"));
   		let seatDiv = seatClassDivs.find((div) => {
   			let seatNameSpan = div.querySelector(".seat-class-name");
   			return seatNameSpan && seatNameSpan.innerText.trim() === "` + selectedClass + `";
   		});

   		if (!seatDiv) throw new Error("Seat class div not found for: " + "` + selectedClass + `");

   		const bookNowBtn = seatDiv.querySelector(".book-now-btn-wrapper .book-now-btn");
   		if (!bookNowBtn) throw new Error("Book now button not found");

   		console.log("Clicking book now button...");
   		bookNowBtn.click();

   		const waitForInterface = new Promise((resolve) => {
   			setTimeout(() => {
   				const bogieSelection = document.getElementById("select-bogie");
   				if (bogieSelection) {
   					const extractNumber = (text) => {
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

   					if (coachWithHighestSeat && coachWithHighestSeat.startsWith("XTR")) {
   						setTimeout(() => {
   							const okBtn = Array.from(document.querySelectorAll("button, input[type='button']"))
   								.find(el => el.textContent.trim().toUpperCase() === "OKAY");
   							if (okBtn) okBtn.click();
   						}, 400);
   					}

   					setTimeout(() => {
   						const availableSeats = document.querySelectorAll('.btn-seat.seat-available');
   						if (!availableSeats || availableSeats.length === 0) {
   							window.seatSelectionFailed = true;
   							resolve(false);
   							return;
   						}
   						resolve(coachWithHighestSeat);
   					}, 500);
   				} else {
   					resolve(null);
   				}
   			}, 500);
   		});

   		waitForInterface.then((coachWithHighestSeat) => {
   			if (coachWithHighestSeat === false || window.seatSelectionFailed) {
   				console.error("Seat selection failed.");
   				return false;
   			}
   			
   			setTimeout(() => {
   				console.log("Starting seat selection with delays...");
   				
   				// FIXED: Click seats with delay between each click
   				const seatCount = parseInt("` + strconv.Itoa(int(arguments.SEAT_COUNT)) + `");
   				const goTowards = "` + arguments.SEAT_FACE + `".includes("Towards");
   				let startSeat = goTowards ? 1 : 100;
   				let increment = goTowards ? 1 : -1;
   				let selected = 0;
   				let currentSeat = startSeat;
   				
   				const clickNextSeat = () => {
   					if (selected >= seatCount) {
   						console.log("All " + selected + " seats selected!");
   						finishSelection();
   						return;
   					}
   					
   					if (currentSeat < 1 || currentSeat > 100) {
   						console.log("Ran out of seat numbers. Selected: " + selected);
   						finishSelection();
   						return;
   					}
   					
   					// Build selector
   					let selector;
   					if (coachWithHighestSeat) {
   						selector = '.btn-seat.seat-available[title="' + coachWithHighestSeat + '-' + currentSeat + '"]';
   					} else {
   						selector = '.btn-seat.seat-available[title$="-' + currentSeat + '"]';
   					}
   					
   					const seatButton = document.querySelector(selector);
   					if (seatButton) {
   						seatButton.click();
   						selected++;
   						console.log("Clicked seat " + currentSeat + " (" + selected + "/" + seatCount + ")");
   					}
   					
   					currentSeat += increment;
   					
   					// Wait 400ms before clicking next seat
   					setTimeout(clickNextSeat, 400);
   				};
   				
   				const finishSelection = () => {
   					setTimeout(() => {
   						console.log("Tracking seats for holding...");
   						if (trackSelectedSeats()) {
   							window.selectedSeatDetails = {
   								coach: coachWithHighestSeat || 'Unknown',
   								seatClass: '` + selectedClass + `',
   								seats: heldSeats.map(s => s.id)
   							};
   							console.log("Selected seats:", window.selectedSeatDetails.seats);
   							startHoldCycle();
   						} else {
   							console.error("No seats tracked!");
   						}
   					}, 1000);
   				};
   				
   				// Start clicking
   				clickNextSeat();
   				
   			}, 500);
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
