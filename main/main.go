package main

import (
	"Rail-Ticket-Notifier/utils/constants"
)

func main() {
	// Create a context
	// Create the URL
	url := constants.BASE_URL + constants.FROM + constants.TO + constants.DATE + constants.CLASS
	messageBody, send := performSearch(url)
	if send {
		sendEmail(messageBody)
	}
}
