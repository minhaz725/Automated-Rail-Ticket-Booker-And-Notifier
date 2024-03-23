package main

import (
	"Rail-Ticket-Notifier/utils/arguments"
	"io"
	"os"
)

// sample arg: go run . -from "Khulna" -to "Dhaka" -date "26-Mar-2024" -seatCount 4 -seatTypes "SNIGDHA,S_CHAIR" -trains "TURNA,SUNDARBAN,BAZAR"
func main() {
	seatBooker, _ := os.Open("seatBooker.js")
	defer seatBooker.Close()

	seatBookerFunctionInBytes, _ := io.ReadAll(seatBooker)

	messageBody, send := performSearch(arguments.GetURL(), string(seatBookerFunctionInBytes))
	if send {
		sendEmail(messageBody)
	}
}
