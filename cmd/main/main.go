package main

import (
	"Rail-Ticket-Notifier/internal/ui"
	"fyne.io/fyne/v2/container"
)

// sample arg: go run . -from "Khulna" -to "Dhaka" -date "26-Mar-2024" -seatCount 4 -seatTypes "SNIGDHA,S_CHAIR" -trains "TURNA,SUNDARBAN,BAZAR"
func main() {
	window, fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry := ui.InitializeUIAndForm()

	form := ui.CreateForm(fromEntry, toEntry, dateEntry, seatCountEntry, seatTypesEntry, trainsEntry)

	window.SetContent(container.NewVBox(form))
	window.ShowAndRun()
}
