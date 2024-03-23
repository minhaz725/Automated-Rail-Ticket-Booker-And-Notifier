package main

import (
	"Rail-Ticket-Notifier/internal/ui"
	"fyne.io/fyne/v2/container"
)

// sample arg: go run . -from "Khulna" -to "Dhaka" -date "26-Mar-2024" -seatCount 4 -seatTypes "SNIGDHA,S_CHAIR" -trains "TURNA,SUNDARBAN,BAZAR"
func main() {

	window, introLabel, fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry := ui.InitializeUIAndForm()

	form := ui.CreateForm(fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry, ui.GetSubmitButton())

	window.SetContent(container.NewVBox(introLabel, form))
	window.ShowAndRun()
}
