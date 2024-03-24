package ui

import (
	"Rail-Ticket-Notifier/cmd/handlers"
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/internal/models"
	"Rail-Ticket-Notifier/utils/constants"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"
	"strconv"
	"strings"
	"time"
)

func InitializeUIAndForm() models.ElementsOfUI {
	a := app.New()

	window := a.NewWindow("Automated Rail Ticket Booker & Notifier")
	window.Resize(fyne.NewSize(800, 600))

	// welcome popup
	label := widget.NewLabel(constants.INTRO_MSG)
	label.Alignment = fyne.TextAlignLeading // Set text alignment to left
	customDialog := dialog.NewCustom("Welcome", "OK", container.NewVBox(label), window)
	customDialog.Show()

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
	emailEntry := widget.NewEntry()
	emailEntry.SetText(arguments.RECEIVER_EMAIL_ADDRESS)
	phoneEntry := widget.NewEntry()
	phoneEntry.SetText(arguments.PHONE_NUMBER)
	phoneEntry.Disable()
	goToBookEntry := widget.NewCheck("Uncheck this to stay at seat selection page to manually adjust seats.", func(value bool) {})
	goToBookEntry.SetChecked(arguments.GO_TO_BOOK_PAGE != 0)

	content := container.NewVBox(fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry, emailEntry, phoneEntry, goToBookEntry)

	scrollContainer := container.NewVScroll(content)

	scrollContainer.SetMinSize(fyne.NewSize(800, 600)) // Set minimum size to window size

	window.SetContent(scrollContainer)

	uiElements := models.ElementsOfUI{
		Window:         window,
		FromEntry:      fromEntry,
		ToEntry:        toEntry,
		DateEntry:      dateEntry,
		SeatCountEntry: seatCountEntry,
		SeatTypesEntry: seatTypesEntry,
		TrainsEntry:    trainsEntry,
		EmailEntry:     emailEntry,
		PhoneEntry:     phoneEntry,
		GoToBookEntry:  goToBookEntry,
	}

	return uiElements
}

func CreateForm(uiElements models.ElementsOfUI) *fyne.Container {

	calendar := GetCalendar(func(t time.Time) {
		uiElements.DateEntry.SetText(t.Format("02-Jan-2006"))
	})

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "From (Title  Case)", Widget: uiElements.FromEntry},
			{Text: "To (Title  Case)", Widget: uiElements.ToEntry},
			{Text: "Date Of Journey (Choose From Calender)", Widget: uiElements.DateEntry},
			{Text: "(Only from current date to next 10 days)", Widget: calendar},
			{Text: "Seat Count (1 to Max 4)", Widget: uiElements.SeatCountEntry},
			{Text: "Seat Types (Prioritize Serially. All Capitals. 1 to Max 3)", Widget: uiElements.SeatTypesEntry},
			{Text: "Trains (Choose only One. All Capitals)", Widget: uiElements.TrainsEntry},
			{Text: "Email address (To receive mail after done)", Widget: uiElements.EmailEntry},
			{Text: "Phone Number (Currently unavailable)", Widget: uiElements.PhoneEntry},
			{Text: "Go To Book Page", Widget: uiElements.GoToBookEntry},
		},
	}

	submitButton := getSubmitButton()

	submitButton.OnTapped = func() {
		handlers.HandleFormSubmission(uiElements, submitButton)
	}

	return container.NewVBox(
		form,
		submitButton,
	)
}

func getSubmitButton() *widget.Button {
	submitButton := widget.NewButton("Start Search", func() {})
	return submitButton
}

func GetCalendar(onSelected func(time.Time)) *xwidget.Calendar {
	startingDate := time.Now()
	return xwidget.NewCalendar(startingDate, onSelected)
}
