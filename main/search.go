package main

import (
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

func performSearch(url string) (string, bool) {
	attemptNo := 0
	for {
		fmt.Println("Search Started")
		//start chrome.exe --remote-debugging-port=9222
		initialCtx, cancel := chromedp.NewContext(context.Background())
		ctx, cancel := chromedp.NewContext(initialCtx)

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
		doc.Find(".single-trip-wrapper").Each(func(i int, element *goquery.Selection) {

			// Filter train by Minimum number of seats
			element.Find(".seat-available-wrap .all-seats").Each(func(j int, seatElement *goquery.Selection) {
				seatCountStr := seatElement.Text()
				seatCount, _ := strconv.ParseUint(seatCountStr, 10, 0)
				if uint(seatCount) >= constants.SEAT_COUNT {
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
				seatCountStr := seatElement.Text()
				seatCount, _ := strconv.ParseUint(seatCountStr, 10, 0)
				if uint(seatCount) >= constants.SEAT_COUNT {
					//fmt.Println("Seat Count:", seatCount)
					if strings.Contains(trainName, constants.SPECIFIC_TRAIN) {
						specificTrain = true
					}
					messageBody = messageBody + "Seat Count:" + strconv.FormatUint(seatCount, 10) + "\n"
				}
			})
		})
		fmt.Println(messageBody)
		if showTrain && specificTrain {
			log.Println(url)
			//var example string
			//err := chromedp.Run(ctx,
			//	chromedp.Evaluate(`(() => {
			//		const headers = Array.from(document.querySelectorAll('h2'));
			//		const header = headers.find(h => h.innerText.includes(constants.SPECIFIC_TRAIN));
			//		if (!header) throw new Error('Header not found');
			//		const appSingleTrip = header.closest('app-single-trip');
			//		if (!appSingleTrip) throw new Error('Parent component not found');
			//
			//		// Filter single-seat-class divs by the text content of the seat-class-name span
			//		const seatClassDivs = Array.from(appSingleTrip.querySelectorAll('.single-seat-class'));
			//		const seatDiv = seatClassDivs.find(div => {
			//		const seatNameSpan = div.querySelector('.seat-class-name');
			//		return seatNameSpan && seatNameSpan.innerText.trim() === '`+constants.SEAT_TYPE+`';
			//		});
			//		if (!seatDiv) throw new Error('Seat class div not found');
			//
			//		// Find and click the book now button within the specific seat class div
			//		const bookNowBtn = seatDiv.querySelector('.book-now-btn-wrapper .book-now-btn');
			//		if (!bookNowBtn) throw new Error('Book now button not found');
			//		bookNowBtn.click();
			//
			//		setTimeout(() => {
			// 		// Find the select element
			//			const bogieSelection = document.getElementById('select-bogie');
			//			if (!bogieSelection) throw new Error('Bogie selection dropdown not found');
			//
			//			// Find the option that contains the coach numb
			//			const coachOption = Array.from(bogieSelection.options).find(option => option.text.includes('`+constants.COACH_NUMB+`'));
			//			if (!coachOption) throw new Error('Option with text `+constants.COACH_NUMB+` not found');
			//
			//			// Set the selected option to the one found
			//			bogieSelection.value = coachOption.value;
			//			// Dispatch an input event to simulate user interaction
			//			bogieSelection.dispatchEvent(new Event('change', { bubbles: true }));
			//
			//			setTimeout(() => {
			//
			//				const seatOne = document.querySelector('.btn-seat.seat-available[title="`+constants.COACH_NUMB+constants.SEAT_ONE_NUMB+`"]');
			//				if (!seatOne) throw new Error('seatOne button not found');
			//				seatOne.click();
			//				const seatTwo = document.querySelector('.btn-seat.seat-available[title="`+constants.COACH_NUMB+constants.SEAT_TWO_NUMB+`"]');
			//				if (!seatTwo) throw new Error('seatTwo button not found');
			//				seatTwo.click();
			//					setTimeout(() => {
			//						const continueButton = document.querySelector('.continue-btn');
			//						if (!continueButton) throw new Error('Continue Purchase button not found');
			//						continueButton.click();
			//					},  500);
			//				},  500); // Delay of  1000 milliseconds (1 second)
			//			},  500);
			//
			//	})()`, &example),
			//	chromedp.Sleep(2*time.Second),
			//)
			//if err != nil {
			//	log.Fatal(err)
			//}
			cancel() // Cancel the context explicitly when done
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
