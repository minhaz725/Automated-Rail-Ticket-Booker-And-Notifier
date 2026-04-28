package search

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/internal/models"
	"Rail-Ticket-Notifier/internal/notifier"
	"Rail-Ticket-Notifier/utils"
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
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const authCacheDir = "Rail-Ticket-Notifier"

func getAuthCachePath() (string, error) {
	index := arguments.INSTANCE_INDEX
	if index < 1 || index > 10 {
		index = 1
	}
	port := 9221 + index
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cacheDir, authCacheDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	fileName := fmt.Sprintf("auth-cache-%d-port-%d.json", index, port)
	return filepath.Join(dir, fileName), nil
}

func saveAuthToFile(auth *models.CapturedAuth) {
	path, err := getAuthCachePath()
	if err != nil {
		log.Printf("Failed to get cache path: %v", err)
		return
	}
	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal auth: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		log.Printf("Failed to save auth cache: %v", err)
		return
	}
	log.Printf("Auth saved to cache: %s", path)
}

func loadAuthFromFile() *models.CapturedAuth {
	path, err := getAuthCachePath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var auth models.CapturedAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		log.Printf("Failed to parse auth cache: %v", err)
		return nil
	}
	if auth.Authorization == "" {
		return nil
	}
	return &auth
}

func validateAuth(auth *models.CapturedAuth) bool {
	_, err := SearchTrainsAPI(auth, "Dhaka", "Chattogram", arguments.DATE, "S_CHAIR")
	if err != nil {
		log.Printf("Cached auth validation failed: %v", err)
		return false
	}
	return true
}

// GetOrCaptureAuth tries to load cached auth from file first, validates it,
// and falls back to browser capture if cache is missing or expired.
func GetOrCaptureAuth(searchUrl string) (*models.CapturedAuth, error) {
	if cached := loadAuthFromFile(); cached != nil {
		log.Println("Found cached auth, validating...")
		if validateAuth(cached) {
			log.Println("Cached auth is valid!")
			return cached, nil
		}
		log.Println("Cached auth expired, recapturing from browser...")
	}

	auth, err := CaptureAuthFromBrowser(searchUrl)
	if err != nil {
		return nil, err
	}
	saveAuthToFile(auth)
	return auth, nil
}

