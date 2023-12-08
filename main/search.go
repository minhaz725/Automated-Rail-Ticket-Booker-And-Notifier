package main

import (
	"Rail-Ticket-Notifier/utils/constants"
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"log"
	"strconv"
	"strings"
	"time"
)

func performSearch(url string, ctx context.Context) string {
	log.Println(url)
	if err := chromedp.Run(ctx, chromedp.Navigate(url)); err != nil {
		log.Fatal(err)
	}

	// Wait for some time (adjust this as needed) to ensure the page has loaded
	// You can use chromedp.Sleep or chromedp.WaitEvent for this purpose
	chromedp.Sleep(5 * time.Second)
	fmt.Println("Search Ended")
	// Extract the page content after it has loaded
	var pageContent string
	if err := chromedp.Run(ctx, chromedp.InnerHTML("html", &pageContent)); err != nil {
		log.Fatal(err)
	}
	// Process and print the page content
	//log.Println(pageContent)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageContent))
	if err != nil {
		log.Fatal(err)
	}
	messageBody := ""
	doc.Find(".single-trip-wrapper").Each(func(i int, element *goquery.Selection) {
		showTrain := false

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
		if showTrain {
			trainName := element.Find(".trip-name h2").Text()
			//fmt.Println("Train Name:", trainName)
			messageBody = messageBody + "Train Name:" + trainName + "\n"
		}

		// Extract the seat numbers
		element.Find(".seat-available-wrap .all-seats").Each(func(j int, seatElement *goquery.Selection) {
			seatCountStr := seatElement.Text()
			seatCount, _ := strconv.ParseUint(seatCountStr, 10, 0)
			if uint(seatCount) > constants.MIN_SEAT_COUNT {
				//fmt.Println("Seat Count:", seatCount)
				messageBody = messageBody + "Seat Count:" + strconv.FormatUint(seatCount, 10) + "\n"
			}
		})
	})
	return messageBody
}
