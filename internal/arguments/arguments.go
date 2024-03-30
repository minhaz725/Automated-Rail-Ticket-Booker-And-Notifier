package arguments

import (
	"Rail-Ticket-Notifier/utils/constants"
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
	GO_TO_BOOK_PAGE        uint
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
	flag.UintVar(&GO_TO_BOOK_PAGE, "purchasePage", 1, "Go to purchase page")

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

func UpdateArguments(from, to, date, email string, goToBookPage bool, seatCount uint, seatTypes, trains []string) {
	if from == "Chapai Nawabganj" {
		FROM = "Chapai%20Nawabganj"
	} else if from == "Cox's Bazar" {
		FROM = "Cox%27s%20Bazar"
	} else {
		FROM = from
	}

	if to == "Chapai Nawabganj" {
		TO = "Chapai%20Nawabganj"
	} else if to == "Cox's Bazar" {
		TO = "Cox%27s%20Bazar"
	} else {
		TO = to
	}
	DATE = date
	RECEIVER_EMAIL_ADDRESS = email
	SEAT_COUNT = seatCount
	SEAT_TYPE_ARRAY = seatTypes
	SPECIFIC_TRAIN_ARRAY = trains
	if goToBookPage {
		GO_TO_BOOK_PAGE = 1
	} else {
		GO_TO_BOOK_PAGE = 0
	}
}

func GenerateURL() string {
	return constants.BASE_URL + constants.FROM_KEY + FROM + constants.TO_KEY + TO + constants.DATE_KEY + DATE + constants.CLASS_KEY + SEAT_TYPE_ARRAY[0]
}
