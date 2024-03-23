package arguments

import (
	"Rail-Ticket-Notifier/utils/constants"
	"flag"
	"strings"
)

var (
	FROM                 string
	TO                   string
	DATE                 string
	SEAT_COUNT           uint
	SEAT_TYPE_ARRAY      = []string{"SNIGDHA", "S_CHAIR"}
	SPECIFIC_TRAIN_ARRAY = []string{"SONAR"} //{"SONAR", "TURNA", "SUBORNO"}
)

func init() {
	flag.StringVar(&FROM, "from", "Chattogram", "From city")
	flag.StringVar(&TO, "to", "Dhaka", "To city")
	flag.StringVar(&DATE, "date", "28-Mar-2024", "Date of travel")
	flag.UintVar(&SEAT_COUNT, "seatCount", 2, "Seat count")

	flag.Func("seatTypes", "Seat types", func(s string) error {
		SEAT_TYPE_ARRAY = strings.Split(s, ",")
		return nil
	})

	flag.Func("trains", "Specific trains", func(s string) error {
		SPECIFIC_TRAIN_ARRAY = strings.Split(s, ",")
		return nil
	})

	flag.Parse()
}

func UpdateArguments(from, to, date string, seatCount uint, seatTypes, trains []string) {
	FROM = from
	TO = to
	DATE = date
	SEAT_COUNT = seatCount
	SEAT_TYPE_ARRAY = seatTypes
	SPECIFIC_TRAIN_ARRAY = trains
}

func GenerateURL() string {
	return constants.BASE_URL + constants.FROM_KEY + FROM + constants.TO_KEY + TO + constants.DATE_KEY + DATE + constants.CLASS_KEY + SEAT_TYPE_ARRAY[0]
}
