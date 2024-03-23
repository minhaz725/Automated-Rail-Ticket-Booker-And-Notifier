package arguments

import (
	"Rail-Ticket-Notifier/utils/constants"
	"flag"
	"strconv"
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

func UpdateArguments(from, to, date string, seatCount string, seatTypes, trains []string) {
	value, _ := strconv.ParseUint(seatCount, 10, 32)
	FROM = from
	TO = to
	DATE = date
	SEAT_COUNT = uint(value)
	SEAT_TYPE_ARRAY = seatTypes
	SPECIFIC_TRAIN_ARRAY = trains
}

func GetURL() string {
	return constants.BASE_URL + "fromcity=" + FROM + "&tocity=" + TO + "&doj=" + DATE + "&class=" + SEAT_TYPE_ARRAY[0]
}
