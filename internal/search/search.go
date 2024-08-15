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
	"strconv"
	"strings"
	"time"
)

func PerformSearch(url string, seatBookerFunction string) (string, bool) {
	attemptNo := 0
	openBrowser := false
	selectedSpecificTrain := ""
	selectedClass := ""
	var availableSeatClassArray []string
	showTrain := false
	messageBodyUpdated := false
	messageBody := ""
	loadTImer := 4 * time.Second

	for {
		log.Println("Search Started")
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
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second+loadTImer) // Set timeout to 30 seconds
		defer cancel()

		err := chromedp.Run(ctxWithTimeout,
			emulation.SetUserAgentOverride("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
			chromedp.Navigate(url),
			chromedp.Sleep(loadTImer),
			chromedp.WaitVisible(`button.modify_search.mod_search`),
			chromedp.WaitVisible(`/privacy-policy`))

		if err != nil {
			if strings.Contains(err.Error(), "net::ERR_INTERNET_DISCONNECTED") {
				log.Println("Can't Connect to Network, check your internet. Retrying...")
			} else if err.Error() == "context deadline exceeded" {
				log.Println("Page load Time exceeded(", loadTImer, "sec) retrying...")
			} else {
				log.Println("Browser didn't start on debug mode, please read the instructions and try again...")
			}
			// increase load time
			if loadTImer < 20*time.Second {
				loadTImer = loadTImer + 2*time.Second
				fmt.Println("Page Load Time Increased to: ", loadTImer)
			}

			fmt.Println()
			cancel()
			time.Sleep(5 * time.Second)
			continue
		}
		loadTImer = 3 * time.Second
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

                const coachWithHighestSeat = 'GA'
                }, 1000); // Delay of 1000 milliseconds (1 second)
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
					
					const seatsToSelect = ['28', '29', '30', '33'];

					// Loop through the seats and attempt to click them
					
					seatsToSelect.forEach(seatNumber => {
						clickSeatButton(seatNumber)
					});

                    resolve(); // Resolve the promise after clicking on seats
                }, 500); // Delay of 500 milliseconds
                });
            };

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
			//return messageBody, showTrain
		}
		// Cancel the context to end this loop's context
		cancel()

		attemptNo++
		time.Sleep(5 * time.Minute)
		log.Println("Search Ended")
		log.Println("Attempt Number: ", attemptNo)
		fmt.Println()

		time.Sleep(constants.SEARCH_DELAY_IN_SEC * time.Second)
	}
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
