package main

import (
	"Rail-Ticket-Notifier/internal/ui"
	"fyne.io/fyne/v2/container"
)

// sample arg: go run . -from "Khulna" -to "Dhaka" -date "26-Mar-2024" -seatCount 4 -seatTypes "SNIGDHA,S_CHAIR" -trains "TURNA,SUNDARBAN,BAZAR"
// fyne package -os windows -icon C:\Users\minha\go\src\Rail-Ticket-Notifier\static\logo.png -src cmd/main
// start chrome.exe --remote-debugging-port=9222
func main() {

	elementsOfUI := ui.InitializeUIAndForm()

	form := ui.CreateForm(elementsOfUI)

	elementsOfUI.Window.SetContent(container.NewVBox(form))
	elementsOfUI.Window.ShowAndRun()
}
