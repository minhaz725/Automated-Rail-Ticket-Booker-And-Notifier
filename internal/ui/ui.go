package ui

import (
	"Rail-Ticket-Notifier/cmd/handlers"
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/utils/constants"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"strconv"
	"strings"
)

func InitializeUIAndForm() (fyne.Window, *widget.Label, *widget.Entry, *widget.Entry, *widget.Entry, *widget.Entry, *widget.Entry, *widget.Entry) {
	a := app.New()

	window := a.NewWindow("Automated Rail Ticket Booker & Notifier")
	window.Resize(fyne.NewSize(500, 400))
	introLabel := widget.NewLabel(constants.INTRO_MSG)
	// Create form fields with default values
	fromEntry := widget.NewEntry()
	fromEntry.SetText(arguments.FROM) // Default value from arguments package
	toEntry := widget.NewEntry()
	toEntry.SetText(arguments.TO) // Default value from arguments package
	dateEntry := widget.NewEntry()
	dateEntry.SetText(arguments.DATE) // Default value from arguments package
	seatCountEntry := widget.NewEntry()
	seatCountEntry.SetText(strconv.Itoa(int(arguments.SEAT_COUNT))) // Convert uint to string
	seatTypesEntry := widget.NewEntry()
	seatTypesEntry.SetText(strings.Join(arguments.SEAT_TYPE_ARRAY, ",")) // Default value from arguments package
	trainsEntry := widget.NewEntry()
	trainsEntry.SetText(strings.Join(arguments.SPECIFIC_TRAIN_ARRAY, ","))

	content := container.NewVBox(introLabel, fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry)

	window.SetContent(content)

	return window, introLabel, fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry
}

func CreateForm(fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry *widget.Entry, submitButton *widget.Button) *fyne.Container {
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "From", Widget: fromEntry},
			{Text: "To", Widget: toEntry},
			{Text: "Date", Widget: dateEntry},
			{Text: "Seat Count", Widget: seatCountEntry},
			{Text: "Seat Types", Widget: seatTypesEntry},
			{Text: "Trains", Widget: trainsEntry},
		},
	}

	submitButton.OnTapped = func() {
		handlers.HandleFormSubmission(fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry, submitButton)
	}

	return container.NewVBox(
		form,
		submitButton,
	)
}

func GetSubmitButton() *widget.Button {
	submitButton := widget.NewButton("Start Search", func() {})
	return submitButton
}
