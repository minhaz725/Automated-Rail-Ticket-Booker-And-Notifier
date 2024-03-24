package ui

import (
	"Rail-Ticket-Notifier/cmd/handlers"
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/utils/constants"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"
	"strconv"
	"strings"
	"time"
)

func InitializeUIAndForm() (fyne.Window, *widget.Label, *widget.Entry, *widget.Entry, *widget.Entry, *widget.Entry, *widget.Entry, *widget.Entry) {
	a := app.New()

	window := a.NewWindow("Automated Rail Ticket Booker & Notifier")
	window.Resize(fyne.NewSize(800, 600))
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

func CreateForm(fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry *widget.Entry, submitButton *widget.Button, window fyne.Window) *fyne.Container {

	calendar := GetCalendar(func(t time.Time) {
		dateEntry.SetText(t.Format("02-Jan-2006"))
	})

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "From (Capital Case)", Widget: fromEntry},
			{Text: "To (Capital Case)", Widget: toEntry},
			{Text: "Date Of Journey (Choose From Calender)", Widget: dateEntry},
			{Text: "(Only from current date to next 10 days)", Widget: calendar},
			{Text: "Seat Count (1 to Max 4)", Widget: seatCountEntry},
			{Text: "Seat Types (Will Prioritize Serial Wise)", Widget: seatTypesEntry},
			{Text: "Trains (Choose only One.)", Widget: trainsEntry},
		},
	}

	submitButton.OnTapped = func() {
		handlers.HandleFormSubmission(fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry, submitButton, window)
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

func GetCalendar(onSelected func(time.Time)) *xwidget.Calendar {
	startingDate := time.Now()
	return xwidget.NewCalendar(startingDate, onSelected)
}
