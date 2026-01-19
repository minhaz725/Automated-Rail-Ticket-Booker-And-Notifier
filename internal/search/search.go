package search

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/internal/notifier"
	"Rail-Ticket-Notifier/utils"
	"Rail-Ticket-Notifier/utils/constants"
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// getUserAgent returns a random user agent from a pool of realistic browser user agents
func getUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/120.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
	}
	return userAgents[rand.Intn(len(userAgents))]
}

// getRandomDelay returns a random delay between SEARCH_DELAY_MIN_SEC and SEARCH_DELAY_MAX_SEC
func getRandomDelay() time.Duration {
	min := constants.SEARCH_DELAY_MIN_SEC
	max := constants.SEARCH_DELAY_MAX_SEC
	delay := rand.Intn(max-min+1) + min
	return time.Duration(delay) * time.Second
}

func PerformSearch(originalUrl string, seatBookerFunction string) (string, bool) {
	// prevent unused param warning (future use maybe dynamic booking strategy)
	_ = seatBookerFunction

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	attemptNo := 0
	url := originalUrl
	altUrl := ""
	searchAltUrl := false
	selectedSpecificTrain := ""
	selectedClass := ""
	var availableSeatClassArray []string
	seatFound := false
	messageBodyUpdated := false
	messageBody := ""
	loadTimer := 1 * time.Second // Reduced initial loadTimer
	var searchCtx context.Context

	// Add cleanup handler for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Received exit signal. Cleaning up seats...")
		if searchCtx != nil {
			err := chromedp.Run(searchCtx, chromedp.Evaluate(`
   			if (typeof window.stopSeatHolding === 'function') {
   				window.stopSeatHolding();
   				console.log("Seats released due to app exit");
   			}
   		`, nil))
			if err != nil {
				log.Printf("Cleanup JS error: %v\n", err)
			}
		}
		log.Println("Cleanup completed. Exiting...")
		os.Exit(0)
	}()

	// Login (remote debugger chrome assumed running)
	loginAlloc, cancelAlloc := chromedp.NewRemoteAllocator(context.Background(), constants.DEBUG_CHROME_URL)
	loginCtx, cancelLogin := chromedp.NewContext(loginAlloc)
	defer cancelLogin()
	defer cancelAlloc()

	err := chromedp.Run(loginCtx,
		emulation.SetUserAgentOverride(getUserAgent()),
		chromedp.Navigate(constants.LOGIN_URL),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		log.Fatal("Login page navigate failed:", err)
	}

	var currentURL string
	err = chromedp.Run(loginCtx, chromedp.Location(&currentURL))
	if err != nil {
		log.Printf("Location error: %v\n", err)
	}
	if currentURL == constants.LOGIN_URL {
		log.Println("Still on login page; user should have logged in via setup dialog.")
	} else {
		log.Println("Logged in.")
	}

	if strings.EqualFold(arguments.FROM, "Dhaka") {
		altUrl = arguments.GenerateAltURL()
		searchAltUrl = true
		log.Println("Alt search (Biman Bandar) enabled")
	}

	// Main search loop - use login context for headless searches
	for {
		log.Println("Search Started")
		loadTimer = 1 * time.Second // Resetting initial loadTimer
		url = createAltUrl(originalUrl, searchAltUrl, attemptNo, url, altUrl)

		// Use the existing login context for headless searches
		searchCtx = loginCtx
		var searchCancel context.CancelFunc

		// If seat was found and we need to open a new tab for booking, create new context
		if seatFound {
			log.Println("Creating new tab for seat holding...")
			bookingAlloc, bookingAllocCancel := chromedp.NewRemoteAllocator(context.Background(), constants.DEBUG_CHROME_URL)
			defer bookingAllocCancel()
			searchCtx, searchCancel = chromedp.NewContext(bookingAlloc)
			defer searchCancel()
		}

		// Get a random user agent for this search attempt
		currentUserAgent := getUserAgent()
		log.Printf("Using User Agent: %s", currentUserAgent)

		err = chromedp.Run(searchCtx,
			emulation.SetUserAgentOverride(currentUserAgent),
			chromedp.Navigate(url),
			chromedp.WaitReady("body"),
			chromedp.Sleep(loadTimer), // You can try removing this line entirely if page loads are fast
		)
		if err != nil {
			if strings.Contains(err.Error(), "net::ERR_INTERNET_DISCONNECTED") {
				log.Println("Network issue. Retrying...")
			} else {
				log.Printf("Navigate error: %v\n", err)
			}
			if loadTimer < 20*time.Second {
				loadTimer += 2 * time.Second
			}
			// Only cancel if we created a new context for booking
			if seatFound && searchCancel != nil {
				searchCancel()
			}
			attemptNo++
			time.Sleep(3 * time.Second)
			continue
		}

		var afterNav string
		err = chromedp.Run(searchCtx, chromedp.Location(&afterNav))
		if err != nil {
			log.Printf("Search location error: %v\n", err)
		}
		if afterNav == constants.LOGIN_URL {
			log.Println("Lost session (redirect to login). Will re-login next loop.")
			// Only cancel if we created a new context for booking
			if seatFound && searchCancel != nil {
				searchCancel()
			}
			attemptNo++
			time.Sleep(2 * time.Second)
			continue
		}

		// Poll for trip results (max 4s)
		var haveResults bool
		for i := 0; i < 20; i++ { // 20 * 200ms = 4s
			_ = chromedp.Run(searchCtx, chromedp.Evaluate(`document.querySelectorAll('.single-trip-wrapper').length>0`, &haveResults))
			if haveResults {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		if !haveResults {
			log.Println("No results yet; increasing wait window")
			if loadTimer < 20*time.Second {
				loadTimer += 2 * time.Second
			}
			// Only cancel if we created a new context for booking
			if seatFound && searchCancel != nil {
				searchCancel()
			}
			attemptNo++

			// Use random delay instead of fixed delay
			randomDelay := getRandomDelay()
			log.Printf("Waiting %v seconds before next search attempt...", randomDelay.Seconds())
			time.Sleep(randomDelay)
			continue
		}

		var pageContent string
		if err := chromedp.Run(searchCtx, chromedp.InnerHTML("html", &pageContent)); err != nil {
			log.Printf("HTML extraction error: %v\n", err)
			// Only cancel if we created a new context for booking
			if seatFound && searchCancel != nil {
				searchCancel()
			}
			attemptNo++
			continue
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageContent))
		if err != nil {
			log.Fatal(err)
		}

		// Check for seats only if not already found
		if !seatFound {
			doc.Find(".single-trip-wrapper").Each(func(i int, el *goquery.Selection) {
				trainName := el.Find(".trip-name h2").Text()
				if trainName == "" || !strings.Contains(trainName, arguments.SPECIFIC_TRAIN_ARRAY[0]) {
					return
				}
				fmt.Println("Search URL:", url)
				fmt.Println("Train Name:", trainName)
				classFound := false
				el.Find(".seat-classes-row .seat-class-name, .seat-classes-row .all-seats").Each(func(i int, s *goquery.Selection) {
					if s.HasClass("seat-class-name") {
						className := s.Text()
						for _, desired := range arguments.SEAT_TYPE_ARRAY {
							if className == desired {
								fmt.Print("Class Name:", className)
								availableSeatClassArray = append(availableSeatClassArray, className)
								classFound = true
							}
						}
					} else if s.HasClass("all-seats") && classFound {
						seatCountStr := strings.TrimSpace(s.Text())
						seatCount, _ := strconv.ParseUint(seatCountStr, 10, 0)
						fmt.Println(" Seat Count:", seatCount)
						classFound = false
						if uint(seatCount) >= arguments.SEAT_COUNT {
							seatFound = true
							selectedSpecificTrain = trainName
							selectedClass = utils.FindFirstMatch(availableSeatClassArray, arguments.SEAT_TYPE_ARRAY)
							if selectedClass == "" {
								log.Fatal("class selection error")
							}
							messageBody, messageBodyUpdated = updateMessageBody(messageBodyUpdated, messageBody, selectedSpecificTrain, selectedClass)
							log.Println("Seat found! Initiating seat holding process...")
							// Seat holding process
							holdDuration := 1 // minutes
							jsCode := buildSeatHoldingJS(selectedSpecificTrain, selectedClass, holdDuration)
							holdingStartTime := time.Now()
							var success bool
							if err := chromedp.Run(searchCtx, chromedp.Evaluate(jsCode, &success), chromedp.Sleep(2*time.Second)); err != nil {
								log.Printf("Seat holding JS error: %v\n", err)
							} else {
								log.Println("Seat holding initiated successfully. Seats will be cycled every 4 minutes.")
								log.Println("To stop holding: execute 'window.stopSeatHolding()' in browser console")
							}
							// Wait for seat selection and extract details
							time.Sleep(5 * time.Second)
							var seatDetails struct {
								Coach     string   `json:"coach"`
								SeatClass string   `json:"seatClass"`
								Seats     []string `json:"seats"`
							}
							err = chromedp.Run(searchCtx, chromedp.Evaluate(`window.selectedSeatDetails || {coach: 'Unknown', seatClass: 'Unknown', seats: []}`, &seatDetails))
							if err != nil {
								log.Printf("Seat details JS error: %v\n", err)
							}

							// Check if enough seats were selected
							if len(seatDetails.Seats) < int(arguments.SEAT_COUNT) {
								log.Printf("Seat selection failed: only %d out of %d seats selected. Discarding and continuing search...", len(seatDetails.Seats), arguments.SEAT_COUNT)
								// If a new tab was opened for booking, close it
								if seatFound && searchCancel != nil {
									searchCancel()
								}
								// Reset state for next search
								seatFound = false
								selectedSpecificTrain = ""
								selectedClass = ""
								availableSeatClassArray = nil
								messageBodyUpdated = false
								messageBody = ""
								return // Exit current attempt, continue search loop
							}

							if len(seatDetails.Seats) > 0 {
								messageBody += fmt.Sprintf("Coach: %s\n", seatDetails.Coach)
								messageBody += fmt.Sprintf("Selected Seats: %s\n", strings.Join(seatDetails.Seats, ", "))
							}
							messageBody += "The tab will remain open. Complete Booking.\n"
							log.Println("Seat holding process initiated. Tab will remain open.")
							log.Println("Complete message:", messageBody)
							log.Println("Keeping tab open for seat holding. Seats will cycle every 4 minutes...")
							emailSent := false
							for {
								time.Sleep(30 * time.Second)
								var holdingStopped bool
								err := chromedp.Run(searchCtx, chromedp.Evaluate(`typeof window.stopSeatHolding === 'undefined'`, &holdingStopped))
								if err != nil {
									log.Printf("Error checking holding status: %v", err)
									break
								}
								if holdingStopped {
									log.Println("Seat holding has been stopped by user.")
									messageBody += "Seat holding stopped. All seats released.\n"
									break
								}
								if !emailSent && time.Since(holdingStartTime) > time.Duration(holdDuration)*time.Second {
									log.Println("Sending notification email after first booking cycle...")
									if notifier.SendEmail(messageBody) {
										log.Println("Email sent successfully")
										emailSent = true
									} else {
										log.Println("Failed to send email")
									}
								}
								log.Printf("Seat holding active... (running indefinitely)")
							}
							return
						} else {
							availableSeatClassArray = availableSeatClassArray[:len(availableSeatClassArray)-1]
						}
					}
				})
			})
		}

		attemptNo++
		log.Println("Search Ended - Attempt:", attemptNo)

		// Use random delay between all search attempts
		randomDelay := getRandomDelay()
		log.Printf("Waiting %v seconds before next search attempt...", randomDelay.Seconds())
		time.Sleep(randomDelay)
	}
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
