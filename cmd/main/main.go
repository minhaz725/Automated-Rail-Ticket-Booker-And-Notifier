package main

import (
	"Rail-Ticket-Notifier/internal/ui"
	"fyne.io/fyne/v2/container"
)

// sample arg: go run . -from "Khulna" -to "Dhaka" -date "26-Mar-2024" -seatCount 4 -seatTypes "SNIGDHA,S_CHAIR" -trains "TURNA,SUNDARBAN,BAZAR"
func main() {

	elementsOfUI := ui.InitializeUIAndForm()

	form := ui.CreateForm(elementsOfUI)

	elementsOfUI.Window.SetContent(container.NewVBox(elementsOfUI.IntroLabel, form))
	elementsOfUI.Window.ShowAndRun()
}
