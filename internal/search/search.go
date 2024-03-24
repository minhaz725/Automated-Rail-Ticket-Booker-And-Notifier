package search

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/utils/constants"
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func PerformSearch(url string, seatBookerFunction string) (string, bool) {
	attemptNo := 0
	openBrowser := false
	for {
		fmt.Println("Search Started")
		//start chrome.exe --remote-debugging-port=9222
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

		if err :=
			chromedp.Run(ctx,
				chromedp.Navigate(url),
				chromedp.Sleep(3*time.Second),
				chromedp.WaitVisible(`button.modify_search.mod_search`),
				chromedp.WaitVisible(`/privacy-policy`)); err != nil {
			log.Fatal(err)
		}

		// Wait for some time (adjust this as needed) to ensure the page has loaded
		// You can use chromedp.Sleep or chromedp.WaitEvent for this purpose

		chromedp.Sleep(2 * time.Second)

		fmt.Println("Search Ended")
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

		messageBody := "Follow this URL to purchase: " + url + "\n"
		showTrain := false
		specificTrain := false
		selectedSpecificTrain := ""
		doc.Find(".single-trip-wrapper").Each(func(i int, element *goquery.Selection) {

			// Filter train by Minimum number of seats
			element.Find(".seat-available-wrap .all-seats").Each(func(j int, seatElement *goquery.Selection) {
				//if specificTrain {
				//	return
				//}
				seatCountStr := seatElement.Text()
				seatCount, _ := strconv.ParseUint(seatCountStr, 10, 0)
				if uint(seatCount) >= arguments.SEAT_COUNT {
					showTrain = true
					return
				}
			})

			// Extract the train name
			trainName := ""
			if showTrain {
				trainName = element.Find(".trip-name h2").Text()
				//fmt.Println("Train Name:", trainName)
				messageBody = messageBody + "Train Name:" + trainName + "\n"
			}

			// Extract the seat numbers
			element.Find(".seat-available-wrap .all-seats").Each(func(j int, seatElement *goquery.Selection) {
				//if specificTrain {
				//	return
				//}
				seatCountStr := seatElement.Text()
				seatCount, _ := strconv.ParseUint(seatCountStr, 10, 0)
				if uint(seatCount) >= arguments.SEAT_COUNT {
					//fmt.Println("Seat Count:", seatCount)
					for _, specificTrainName := range arguments.SPECIFIC_TRAIN_ARRAY {
						// Check if trainName contains the specific train name
						if strings.Contains(trainName, specificTrainName) {
							specificTrain = true
							selectedSpecificTrain = specificTrainName
							//return
						}
					}
					messageBody = messageBody + "Seat Count:" + strconv.FormatUint(seatCount, 10) + "\n"
				}
			})
		})
		fmt.Println(messageBody)

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

            let seatTypeArrayLength = parseInt(
                "` + strconv.Itoa(len(arguments.SEAT_TYPE_ARRAY)) + `"
            );
            let bookNowBtn;

            for (let i = 0; i < seatTypeArrayLength; i++) {
                let seatType;
                if (i == 0) seatType = "` + arguments.SEAT_TYPE_ARRAY[0] + `";
                if (i == 1) seatType = "` + arguments.SEAT_TYPE_ARRAY[1] + `";
                //if(i==2) seatType = '` + /*constants.SEAT_TYPE_ARRAY[2] +*/ `'
                let seatDiv = seatClassDivs.find((div) => {
                let seatNameSpan = div.querySelector(".seat-class-name");
                return seatNameSpan && seatNameSpan.innerText.trim() === seatType;
                });
                //throw new Error('Seat class div not found');

                // Find and click the book now button within the specific seat class div
                bookNowBtn = seatDiv.querySelector(".book-now-btn-wrapper .book-now-btn");
                if (bookNowBtn != null) break;
            }

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

                    resolve(); // Resolve the promise after clicking on seats
                }, 500); // Delay of 500 milliseconds
                });
            };
            
            waitForSelectBogie
                .then((coachWithHighestSeat) => {
                clickSeatButtons(coachWithHighestSeat)
                    .then(() => {
                    // After clicking on seats, find and click the "Continue Purchase" button
                    //setTimeout(() => {
                    //    const continueButton = document.querySelector(".continue-btn");
                    //    if (!continueButton)
                    //    throw new Error("Continue Purchase button not found");
                    //    continueButton.click();
                    //}, 500); // Delay of 500 milliseconds after clicking on seats
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

		if showTrain && specificTrain {
			if openBrowser == false {
				openBrowser = true
				continue
				// open browser if conditions are matched
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
		fmt.Println("Attempt Number: ", attemptNo)
		time.Sleep(constants.SEARCH_DELAY_IN_SEC * time.Second)
	}
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
	fmt.Println("HTML file generated:", filename)
}

func printHtml(err error, doc *goquery.Document) string {
	renderedHTML, err := doc.Html()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(renderedHTML)
	return renderedHTML
}
