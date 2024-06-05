package utils

func FindFirstMatch(availableSeatClassArray, SeatTypeArray []string) string {
	for _, seatType := range SeatTypeArray {
		for _, availableSeatClass := range availableSeatClassArray {
			if seatType == availableSeatClass {
				return seatType
			}
		}
	}
	return ""
}

// fyne package -os windows -icon C:\Users\minha\go\src\Rail-Ticket-Notifier\static\logo.png -src cmd/main -name YourDesiredOutputName
