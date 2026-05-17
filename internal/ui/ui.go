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

func Run() {
	os.Setenv("FYNE_SCALE", "0.8")
	a := app.NewWithID("Rail-Ticket-Notifier")
	showInstancePicker(a)
}

func showInstancePicker(a fyne.App) {
	pickerWindow := a.NewWindow("Select Instance")
	pickerWindow.Resize(fyne.NewSize(350, 200))
	pickerWindow.SetFixedSize(true)

	lastIndex := strconv.Itoa(utils.LoadLastInstanceIndex())

	instanceOptions := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"}
	instanceSelect := widget.NewSelect(instanceOptions, nil)
	instanceSelect.SetSelected(lastIndex)

	continueBtn := widget.NewButton("Continue", func() {
		idx, err := strconv.Atoi(instanceSelect.Selected)
		if err != nil || idx < 1 || idx > 10 {
			idx = 1
		}
		arguments.INSTANCE_INDEX = idx
		utils.SaveLastInstanceIndex(idx)

		elementsOfUI := initializeUIAndForm(a, idx)
		form := CreateForm(elementsOfUI)
		elementsOfUI.Window.SetContent(container.NewVBox(form))
		elementsOfUI.Window.Show()

		pickerWindow.Close()
	})

	pickerWindow.SetContent(container.NewVBox(
		widget.NewLabel("Select Chrome Instance:"),
		instanceSelect,
		continueBtn,
	))

	pickerWindow.ShowAndRun()
}

func initializeUIAndForm(a fyne.App, index int) models.ElementsOfUI {
	prefs := utils.LoadInstancePrefs(index)

	window := a.NewWindow("Automated Rail Ticket Booker & Notifier")
	window.Resize(fyne.NewSize(800, 600))

	// welcome popup
	label := widget.NewLabel(constants.INTRO_MSG)
	label.Alignment = fyne.TextAlignLeading // Set text alignment to left
	customDialog := dialog.NewCustom("Welcome", "I've Read, Continue", container.NewVBox(label), window)
	customDialog.SetOnClosed(setChromeAfterIntroContinuePressed(window))
	// POPUP
	//customDialog.Show()

	// Create form fields with default values
	fromEntry := widget.NewEntry()
	fromEntry.SetText(utils.GetPrefWithFallback(prefs, "fromEntry", arguments.FROM))

	toEntry := widget.NewEntry()
	toEntry.SetText(utils.GetPrefWithFallback(prefs, "toEntry", arguments.TO))

	dateEntry := widget.NewEntry()
	dateEntry.SetText(utils.GetPrefWithFallback(prefs, "dateEntry", arguments.DATE))

	seatCountEntry := widget.NewEntry()
	seatCountEntry.SetText(utils.GetPrefWithFallback(prefs, "seatCountEntry", strconv.Itoa(int(arguments.SEAT_COUNT))))

	seatTypesEntry := widget.NewEntry()
	seatTypesEntry.SetText(utils.GetPrefWithFallback(prefs, "seatTypesEntry", strings.Join(arguments.SEAT_TYPE_ARRAY, ",")))

	trainsEntry := widget.NewEntry()
	trainsEntry.SetText(utils.GetPrefWithFallback(prefs, "trainsEntry", strings.Join(arguments.SPECIFIC_TRAIN_ARRAY, ",")))

	emailEntry := widget.NewEntry()
	emailEntry.SetText(utils.GetPrefWithFallback(prefs, "emailEntry", arguments.RECEIVER_EMAIL_ADDRESS))

	phoneEntry := widget.NewEntry()
	phoneEntry.SetText(utils.GetPrefWithFallback(prefs, "phoneEntry", arguments.PHONE_NUMBER))
	phoneEntry.Disable()

	options := []string{"Travelling Towards Dhaka", "Travelling From Dhaka"}

	seatFaceEntry := widget.NewRadioGroup(options, func(value string) {})
	seatFaceEntry.Horizontal = true
	seatFaceEntry.SetSelected(utils.GetPrefWithFallback(prefs, "seatFaceEntry", arguments.SEAT_FACE))

	content := container.NewVBox(fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry, emailEntry, phoneEntry)

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
		SeatFaceEntry:  seatFaceEntry,
		InstanceIndex:  index,
	}

	return uiElements
}

func CreateForm(uiElements models.ElementsOfUI) *fyne.Container {

	calendar := GetCalendar(func(t time.Time) {
		uiElements.DateEntry.SetText(t.Format("02-Jan-2006"))
	})

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "From", Widget: uiElements.FromEntry},
			{Text: "Destination", Widget: uiElements.ToEntry},
			{Text: "Date Of Journey (Choose From Calender)", Widget: uiElements.DateEntry},
			{Text: "(Only from current date to next 10 days)", Widget: calendar},
			{Text: "Seat Count (1 to Max 4)", Widget: uiElements.SeatCountEntry},
			{Text: "Seat Types (Prioritize Serially.Separate by comma(,) no space)", Widget: uiElements.SeatTypesEntry},
			{Text: "Trains (Choose only One. All Capitals)", Widget: uiElements.TrainsEntry},
			{Text: "Email address (To receive mail after done)", Widget: uiElements.EmailEntry},
			{Text: "Phone Number (To Receive call. Currently unavailable)", Widget: uiElements.PhoneEntry},
			{Text: "Seat Facing (Prioritize Seats towards train's direction)", Widget: uiElements.SeatFaceEntry},
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
