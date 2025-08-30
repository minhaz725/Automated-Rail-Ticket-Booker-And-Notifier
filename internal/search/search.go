package search

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/utils"
	"Rail-Ticket-Notifier/utils/constants"
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/chromedp"
	"log"
	"os"
	"strconv"
	"strings"
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
		searchCtx := loginCtx
		var searchCancel context.CancelFunc

		// If seat was found and we need to open a new tab for booking, create new context
		if seatFound {
			log.Println("Creating new tab for seat booking...")
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
							return
						} else {
							availableSeatClassArray = availableSeatClassArray[:len(availableSeatClassArray)-1]
						}
					}
				})
			})
		}

		// If seat found, handle the booking process
		if seatFound {
			if !messageBodyUpdated {
				fmt.Println(availableSeatClassArray)
				selectedClass = utils.FindFirstMatch(availableSeatClassArray, arguments.SEAT_TYPE_ARRAY)
				if selectedClass == "" {
					log.Fatal("class selection error")
				}
				messageBody, messageBodyUpdated = updateMessageBody(messageBodyUpdated, messageBody, selectedSpecificTrain, selectedClass)
				log.Println("Seat found! Will open new tab for booking on next iteration...")
				attemptNo++
				time.Sleep(constants.SEARCH_DELAY_IN_SEC * time.Second)
				continue
			}

			// Now we're in the booking tab, click the seat booking button
			jsCode := buildSeatBookingJS(selectedSpecificTrain, selectedClass)
			var success bool
			if err := chromedp.Run(searchCtx, chromedp.Evaluate(jsCode, &success), chromedp.Sleep(2*time.Second)); err != nil {
				log.Printf("Seat booking JS error: %v\n", err)
				// Don't return on error, let user handle manually
			} else {
				log.Println("Seat booking button clicked successfully")
			}

			// Wait for coach selection page to load and handle it
			log.Println("Waiting for coach selection page...")
			var coachPageLoaded bool
			for i := 0; i < 30; i++ { // Wait up to 15 seconds for coach page
				err := chromedp.Run(searchCtx, chromedp.Evaluate(`
					document.querySelector('.coach-selection') !== null ||
					document.querySelector('.seat-selection') !== null ||
					document.querySelector('[class*="coach"]') !== null ||
					document.querySelector('[class*="seat"]') !== null ||
					document.URL.includes('seat') ||
					document.URL.includes('coach')
				`, &coachPageLoaded))
				if err == nil && coachPageLoaded {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}

			if coachPageLoaded {
				log.Println("Coach selection page loaded, attempting to select coach and seats...")

				// First, try to select a coach/boarding station if needed
				log.Println("Looking for boarding station or coach selection...")
				var boardingStationSelected bool
				boardingStationJS := `(() => {
					try {
						// Look for boarding station dropdown
						let dropdown = document.querySelector('select');
						if (dropdown && dropdown.options.length > 1) {
							dropdown.selectedIndex = 1; // Select first available option
							dropdown.dispatchEvent(new Event('change', {bubbles: true}));
							console.log("Boarding station selected");
							return true;
						}

						// Look for coach buttons
						let coachButtons = document.querySelectorAll('[class*="coach"], [class*="wagon"], button[class*="select"]');
						if (coachButtons.length > 0) {
							coachButtons[0].click();
							console.log("Coach selected");
							return true;
						}

						return false;
					} catch (e) {
						console.error("Boarding station/coach selection error:", e);
						return false;
					}
				})();`

				chromedp.Run(searchCtx, chromedp.Evaluate(boardingStationJS, &boardingStationSelected))
				if boardingStationSelected {
					log.Println("Boarding station/coach selection completed")
				}

				// Wait for seat selection interface to load
				log.Println("Waiting for seat selection interface to load...")
				time.Sleep(3 * time.Second)

				// Now look for and select seats
				var seatsSelected bool
				seatSelectionJS := `(() => {
					try {
						console.log("Looking for seats...");

						// Wait a moment for seats to render
						let attempts = 0;
						let maxAttempts = 10;

						function selectSeats() {
							attempts++;
							console.log("Seat selection attempt:", attempts);

							// Look for various seat selector patterns
							let seatSelectors = [
								'[class*="seat"]:not([class*="occupied"]):not([class*="booked"]):not([class*="selected"])',
								'button[class*="seat"]:not([disabled])',
								'.seat:not(.occupied):not(.booked):not(.selected)',
								'[data-seat]:not([disabled])',
								'.available-seat',
								'button[class*="available"]'
							];

							let seatButtons = [];
							for (let selector of seatSelectors) {
								seatButtons = document.querySelectorAll(selector);
								if (seatButtons.length > 0) {
									console.log("Found", seatButtons.length, "seats with selector:", selector);
									break;
								}
							}

							if (seatButtons.length === 0 && attempts < maxAttempts) {
								console.log("No seats found yet, retrying in 500ms...");
								setTimeout(selectSeats, 500);
								return;
							}

							if (seatButtons.length === 0) {
								console.log("No seats found after all attempts");
								return false;
							}

							let seatsNeeded = ` + fmt.Sprintf("%d", arguments.SEAT_COUNT) + `;
							let selectedSeats = 0;

							console.log("Attempting to select", seatsNeeded, "seats from", seatButtons.length, "available");

							for (let i = 0; i < seatButtons.length && selectedSeats < seatsNeeded; i++) {
								let seat = seatButtons[i];

								// Check if seat is clickable
								if (seat.disabled || seat.classList.contains('occupied') || seat.classList.contains('booked')) {
									continue;
								}

								try {
									seat.click();
									selectedSeats++;
									console.log("Selected seat", selectedSeats, "of", seatsNeeded);

									// Add visual feedback
									seat.style.backgroundColor = '#4CAF50';
									seat.style.border = '2px solid #45a049';
								} catch (e) {
									console.error("Error clicking seat:", e);
								}

								// Small delay between seat selections
								if (selectedSeats < seatsNeeded) {
									setTimeout(() => {}, 200);
								}
							}

							console.log("Total seats selected:", selectedSeats);

							// Wait a moment then check if continue button is enabled
							setTimeout(() => {
								let continueBtn = document.querySelector('button[class*="continue"], button[class*="purchase"], input[type="submit"]');
								if (continueBtn) {
									console.log("Continue button found, enabled:", !continueBtn.disabled);
									if (!continueBtn.disabled) {
										console.log("Clicking continue button...");
										continueBtn.click();
									} else {
										console.log("Continue button is still disabled - may need more seats selected");
									}
								} else {
									console.log("Continue button not found");
								}
							}, 1000);

							return selectedSeats > 0;
						}

						selectSeats();
						return true;

					} catch (e) {
						console.error("Seat selection error:", e);
						return false;
					}
				})();`

				if err := chromedp.Run(searchCtx, chromedp.Evaluate(seatSelectionJS, &seatsSelected)); err != nil {
					log.Printf("Seat selection error: %v\n", err)
				} else {
					log.Println("Seat selection process initiated")
				}

				// Wait additional time for the seat selection and continue button click
				log.Println("Waiting for seat selection to complete...")
				time.Sleep(8 * time.Second)

				// Check if we've progressed past seat selection
				var progressedPastSeats bool
				chromedp.Run(searchCtx, chromedp.Evaluate(`
					// Check if we're past the seat selection stage
					document.URL.includes('payment') ||
					document.URL.includes('checkout') ||
					document.URL.includes('confirm') ||
					document.URL.includes('passenger') ||
					document.querySelector('[class*="payment"]') !== null ||
					document.querySelector('[class*="checkout"]') !== null ||
					document.querySelector('[class*="passenger"]') !== null ||
					!document.body.innerText.toLowerCase().includes('choose seat')
				`, &progressedPastSeats))

				if progressedPastSeats {
					log.Println("Successfully progressed past seat selection")
				} else {
					log.Println("Still on seat selection page - may need manual intervention")

					// Try one more time to click continue if seats were selected
					var retryClick bool
					chromedp.Run(searchCtx, chromedp.Evaluate(`
						(() => {
							let continueBtn = document.querySelector('button[class*="continue"], button[class*="purchase"], input[type="submit"]');
							if (continueBtn && !continueBtn.disabled) {
								continueBtn.click();
								return true;
							}
							return false;
						})();
					`, &retryClick))

					if retryClick {
						log.Println("Retry continue button click successful")
						time.Sleep(3 * time.Second)
					}
				}

			} else {
				log.Println("Coach selection page did not load as expected - user may need to continue manually")
			}

			// Add instruction to the message about manual completion
			messageBody += "\nBooking process initiated. Please complete the remaining steps manually in the opened browser tab.\n"
			messageBody += "The tab will remain open for you to finish the booking process.\n"

			log.Println("Booking process initiated. Tab will remain open for manual completion.")
			log.Println("Complete message:", messageBody)

			// Don't return yet - keep the context alive by continuing the loop
			// This prevents the tab from closing prematurely
			// Only return after a longer wait to ensure user has time to complete booking
			log.Println("Keeping tab open for manual booking completion...")

			for i := 0; i < 120; i++ { // Keep alive for 2 minutes
				time.Sleep(1 * time.Second)

				// Check if booking is completed by looking for success page
				var bookingCompleted bool
				err := chromedp.Run(searchCtx, chromedp.Evaluate(`
					document.URL.includes('success') || 
					document.URL.includes('confirmation') ||
					document.querySelector('[class*="success"]') !== null ||
					document.querySelector('[class*="confirmed"]') !== null ||
					document.body.innerText.toLowerCase().includes('booking confirmed') ||
					document.body.innerText.toLowerCase().includes('ticket booked')
				`, &bookingCompleted))

				if err == nil && bookingCompleted {
					log.Println("Booking appears to be completed successfully!")
					messageBody += "Booking completed successfully!\n"
					break
				}

				// Every 30 seconds, log that we're still waiting
				if i > 0 && i%30 == 0 {
					log.Printf("Still waiting for booking completion... (%d seconds elapsed)", i)
				}
			}

			return messageBody, seatFound
		}

		attemptNo++
		log.Println("Search Ended - Attempt:", attemptNo)
		time.Sleep(constants.SEARCH_DELAY_IN_SEC * time.Second)
	}
}

