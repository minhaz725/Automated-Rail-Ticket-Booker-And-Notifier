package ui

import (
	"Rail-Ticket-Notifier/cmd/handlers"
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/internal/models"
	"Rail-Ticket-Notifier/utils"
	"Rail-Ticket-Notifier/utils/constants"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func InitializeUIAndForm() models.ElementsOfUI {
	a := app.NewWithID("Rail-Ticket-Notifier")

	window := a.NewWindow("Automated Rail Ticket Booker & Notifier")
	window.Resize(fyne.NewSize(800, 600))

	// welcome popup
	label := widget.NewLabel(constants.INTRO_MSG)
	label.Alignment = fyne.TextAlignLeading // Set text alignment to left
	customDialog := dialog.NewCustom("Welcome", "I've Read, Continue", container.NewVBox(label), window)
	customDialog.SetOnClosed(setChromeAfterIntroContinuePressed(window))
	customDialog.Show()

	// Create form fields with default values
	fromEntry := widget.NewEntry()
	fromEntry.SetText(a.Preferences().StringWithFallback("fromEntry", arguments.FROM))

	toEntry := widget.NewEntry()
	toEntry.SetText(a.Preferences().StringWithFallback("toEntry", arguments.TO))

	dateEntry := widget.NewEntry()
	dateEntry.SetText(a.Preferences().StringWithFallback("dateEntry", arguments.DATE))

	seatCountEntry := widget.NewEntry()
	seatCountEntry.SetText(a.Preferences().StringWithFallback("seatCountEntry", strconv.Itoa(int(arguments.SEAT_COUNT))))

	seatTypesEntry := widget.NewEntry()
	seatTypesEntry.SetText(a.Preferences().StringWithFallback("seatTypesEntry", strings.Join(arguments.SEAT_TYPE_ARRAY, ",")))

	trainsEntry := widget.NewEntry()
	trainsEntry.SetText(a.Preferences().StringWithFallback("trainsEntry", strings.Join(arguments.SPECIFIC_TRAIN_ARRAY, ",")))

	emailEntry := widget.NewEntry()
	emailEntry.SetText(a.Preferences().StringWithFallback("emailEntry", arguments.RECEIVER_EMAIL_ADDRESS))

	phoneEntry := widget.NewEntry()
	phoneEntry.SetText(a.Preferences().StringWithFallback("phoneEntry", arguments.PHONE_NUMBER))
	phoneEntry.Disable()

	options := []string{"Travelling Towards Dhaka", "Travelling From Dhaka"}

	seatFaceEntry := widget.NewRadioGroup(options, func(value string) {})
	seatFaceEntry.Horizontal = true
	seatFaceEntry.SetSelected(a.Preferences().StringWithFallback("seatFaceEntry", arguments.SEAT_FACE))

	goToBookEntry := widget.NewCheck("Uncheck this to stay at seat selection page to manually adjust seats.", func(value bool) {})
	goToBookEntry.SetChecked(a.Preferences().BoolWithFallback("goToBookEntry", arguments.GO_TO_BOOK_PAGE != 0))

	content := container.NewVBox(fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry, emailEntry, phoneEntry, goToBookEntry)

	scrollContainer := container.NewVScroll(content)

	scrollContainer.SetMinSize(fyne.NewSize(800, 600)) // Set minimum size to window size

	window.SetContent(scrollContainer)

	uiElements := models.ElementsOfUI{
		App:            a,
		Window:         window,
		FromEntry:      fromEntry,
		ToEntry:        toEntry,
		DateEntry:      dateEntry,
		SeatCountEntry: seatCountEntry,
		SeatTypesEntry: seatTypesEntry,
		TrainsEntry:    trainsEntry,
		EmailEntry:     emailEntry,
		PhoneEntry:     phoneEntry,
		SeatFaceEntry:  seatFaceEntry,
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
			{Text: "From (Title Case)", Widget: uiElements.FromEntry},
			{Text: "To (Title Case)", Widget: uiElements.ToEntry},
			{Text: "Date Of Journey (Choose From Calender)", Widget: uiElements.DateEntry},
			{Text: "(Only from current date to next 10 days)", Widget: calendar},
			{Text: "Seat Count (1 to Max 4)", Widget: uiElements.SeatCountEntry},
			{Text: "Seat Types (Prioritize Serially.Separate by comma(,) no space. All Capitals)", Widget: uiElements.SeatTypesEntry},
			{Text: "Trains (Choose only One. All Capitals)", Widget: uiElements.TrainsEntry},
			{Text: "Email address (To receive mail after done)", Widget: uiElements.EmailEntry},
			{Text: "Phone Number (To Receive call. Currently unavailable)", Widget: uiElements.PhoneEntry},
			{Text: "Seat Facing (Prioritize Seats towards train's direction)", Widget: uiElements.SeatFaceEntry},
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

func setChromeAfterIntroContinuePressed(window fyne.Window) func() {
	return func() {
		if !utils.SetupChrome(window) {
			dialog.ShowInformation("Failed", constants.CHROME_SETUP_FAILURE_MSG, window)
			log.Println("Chrome Setup Failed. Maybe Chrome not installed or OS not supported. Exiting Program")
			time.Sleep(5 * time.Second)
			// Terminate the program
			os.Exit(0)
		}
	}
}

func getSubmitButton() *widget.Button {
	submitButton := widget.NewButton("Start Search", func() {})
	return submitButton
}

func GetCalendar(onSelected func(time.Time)) *xwidget.Calendar {
	startingDate := time.Now()
	return xwidget.NewCalendar(startingDate, onSelected)
}
