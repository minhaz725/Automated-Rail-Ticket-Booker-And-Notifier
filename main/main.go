package main

import (
	"Rail-Ticket-Notifier/utils/arguments"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

// sample arg: go run . -from "Khulna" -to "Dhaka" -date "26-Mar-2024" -seatCount 4 -seatTypes "SNIGDHA,S_CHAIR" -trains "TURNA,SUNDARBAN,BAZAR"
func main() {
	window, fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry := InitializeUIAndForm()

	// Create a form
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "From", Widget: fromEntry},
			{Text: "To", Widget: toEntry},
			{Text: "Date", Widget: dateEntry},
			{Text: "Seat Count", Widget: seatCountEntry},
			{Text: "Seat Types", Widget: seatTypesEntry},
			{Text: "Trains", Widget: trainsEntry},
		},
		OnSubmit: func() {
			// Attempt to parse seatCount to uint
			value, err := strconv.ParseUint(seatCountEntry.Text, 10, 32)
			if err != nil {
				log.Println("Error parsing seatCount:", err)
				return
			}

			// Update global variables in the arguments package
			arguments.UpdateArguments(
				fromEntry.Text,
				toEntry.Text,
				dateEntry.Text,
				strconv.FormatUint(value, 10), // Convert uint64 back to string for UpdateArguments
				strings.Split(seatTypesEntry.Text, ","),
				strings.Split(trainsEntry.Text, ","),
			)

			// Proceed with your application logic in a separate goroutine
			go coreOperation()
		},
	}

	//form.Disable()
	// Add the form to the window
	window.SetContent(container.NewVBox(form))

	window.ShowAndRun()
}

func coreOperation() {
	seatBooker, _ := os.Open("seatBooker.js")
	defer seatBooker.Close()

	seatBookerFunctionInBytes, _ := io.ReadAll(seatBooker)

	messageBody, send := performSearch(arguments.GetURL(), string(seatBookerFunctionInBytes))
	if send {
		sendEmail(messageBody)
	}
}
