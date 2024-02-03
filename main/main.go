package main

import (
	"Rail-Ticket-Notifier/utils/constants"
)

func main() {
	// Create a context
	// Navigate to the URL
	date := "13-Feb-2024"
	//date := "01-Jan-2024"
	url := constants.BASE_URL + constants.FROM + constants.TO + constants.DATE + date + constants.CLASS
	_, send := performSearch(url)
	if send {
		//sendEmail(messageBody, date)
	}
}