func buildSeatBookingJS(trainName, selectedClass string) string {
	return `(() => {
            const headers = Array.from(document.querySelectorAll("h2"));
            const header = headers.find((h) =>
                h.innerText.includes("` + trainName + `")
            );
            if (!header) throw new Error("Header not found");
            const appSingleTrip = header.closest("app-single-trip");
            if (!appSingleTrip) throw new Error("Parent component not found");

            // Filter single-seat-class divs by the text content of the seat-class-name span
            const seatClassDivs = Array.from(
                appSingleTrip.querySelectorAll(".single-seat-class")
            );

            let bookNowBtn;

			let seatType;
            seatType = "` + selectedClass + `";
            let seatDiv = seatClassDivs.find((div) => {
            	let seatNameSpan = div.querySelector(".seat-class-name");
                return seatNameSpan && seatNameSpan.innerText.trim() === seatType;
            });
            //throw new Error('Seat class div not found');

             // Find and click the book now button within the specific seat class div
             bookNowBtn = seatDiv.querySelector(".book-now-btn-wrapper .book-now-btn");

            if (!bookNowBtn)
                throw new Error("Book now button not found for All given Types" + seatType);

            bookNowBtn.click();

            const waitForSelectBogie = new Promise((resolve, reject) => {
                setTimeout(() => {
                const bogieSelection = document.getElementById("select-bogie");
                if (!bogieSelection)
                    reject(new Error("Bogie selection dropdown not found"));

                const extractNumber = (text) => {
                    const match = text.match(/\d+/);
                    return match ? parseInt(match[0]) : 0;
                };

                const options = Array.from(bogieSelection.options);
                const highestOption = options.reduce((highest, current) => {
                    const highestNumber = extractNumber(highest.text);
                    const currentNumber = extractNumber(current.text);
                    return currentNumber > highestNumber ? current : highest;
                }, options[0]);

                const coachWithHighestSeat = highestOption.text.split(" - ")[0];

                const coachOption = Array.from(bogieSelection.options).find((option) =>
                    option.text.includes(coachWithHighestSeat)
                );

                bogieSelection.value = coachOption.value;
                bogieSelection.dispatchEvent(new Event("change", { bubbles: true }));

                resolve(coachWithHighestSeat);
                }, 500); // Delay of 500 milliseconds
            });

            const clickSeatButtons = (coachWithHighestSeat) => {
                return new Promise((resolve, reject) => {
                setTimeout(() => {
                    const clickSeatButton = (seatNumber) => {
						const selector =
							'.btn-seat.seat-available[title^="' +
							coachWithHighestSeat +
							'-"][title$="-' +
							seatNumber +
							'"]';
						const seatButton = document.querySelector(selector);
	
						if (seatButton) {
							seatButton.click();
							return true; // Seat button found and clicked
						}
						return false; // Seat button not found
                    };
					
					if("` + arguments.SEAT_FACE + `".includes("Towards")) {
                    	let seatNumber = 1;
                    	let seatCount = parseInt(
                    	"` + strconv.Itoa(int(arguments.SEAT_COUNT)) + `"
                    	);

                    	// Loop to find and click on seat buttons
                    	while (seatCount > 0) {
                    	if (clickSeatButton(seatNumber)) {
                        	seatCount--;
                    	}
                    	seatNumber++; // Increment the seat number for the next iteration
                    	}
				    } else {
                    	let seatNumber = 100;
                    	let seatCount = parseInt(
                    	"` + strconv.Itoa(int(arguments.SEAT_COUNT)) + `"
                    	);

                    	// Loop to find and click on seat buttons
                    	while (seatCount > 0) {
                    	if (clickSeatButton(seatNumber)) {
                        	seatCount--;
                    	}
                    	seatNumber--; // Decrement the seat number for the next iteration
                    	}
				    }

                    resolve(); // Resolve the promise after clicking on seats
                }, 100); // Delay of 100 milliseconds
                });
            };
            
            waitForSelectBogie
                .then((coachWithHighestSeat) => {
                clickSeatButtons(coachWithHighestSeat)
                    .then(() => {
                    // After clicking on seats, find and click the "Continue Purchase" button
						let purchasePage = parseInt("` + strconv.Itoa(int(arguments.GO_TO_BOOK_PAGE)) + `");
						if(purchasePage == 1) {
							setTimeout(() => {
						   		const continueButton = document.querySelector(".continue-btn");
						   		if (!continueButton)
						   		throw new Error("Continue Purchase button not found");
						   		continueButton.click();
							}, 100); // Delay of 100 milliseconds after clicking on seats
						}
                    })
                    .catch((error) => {
                    console.error(error); // Handle any errors from clicking on seat buttons
                    });
                })
                .catch((error) => {
                console.error(error); // Handle any errors from selecting the bogie
                });

            return true;
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
		messageBody += "Train Name:" + selectedSpecificTrain + "\n"
		messageBody += "Seat Class:" + selectedClass + "\n"
		messageBody += "Go to the opened tab in your chrome browser and complete purchase.\n"
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

// autoLogin tries to populate username & password fields then click login submit button using richer event simulation.
func autoLogin(ctx context.Context) error {
	username := constants.LOGIN_UID_VALUE
	password := constants.LOGIN_PASS_VALUE

	// JS helper now triggers keydown/keypress/input/change/blur events so Angular/React forms update state.
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
		chromedp.Sleep(1200*time.Millisecond), // give time for redirect
	); err != nil {
		return err
	}

	log.Printf("AutoLogin -> userSel:%s passSel:%s clicked:%v btnDisabled:%v\n", result.UserSel, result.PassSel, result.Clicked, result.BtnDisabled)

	// If button not clicked or selectors null, signal error so fallback path can attempt.
	if !result.Clicked || result.UserSel == "" || result.PassSel == "" {
		return fmt.Errorf("autologin incomplete (clicked=%v userSel=%s passSel=%s)", result.Clicked, result.UserSel, result.PassSel)
	}
	return nil
}
