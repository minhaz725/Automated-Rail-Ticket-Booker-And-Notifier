package main

import (
	"Rail-Ticket-Notifier/utils/constants"
	"io"
	"os"
)

func main() {
	seatBooker, _ := os.Open("seatBooker.js")
	defer seatBooker.Close()

	seatBookerFunctionInBytes, _ := io.ReadAll(seatBooker)

	// Create the URL
	url := constants.BASE_URL + constants.FROM + constants.TO + constants.DATE + constants.CLASS
	messageBody, send := performSearch(url, string(seatBookerFunctionInBytes))
	if send {
		sendEmail(messageBody)
	}
}
