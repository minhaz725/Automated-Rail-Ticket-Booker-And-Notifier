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
	attemptNo := 0
	openBrowser := false
	url := originalUrl
	altUrl := ""
	searchAltUrl := false
	selectedSpecificTrain := ""
	selectedClass := ""
	var availableSeatClassArray []string
	showTrain := false
	messageBodyUpdated := false
	messageBody := ""
	loadTimer := 4 * time.Second

	loginInitCtx, cancel := chromedp.NewRemoteAllocator(context.Background(), constants.DEBUG_CHROME_URL)
	loginCtx, cancel := chromedp.NewContext(loginInitCtx)
	defer cancel()

	// Login if needed
	err := chromedp.Run(loginCtx,
		emulation.SetUserAgentOverride("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
		chromedp.Navigate(constants.LOGIN_URL),
		chromedp.WaitReady("body"),
	)
	if err != nil {
		log.Fatal("Failed to navigate:", err)
	}

	// Check if redirected to homepage (already logged in)
	var currentURL string
	err = chromedp.Run(loginCtx,
		chromedp.Location(&currentURL),
	)
	if err != nil {
		log.Fatal("Failed to get current URL:", err)
	}

	// Only perform login if we're still on login page
	if currentURL == constants.LOGIN_URL {
		err = chromedp.Run(loginCtx,
			chromedp.WaitVisible(`#loginForm`),
			//chromedp.Sleep(100*time.Millisecond),
			chromedp.Click(`button[type="submit"].login-form-submit-btn`),
		)
		if err != nil {
			log.Fatal("Failed to login:", err)
		}
		log.Println("Successfully logged in!")
	} else {
		log.Println("Already logged in.")
	}

	if strings.EqualFold(arguments.FROM, "Dhaka") {
		altUrl = arguments.GenerateAltURL()
		searchAltUrl = true
		log.Println("Searching also from Biman Bandar station, as starting station is Dhaka.")
	}

	for {
		log.Println("Search Started")
		url = funcName(originalUrl, searchAltUrl, attemptNo, url, altUrl)

		var initialCtx context.Context
		var cancel context.CancelFunc
		var ctx context.Context

		if openBrowser {
			initialCtx, cancel = chromedp.NewRemoteAllocator(context.Background(), constants.DEBUG_CHROME_URL)
			ctx, cancel = chromedp.NewContext(initialCtx)
		} else {
			initialCtx, cancel = chromedp.NewContext(context.Background())
			ctx, cancel = chromedp.NewContext(initialCtx)
		}

		err := chromedp.Run(ctx,
			emulation.SetUserAgentOverride("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
			chromedp.Navigate(url),
			chromedp.Sleep(loadTimer),
			chromedp.WaitVisible(`button.modify_search.mod_search`),
			chromedp.WaitVisible(`/privacy-policy`),
		)

		if err != nil {
			if strings.Contains(err.Error(), "net::ERR_INTERNET_DISCONNECTED") {
				log.Println("Can't Connect to Network, check your internet. Retrying...")
			} else {
				log.Println("Browser error:", err)
			}

			// Increase load time if needed
			if loadTimer < 20*time.Second {
				loadTimer = loadTimer + 2*time.Second
				fmt.Println("Page Load Time Increased to: ", loadTimer)
			}

			cancel()
			time.Sleep(5 * time.Second)
			continue
		}
		loadTimer = 3 * time.Second
		// Wait for some time (adjust this as needed) to ensure the page has loaded
		// You can use chromedp.Sleep or chromedp.WaitEvent for this purpose
		chromedp.Sleep(2 * time.Second)

		// Extract the page content after it has loaded
		var pageContent string
		if err := chromedp.Run(ctx, chromedp.InnerHTML("html", &pageContent)); err != nil {
			log.Fatal(err)
		}
		// Process and print the page content

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageContent))
		if err != nil {
			log.Fatal(err)
		}

		//renderedHTML := printHtml(err, doc)
		//generateHtmlFile(err, renderedHTML)
		if !openBrowser {
			doc.Find(".single-trip-wrapper").Each(func(i int, element *goquery.Selection) {

				trainName := ""
				trainName = element.Find(".trip-name h2").Text()
				if !strings.Contains(trainName, arguments.SPECIFIC_TRAIN_ARRAY[0]) {
					return
				}
				fmt.Println("Search URL: ", url)
				fmt.Println("Train Name:", trainName)
				classFound := false
				element.Find(".seat-classes-row .seat-class-name, .seat-classes-row .all-seats").Each(func(i int, s *goquery.Selection) {
					// Check if the current element is a .seat-class-name or .all-seats
					className := ""
					if s.HasClass("seat-class-name") {
						// This is a .seat-class-name element
						className = s.Text()
						for k := 0; k < len(arguments.SEAT_TYPE_ARRAY); k++ {
							if className == arguments.SEAT_TYPE_ARRAY[k] {
								fmt.Print("Class Name:", className)
								availableSeatClassArray = append(availableSeatClassArray, className)
								classFound = true
							}
						}
					}
					if s.HasClass("all-seats") {
						// This is an .all-seats element
						//fmt.Println(className)
						if classFound {
							seatCountStr := s.Text()
							seatCount, _ := strconv.ParseUint(seatCountStr, 10, 0)
							fmt.Println(" Seat Count:", seatCount)
							classFound = false
							if uint(seatCount) >= arguments.SEAT_COUNT {
								showTrain = true
								selectedSpecificTrain = trainName
								return
							} else {
								availableSeatClassArray = availableSeatClassArray[:len(availableSeatClassArray)-1]
							}
						}

					}
				})
			})
		}

		jsCode := `(() => {
            const headers = Array.from(document.querySelectorAll("h2"));
            const header = headers.find((h) =>
                h.innerText.includes("` + selectedSpecificTrain + `")
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
                }, 500); // Delay of 500 milliseconds (1 second)
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
                a;
                })
                .catch((error) => {
                console.error(error); // Handle any errors from selecting the bogie
                });

            return true;
            })();
		`

		if showTrain {
			if openBrowser == false {
				fmt.Println(availableSeatClassArray)
				selectedClass = utils.FindFirstMatch(availableSeatClassArray, arguments.SEAT_TYPE_ARRAY)
				if selectedClass == "" {
					log.Fatal("class selection error")
				}
				messageBody, messageBodyUpdated = updateMessageBody(messageBodyUpdated, messageBody, selectedSpecificTrain, selectedClass)
				openBrowser = true
				continue
				// open browser in next iteration if conditions are matched
			}

			log.Println(url)
			var success bool
			err := chromedp.Run(ctx,
				chromedp.Evaluate(jsCode, &success),
				chromedp.Sleep(2*time.Second),
			)
			if err != nil {
				log.Fatal(err)
			}
			//cancel() // Cancel the context explicitly when done
			return messageBody, showTrain
		}
		// Cancel the context to end this loop's context
		cancel()

		attemptNo++
		log.Println("Search Ended")
		log.Println("Attempt Number: ", attemptNo)
		fmt.Println()

		time.Sleep(constants.SEARCH_DELAY_IN_SEC * time.Second)
	}
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
		messageBody = messageBody + "Train Name:" + selectedSpecificTrain + "\n"
		messageBody = messageBody + "Seat Class:" + selectedClass + "\n"
		messageBody = messageBody + "Go to the opened tab in your chrome browser and complete purchase." + "\n"
		messageBodyUpdated = true
	}
	return messageBody, messageBodyUpdated
}

func generateHtmlFile(err error, renderedHTML string) {
	//Write the rendered HTML to a file
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
