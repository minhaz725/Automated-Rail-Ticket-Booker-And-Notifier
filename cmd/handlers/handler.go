package handlers

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/internal/notifier"
	"Rail-Ticket-Notifier/internal/search"
	"fyne.io/fyne/v2/widget"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

func HandleFormSubmission(fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry *widget.Entry) {
	// Attempt to parse seatCount to uint
	seatCountVal, err := strconv.ParseUint(seatCountEntry.Text, 10, 32)
	if err != nil {
		log.Println("Error parsing seatCount:", err)
		return
	}

	// Update global variables in the arguments package
	arguments.UpdateArguments(
		fromEntry.Text,
		toEntry.Text,
		dateEntry.Text,
		uint(seatCountVal),
		strings.Split(seatTypesEntry.Text, ","),
		strings.Split(trainsEntry.Text, ","),
	)

	// Proceed with your application logic in a separate goroutine
	go handleCoreOperation()
}

func handleCoreOperation() {
	seatBooker, _ := os.Open("seatBooker.js")
	defer seatBooker.Close()

	seatBookerFunctionInBytes, _ := io.ReadAll(seatBooker)

	messageBody, send := search.PerformSearch(arguments.GenerateURL(), string(seatBookerFunctionInBytes))
	if send {
		notifier.SendEmail(messageBody)
	}
}
