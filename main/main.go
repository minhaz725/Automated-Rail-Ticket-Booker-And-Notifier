package main

import (
	"Rail-Ticket-Notifier/utils/constants"
	"fmt"
)

func main() {
	// Create a context
	// Navigate to the URL
	fmt.Println("Search Started")
	date := "18-Jan-2024"
	//date := "01-Jan-2024"
	url := constants.BASE_URL + constants.FROM + constants.TO + constants.DATE + date + constants.CLASS
	messageBody, send := performSearch(url)
	if send {
		sendEmail(messageBody, date)
	}
}
