package handlers

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/internal/models"
	"Rail-Ticket-Notifier/internal/notifier"
	"Rail-Ticket-Notifier/internal/search"
	"Rail-Ticket-Notifier/utils/constants"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func HandleFormSubmission(uiElements models.ElementsOfUI, submitButton *widget.Button) bool {
	// Disable the submit button
	submitButton.Disable()

	// Attempt to parse seatCount to uint
	seatCountVal, err := strconv.ParseUint(uiElements.SeatCountEntry.Text, 10, 32)
	if err != nil {
		log.Println("Error parsing seatCount:", err)
		return false
	}

	// Update global variables in the arguments package
	arguments.UpdateArguments(
		uiElements.FromEntry.Text,
		uiElements.ToEntry.Text,
		uiElements.DateEntry.Text,
		uiElements.EmailEntry.Text,
		uiElements.GoToBookEntry.Checked,
		uint(seatCountVal),
		strings.Split(uiElements.SeatTypesEntry.Text, ","),
		strings.Split(uiElements.TrainsEntry.Text, ","),
	)

	// Proceed with your application logic in a separate goroutine
	successChan := make(chan bool)

	go handleCoreOperation(successChan)

	success := <-successChan
	if success {

		dialog.ShowInformation("Success", constants.OUTRO_SUCCESS_MSG, uiElements.Window)
		log.Println("Success!")
		time.Sleep(10 * time.Second)
		os.Exit(0)
		return true
	} else {
		dialog.ShowInformation("Failed", constants.OUTRO_FAILURE_MSG, uiElements.Window)
		log.Println("Operation failed.")
		time.Sleep(5 * time.Second)
		// Terminate the program
		os.Exit(0)
		return false
	}
}

func handleCoreOperation(successChan chan bool) {
	//seatBooker, _ := os.Open("seatBooker.js")
	//defer seatBooker.Close()
	//todo read js segment from file
	//seatBookerFunctionInBytes, _ := io.ReadAll(seatBooker)
	seatTypeValidate()
	messageBody, send := search.PerformSearch(arguments.GenerateURL(), "string(seatBookerFunctionInBytes)")
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

func seatTypeValidate() {
	if len(arguments.SEAT_TYPE_ARRAY) < 3 {
		// Append "SNIGDHA" and "S_CHAIR" to the slice
		var newArray []string
		for _, seatType := range arguments.SEAT_TYPE_ARRAY {
			if seatType != " " {
				newArray = append(newArray, seatType)
			}
		}
		arguments.SEAT_TYPE_ARRAY = append(arguments.SEAT_TYPE_ARRAY, "SNIGDHA", "S_CHAIR", "AC_B")
	}
}
