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
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func PerformSearch(originalUrl string, seatBookerFunction string) (string, bool) {
	// prevent unused param warning (future use maybe dynamic booking strategy)
	_ = seatBookerFunction

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
	loadTimer := 4 * time.Second
	var searchCtx context.Context

	// Add cleanup handler for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Received exit signal. Cleaning up seats...")
		if searchCtx != nil {
			chromedp.Run(searchCtx, chromedp.Evaluate(`
   			if (typeof window.stopSeatHolding === 'function') {
   				window.stopSeatHolding();
   				console.log("Seats released due to app exit");
   			}
   		`, nil))
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
		emulation.SetUserAgentOverride("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		chromedp.Navigate(constants.LOGIN_URL),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		log.Fatal("Login page navigate failed:", err)
	}

	var currentURL string
	chromedp.Run(loginCtx, chromedp.Location(&currentURL))
	if currentURL == constants.LOGIN_URL && constants.AUTO_LOGIN_ENABLED {
		log.Println("Attempting auto login...")
		if err := autoLogin(loginCtx); err != nil {
			log.Printf("Auto login error: %v\n", err)
		} else {
			chromedp.Run(loginCtx, chromedp.Sleep(1500*time.Millisecond), chromedp.Location(&currentURL))
		}
	}
	if currentURL == constants.LOGIN_URL {
		log.Println("Still on login page; continuing (session may require manual OTP). Searches will attempt anyway.")
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
		url = funcName(originalUrl, searchAltUrl, attemptNo, url, altUrl)

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

		err = chromedp.Run(searchCtx,
			emulation.SetUserAgentOverride("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
			chromedp.Navigate(url),
			chromedp.WaitReady("body"),
			chromedp.Sleep(loadTimer),
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
		chromedp.Run(searchCtx, chromedp.Location(&afterNav))
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

		// Poll for trip results (max 10s)
		var haveResults bool
		for i := 0; i < 20; i++ { // 20 * 500ms
			_ = chromedp.Run(searchCtx, chromedp.Evaluate(`document.querySelectorAll('.single-trip-wrapper').length>0`, &haveResults))
			if haveResults {
				break
			}
			time.Sleep(500 * time.Millisecond)
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
			time.Sleep(constants.SEARCH_DELAY_IN_SEC * time.Second)
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
							holdDuration := 4 // minutes
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
							chromedp.Run(searchCtx, chromedp.Evaluate(`window.selectedSeatDetails || {coach: 'Unknown', seatClass: 'Unknown', seats: []}`, &seatDetails))
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
		time.Sleep(constants.SEARCH_DELAY_IN_SEC * time.Second)
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
   						const match = text.match(/\\d+/);
   						return match ? parseInt(match[0]) : 0;
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
   					resolve(coachWithHighestSeat);
   				} else {
   					console.log("Bogie selection not found, continuing anyway...");
   					resolve(null);
   				}
   			}, 500);
   		});

   		waitForInterface.then((coachWithHighestSeat) => {
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

func funcName(originalUrl string, searchAltUrl bool, attemptNo int, url string, altUrl string) string {
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

func autoLogin(ctx context.Context) error {
	username := constants.LOGIN_UID_VALUE
	password := constants.LOGIN_PASS_VALUE

	js := `(function(u,p){
   	function fill(cands,val){
   		for(const sel of cands){
   			let el=document.querySelector(sel);
   			if(!el) continue;
   			el.focus();
   			el.value='';
   			['keydown','keypress','input'].forEach(ev=>el.dispatchEvent(new KeyboardEvent(ev,{bubbles:true,cancelable:true,key:val.slice(-1)})));
   			for(const ch of val){
   				el.value+=ch;
   				el.dispatchEvent(new Event('input',{bubbles:true}));
   			}
   			el.dispatchEvent(new Event('change',{bubbles:true}));
   			el.blur();
   			return sel;
   		}
   		return null;
   	}
   	const userSel=fill([
   		'input[formcontrolname="mobile"]',
   		'input[name="mobile"]',
   		'input[name="username"]',
   		'input[id*="mobile"]',
   		'input[id="username"]',
   		'input[name="email"]'
   	], u);
   	const passSel=fill([
   		'input[formcontrolname="password"]',
   		'input[type="password"]',
   		'input[name="password"]',
   		'input[id="password"]'
   	], p);

   	let btn=document.querySelector('button[type="submit"].login-form-submit-btn')||document.querySelector('button[type="submit"]');
   	if(btn && (btn.disabled || btn.getAttribute('disabled')!==null)){
   		btn.disabled=false; btn.removeAttribute('disabled');
   	}
   	if(btn){
   		btn.focus();
   		btn.click();
   	}
   	return {userSel,passSel,clicked:!!btn,btnDisabled:btn?btn.disabled:null};
   })(%q,%q);`

	var result struct {
		UserSel     string `json:"userSel"`
		PassSel     string `json:"passSel"`
		Clicked     bool   `json:"clicked"`
		BtnDisabled *bool  `json:"btnDisabled"`
	}

	if err := chromedp.Run(ctx,
		chromedp.WaitReady("body"),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Evaluate(fmt.Sprintf(js, username, password), &result),
		chromedp.Sleep(1200*time.Millisecond),
	); err != nil {
		return err
	}

	log.Printf("AutoLogin -> userSel:%s passSel:%s clicked:%v btnDisabled:%v\n", result.UserSel, result.PassSel, result.Clicked, result.BtnDisabled)

	if !result.Clicked || result.UserSel == "" || result.PassSel == "" {
		return fmt.Errorf("autologin incomplete (clicked=%v userSel=%s passSel=%s)", result.Clicked, result.UserSel, result.PassSel)
	}
	return nil
}
