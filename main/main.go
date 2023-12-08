package main

import (
	"Rail-Ticket-Notifier/utils/constants"
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
)

func main() {
	// Create a context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Navigate to the URL
	fmt.Println("Search Started")
	url := constants.BASE_URL + constants.FROM + constants.TO + constants.DATE + "17-Dec-2023" + constants.CLASS
	messageBody, send := performSearch(url, ctx)
	fmt.Println(messageBody)
	if send {
		sendEmail(messageBody)
	}
}
