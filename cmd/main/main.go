package main

import (
	"Rail-Ticket-Notifier/internal/ui"
	"fyne.io/fyne/v2/container"
)

func main() {
	elementsOfUI := ui.InitializeUIAndForm()
	form := ui.CreateForm(elementsOfUI)
	elementsOfUI.Window.SetContent(container.NewVBox(form))
	elementsOfUI.Window.ShowAndRun()
}
