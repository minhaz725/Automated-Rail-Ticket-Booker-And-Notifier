package models

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type ElementsOfUI struct {
	Window         fyne.Window
	IntroLabel     *widget.Label
	FromEntry      *widget.Entry
	ToEntry        *widget.Entry
	DateEntry      *widget.Entry
	SeatCountEntry *widget.Entry
	SeatTypesEntry *widget.Entry
	TrainsEntry    *widget.Entry
	EmailEntry     *widget.Entry
	PhoneEntry     *widget.Entry
	GoToBookEntry  *widget.Check
}
