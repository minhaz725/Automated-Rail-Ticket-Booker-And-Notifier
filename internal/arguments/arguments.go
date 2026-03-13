package arguments

import (
	"flag"
	"strings"
	"time"
)

var (
	FROM                   string
	TO                     string
	DATE                   string
	SEAT_COUNT             uint
	PHONE_NUMBER           string
	SEAT_TYPE_ARRAY        = []string{"SNIGDHA", "F_BERTH", "AC_B", "AC_S", "S_CHAIR", "F_SEAT", "SHOVON"}
	SPECIFIC_TRAIN_ARRAY   = []string{"SUBORNO"} //{"SONAR", "TURNA", "SUBORNO"}
	RECEIVER_EMAIL_ADDRESS string
	SEAT_FACE              string
	GO_TO_BOOK_PAGE        uint
	AUTH_FILE              string
)

func init() {
	now := time.Now()
	twoDaysLater := now.AddDate(0, 0, 10)
	formattedDateAfterTwoDays := twoDaysLater.Format("02-Jan-2006")

	flag.StringVar(&FROM, "from", "Dhaka", "From city")
	flag.StringVar(&TO, "to", "Chattogram", "To city")
	flag.StringVar(&PHONE_NUMBER, "phone", "+8801555555555", "Phone")
	flag.StringVar(&DATE, "date", formattedDateAfterTwoDays, "Date of travel")
	flag.StringVar(&RECEIVER_EMAIL_ADDRESS, "email", "minhaz725@gmail.com", "Email address")
	flag.UintVar(&SEAT_COUNT, "seatCount", 2, "Seat count")
	flag.StringVar(&SEAT_FACE, "seatFace", "Travelling Towards Dhaka", "Seat Face")
	flag.UintVar(&GO_TO_BOOK_PAGE, "purchasePage", 1, "Go to purchase page")
	flag.StringVar(&AUTH_FILE, "auth", "auth.json", "Path to auth.json file")

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