func CaptureAuthFromBrowser(searchUrl string) (*models.CapturedAuth, error) {
	allocCtx, cancel := chromedp.NewRemoteAllocator(context.Background(), utils.GetDebugChromeURL())
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	auth := &models.CapturedAuth{}

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

	var result models.TrainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// FindAvailableSeats checks for seats matching criteria, respecting both seat type AND train priority order
func FindAvailableSeats(trains *models.TrainResponse, targetTrains []string, targetSeatTypes []string, minSeats uint) (string, string, int) {
	// Priority order: Train first, then Seat type
	// So if you input trains: SUBORNO,SONAR and seats: S_CHAIR,SNIGDHA
	// It will first try SUBORNO+S_CHAIR, then SUBORNO+SNIGDHA, then SONAR+S_CHAIR, then SONAR+SNIGDHA
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

// ========== MAIN SEARCH ==========

func PerformSearch(originalUrl string, seatBookerFunction string) (string, bool) {
	_ = seatBookerFunction
	rand.Seed(time.Now().UnixNano())

	// Ensure Chrome is running (handles the case where SetupChrome was never
	// called because the intro dialog is skipped / commented out).
	if err := utils.EnsureChrome(); err != nil {
		log.Fatalf("Failed to launch Chrome: %v", err)
	}

	messageBody := ""

	// Setup browser context - KEEP THIS OPEN FOR THE ENTIRE SESSION
	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), utils.GetDebugChromeURL())
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

	// Step 1: Try cached auth first, fall back to browser capture
	log.Println("Getting auth credentials...")
	auth, err := GetOrCaptureAuth(originalUrl)
	if err != nil {
		log.Fatal("Failed to get auth:", err)
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
		log.Println("Search Url: " + createAltUrl(originalUrl, searchAltUrl, attemptNo, originalUrl, altUrl))

		// Decide which FROM to use

		currentFrom := arguments.FROM
		//currentTo := arguments.TO
		//currentDate := arguments.DATE
		if searchAltUrl && attemptNo%2 == 1 {
			currentFrom = "Biman_Bandar"
			//currentTo = "Sylhet"
			//currentDate = "28-Mar-2026"
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
				} else {
					saveAuthToFile(auth)
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

	// Run notifications in background goroutines to not block booking
	go notifier.SendEmail(*messageBody)
	go notifier.MakeCall()

	log.Println("Opening NEW TAB in debug Chrome...")

	// Create new tab in debug Chrome via CDP
	allocCtx, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), utils.GetDebugChromeURL())
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

	// Wait for page to load with faster polling (200ms intervals)
	log.Println("Waiting for page to load...")
	var hasResults bool
	maxWaitMs := 20000 // 20 seconds max
	for elapsed := 0; elapsed < maxWaitMs; elapsed += 200 {
		time.Sleep(200 * time.Millisecond)
		chromedp.Run(bookingCtx, chromedp.Evaluate(`document.querySelectorAll('.single-trip-wrapper').length > 0`, &hasResults))
		if hasResults {
			log.Printf("Page loaded after %dms", elapsed+200)
			break
		}
	}

	if !hasResults {
		log.Println("Page didn't load. Manual booking required.")
		log.Println("URL:", searchUrl)
		// Don't close tab - user can try manually
		return false
	}

	log.Println("PAGE LOADED! Running seat selection JS...")

	// Execute seat holding JS
	jsCode := buildSeatHoldingJS(trainName, seatClass, 250)
	chromedp.Run(bookingCtx, chromedp.Evaluate(jsCode, nil))

	// Poll for seat selection completion with faster polling (200ms intervals)
	// Increase max wait to 15 seconds since JS needs time for coach selection + seat loading
	var seatDetails struct {
		Coach     string   `json:"coach"`
		SeatClass string   `json:"seatClass"`
		Seats     []string `json:"seats"`
	}

	maxSeatWaitMs := 15000 // 15 seconds max for seat selection
	for elapsed := 0; elapsed < maxSeatWaitMs; elapsed += 200 {
		time.Sleep(200 * time.Millisecond)
		chromedp.Run(bookingCtx, chromedp.Evaluate(`window.selectedSeatDetails || {coach: '', seatClass: '', seats: []}`, &seatDetails))
		if len(seatDetails.Seats) >= int(arguments.SEAT_COUNT) {
			log.Printf("Seats selected after %dms", elapsed+200)
			break
		}
	}

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

func buildSeatHoldingJS(trainName, selectedClass string, holdDurationSeconds int) string {
	return `(() => {
   	console.log("Starting seat holding process...");
   	let heldSeats = [];
   	let isHolding = false;
   	let stopRequested = false;

   	function trackSelectedSeats() {
   		const selectedSeatElements = document.querySelectorAll('.btn-seat.seat-selected, [class*="seat"][class*="selected"]');
   		heldSeats = [];
   		selectedSeatElements.forEach(seat => {
   			const seatId = seat.getAttribute('title') || seat.getAttribute('data-seat') || seat.textContent;
   			if (seatId) {
   				heldSeats.push({ id: seatId, selector: '[title="' + seatId + '"]' });
   			}
   		});
   		console.log("Tracked seats:", heldSeats.map(s => s.id));
   		return heldSeats.length > 0;
   	}

   	function sleep(ms) {
   		return new Promise(resolve => setTimeout(resolve, ms));
   	}

   	function getTimeLog() {
   		const now = new Date();
   		const day = now.getDate();
   		const month = now.toLocaleString('default', { month: 'long' });
   		const hours = now.getHours();
   		const minutes = String(now.getMinutes()).padStart(2, '0');
   		const ampm = hours >= 12 ? 'pm' : 'am';
   		const h = hours % 12 || 12;
   		return day + ' ' + month + ', ' + h + ':' + minutes + ' ' + ampm;
   	}

   	async function toggleSeats(hold) {
   		console.log((hold ? "Re-selecting" : "Unselecting") + " " + heldSeats.length + " seats...");
   		for (let i = 0; i < heldSeats.length; i++) {
   			if (stopRequested) return;
   			try {
   				let el = document.querySelector(heldSeats[i].selector);
   				if (el && !el.disabled) {
   					el.click();
   					console.log((hold ? "Re-selected" : "Unselected") + " seat " + (i+1) + "/" + heldSeats.length + ":", heldSeats[i].id);
   				} else {
   					console.log("Seat not found or disabled:", heldSeats[i].id);
   				}
   			} catch (e) {
   				console.error("Error toggling seat:", heldSeats[i].id, e);
   			}
   			// Always wait between seats (except after last one)
   			if (i < heldSeats.length - 1) {
   				await sleep(300);
   			}
   		}
   		console.log((hold ? "Re-selection" : "Unselection") + " complete");
   	}

   	async function holdCycle() {
   		if (stopRequested) return;
   		
   		console.log("--- Hold cycle starting [" + getTimeLog() + "] ---");
   		
   		// Unselect all seats
   		await toggleSeats(false);
   		
   		if (stopRequested) return;
   		
   		// Wait 1.5 seconds
   		console.log("Waiting 1.5s before re-selecting...");
   		await sleep(1500);
   		
   		if (stopRequested) return;
   		
   		// Re-select all seats
   		await toggleSeats(true);
   		
   		console.log("--- Hold cycle complete [" + getTimeLog() + "] ---");
   		
   		// Schedule next cycle
   		if (!stopRequested) {
   			const cycleInterval = ` + fmt.Sprintf("%d", holdDurationSeconds) + ` * 1000;
   			console.log("Next cycle in " + (cycleInterval/1000) + " seconds");
   			setTimeout(holdCycle, cycleInterval);
   		}
   	}

   	function startHoldCycle() {
   		if (isHolding) return;
   		isHolding = true;
   		stopRequested = false;
   		const cycleInterval = ` + fmt.Sprintf("%d", holdDurationSeconds) + ` * 1000;
   		console.log("Seat holding started. First cycle in " + (cycleInterval/1000) + " seconds");
   		// Start first cycle after the interval
   		setTimeout(holdCycle, cycleInterval);
   		
   		window.stopSeatHolding = async () => {
   			console.log("Stopping seat holding...");
   			stopRequested = true;
   			isHolding = false;
   			await toggleSeats(false);
   		};
   	}

   	// Fast polling - checks every 50ms, exits immediately when found
   	function poll(checkFn, maxMs = 3000) {
   		return new Promise((resolve) => {
   			const start = Date.now();
   			const check = () => {
   				const result = checkFn();
   				if (result) { resolve(result); return; }
   				if (Date.now() - start > maxMs) { resolve(null); return; }
   				setTimeout(check, 50);
   			};
   			check();
   		});
   	}

   	async function run() {
   		try {
   			console.log("Looking for train:", "` + trainName + `");
   			const header = Array.from(document.querySelectorAll("h2")).find(h => h.innerText.includes("` + trainName + `"));
   			if (!header) throw new Error("Header not found");

   			const appSingleTrip = header.closest("app-single-trip");
   			if (!appSingleTrip) throw new Error("Parent not found");

   			const seatDiv = Array.from(appSingleTrip.querySelectorAll(".single-seat-class")).find(div => {
   				const span = div.querySelector(".seat-class-name");
   				return span && span.innerText.trim() === "` + selectedClass + `";
   			});
   			if (!seatDiv) throw new Error("Seat class not found");

   			const bookNowBtn = seatDiv.querySelector(".book-now-btn-wrapper .book-now-btn");
   			if (!bookNowBtn) throw new Error("Book button not found");

   			console.log("Clicking book now...");
   			bookNowBtn.click();

   			// Poll for bogie dropdown (50ms intervals, max 3s)
   			const bogieSelection = await poll(() => {
   				const el = document.getElementById("select-bogie");
   				return (el && el.options.length >= 1) ? el : null;
   			}, 3000);

   			if (!bogieSelection) throw new Error("Bogie dropdown not found");

   			const extractNum = (text) => {
   				const m = text.match(/ - (\d+) Seat\(s\)/);
   				return m ? parseInt(m[1]) : 0;
   			};

   			const options = Array.from(bogieSelection.options);
   			const best = options.reduce((a, b) => extractNum(b.text) > extractNum(a.text) ? b : a, options[0]);
   			const coachName = best.text.split(" - ")[0];
   			console.log("Selected coach:", coachName);

   			bogieSelection.value = best.value;
   			bogieSelection.dispatchEvent(new Event("change", { bubbles: true }));

   			// Handle XTR popup immediately if needed
   			if (coachName.startsWith("XTR")) {
   				await poll(() => {
   					const btn = Array.from(document.querySelectorAll("button")).find(el => el.textContent.trim().toUpperCase() === "OKAY");
   					if (btn) { btn.click(); return true; }
   					return null;
   				}, 1000);
   			}

   			// Poll for seats to appear (50ms intervals, max 2s since they load fast)
   			const seats = await poll(() => {
   				const s = document.querySelectorAll('.btn-seat.seat-available');
   				return s.length > 0 ? s : null;
   			}, 2000);

   			if (!seats) throw new Error("No seats found");
   			console.log("Found " + seats.length + " seats");

			// Log all seat states
			const allSeats = document.querySelectorAll('.btn-seat:not(.seat-hidden)');
			const seatLog = { available: [], booked: [], inProgress: [] };
			
			allSeats.forEach(btn => {
				const title = btn.getAttribute('title');
				if (!title) return;
				if (btn.classList.contains('seat-available'))       seatLog.available.push(title);
				else if (btn.classList.contains('seat-booked'))     seatLog.booked.push(title);
				else if (btn.classList.contains('seat-in-progress')) seatLog.inProgress.push(title);
			});
			
			console.log("Available seats:", seatLog.available);
			console.log("Booked seats:", seatLog.booked);
			console.log("In-progress seats:", seatLog.inProgress);

   			// Click seats - first seat fast, 300ms delay before consecutive seats
   			const seatCount = parseInt("` + strconv.Itoa(int(arguments.SEAT_COUNT)) + `");
   			const goTowards = "` + arguments.SEAT_FACE + `".includes("Towards");
   			let current = goTowards ? 1 : 100;
   			const inc = goTowards ? 1 : -1;
   			let selected = 0;
   			const isBerth = ["AC_B", "F_BERTH"].includes("` + selectedClass + `");

   			for (let i = 0; i < 100 && selected < seatCount; i++) {
   				const seatTitle = isBerth
   					? coachName + '-' + (current % 2 === 1 ? 'LO' : 'UP') + '-' + current
   					: coachName + '-' + current;
   				const sel = '.btn-seat.seat-available[title="' + seatTitle + '"]';
   				const btn = document.querySelector(sel);
   				if (btn) {
   					// Wait 300ms before clicking 2nd, 3rd, 4th... seats (not first)
   					if (selected > 0) {
   						await new Promise(r => setTimeout(r, 600));
   					}
   					btn.click();
   					selected++;
   					console.log("Clicked seat " + seatTitle + " (" + selected + "/" + seatCount + ")");
   				}
   				current += inc;
   			}

   			console.log("Selected " + selected + " seats");

   			// Wait a bit for UI to update, then track
   			await new Promise(r => setTimeout(r, 500));

   			// Track and set result
   			if (trackSelectedSeats()) {
   				window.selectedSeatDetails = {
   					coach: coachName,
   					seatClass: '` + selectedClass + `',
   					seats: heldSeats.map(s => s.id)
   				};
   				console.log("Done! Seats:", window.selectedSeatDetails.seats);
   				startHoldCycle();
   			} else {
   				console.error("Failed to track selected seats!");
   			}

   		} catch (error) {
   			console.error("Error:", error);
   		}
   	}

   	run();
   	return true;
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
