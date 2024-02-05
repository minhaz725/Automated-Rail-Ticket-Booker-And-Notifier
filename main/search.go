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
		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", false),
		)
		// Create a context with options.
		initialCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)

		//start chrome.exe --remote-debugging-port=9222
		//ctx, cancel := chromedp.NewRemoteAllocator(context.Background(), "http://localhost:9222")
		ctx, cancel := chromedp.NewContext(initialCtx)

		if err :=
			chromedp.Run(ctx,
				chromedp.Navigate(url),
				chromedp.WaitVisible(`button.modify_search.mod_search`),
				chromedp.WaitVisible(`/privacy-policy`)); err != nil {
			log.Fatal(err)
		}

		// Wait for some time (adjust this as needed) to ensure the page has loaded
		// You can use chromedp.Sleep or chromedp.WaitEvent for this purpose

		chromedp.Sleep(7 * time.Second)

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
				if uint(seatCount) > constants.MIN_SEAT_COUNT {
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
				if uint(seatCount) > constants.MIN_SEAT_COUNT {
					//fmt.Println("Seat Count:", seatCount)
					if trainName == constants.SPECIFIC_TRAIN {
						specificTrain = true
					}
					messageBody = messageBody + "Seat Count:" + strconv.FormatUint(seatCount, 10) + "\n"
				}
			})
		})
		fmt.Println(messageBody)
		if showTrain && specificTrain {
			log.Println(url)
			var example string
			err := chromedp.Run(ctx,
				chromedp.Evaluate(`(() => {
					const headers = Array.from(document.querySelectorAll('h2'));
					const header = headers.find(h => h.innerText === '`+constants.SPECIFIC_TRAIN+`');
					if (!header) throw new Error('Header not found');
					const btn = header.closest('app-single-trip').querySelector('.trip-details-btn');
					if (!btn) throw new Error('Button not found');
					btn.click();
					})()`,
					&example),

				chromedp.Sleep(10*time.Second),
			)
			if err != nil {
				log.Fatal(err)
			}
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
