package main

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

func main() {
	// Create a context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Navigate to the URL
	fmt.Println("Search Started")
	url := "https://eticket.railway.gov.bd/booking/train/search/en?fromcity=Dhaka&tocity=Chattogram&doj=29-Oct-2023&class=S_CHAIR"
	if err := chromedp.Run(ctx, chromedp.Navigate(url)); err != nil {
		log.Fatal(err)
	}

	// Wait for some time (adjust this as needed) to ensure the page has loaded
	// You can use chromedp.Sleep or chromedp.WaitEvent for this purpose
	chromedp.Sleep(time.Second)
	fmt.Println("Search Ended")
	// Extract the page content after it has loaded
	var pageContent string
	if err := chromedp.Run(ctx, chromedp.InnerHTML("html", &pageContent)); err != nil {
		log.Fatal(err)
	}

	// Process and print the page content
	//fmt.Println(pageContent)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageContent))
	if err != nil {
		log.Fatal(err)
	}

	doc.Find("h2").Each(func(index int, element *goquery.Selection) {
		// Get the text within the h2 element
		trainName := element.Text()

		// Check if the text contains "(number)" and extract the train name
		parts := strings.Split(trainName, "(")
		if len(parts) >= 2 {
			train := strings.TrimSpace(parts[0])

			// Print the train name
			fmt.Println("Train Name:", train)
		}
	})
}
