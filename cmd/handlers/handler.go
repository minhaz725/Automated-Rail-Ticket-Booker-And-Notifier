package handlers

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/internal/notifier"
	"Rail-Ticket-Notifier/internal/search"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func HandleFormSubmission(fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry *widget.Entry, submitButton *widget.Button, window fyne.Window) bool {
	// Disable the submit button
	submitButton.Disable()

	// Attempt to parse seatCount to uint
	seatCountVal, err := strconv.ParseUint(seatCountEntry.Text, 10, 32)
	if err != nil {
		log.Println("Error parsing seatCount:", err)
		return false
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
	successChan := make(chan bool)

	go handleCoreOperation(successChan)

	success := <-successChan
	if success {

		dialog.ShowInformation("Success", "Operation completed successfully. Application will automatically close in 10 secs.\n"+
			" Go to the opened tab and finish your purchase. Thanks for using the app!", window)
		log.Println("Success!")
		time.Sleep(10 * time.Second)
		os.Exit(0)
		return true
	} else {
		dialog.ShowInformation("Failed", "Operation Failed. Please try again!", window)
		log.Println("Operation failed.")
		time.Sleep(5 * time.Second)
		// Terminate the program
		os.Exit(0)
		return false
	}
}

func handleCoreOperation(successChan chan bool) {
	seatBooker, _ := os.Open("seatBooker.js")
	defer seatBooker.Close()

	seatBookerFunctionInBytes, _ := io.ReadAll(seatBooker)

	messageBody, send := search.PerformSearch(arguments.GenerateURL(), string(seatBookerFunctionInBytes))
	mailSuccess := false
	callSuccess := false
	if send {
		mailSuccess = notifier.SendEmail(messageBody)
		callSuccess = notifier.MakeCall()
	}
	if mailSuccess && callSuccess {
		// Send success status through the channel
		successChan <- true
	} else {
		// Send failure status through the channel
		successChan <- false
	}
}
