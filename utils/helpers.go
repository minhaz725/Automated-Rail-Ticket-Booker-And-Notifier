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
