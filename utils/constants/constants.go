package constants

const (
	BASE_URL               = "https://eticket.railway.gov.bd/booking/train/search/?"
	FROM                   = "fromcity=Dhaka"
	TO                     = "&tocity=Cox%27s%20Bazar"
	DATE                   = "&doj=10-Feb-2024"
	CLASS                  = "&class=S_CHAIR"
	SEAT_COUNT             = 2
	SENDER_EMAIL_ADDRESS   = "minhaz725@gmail.com"
	SENDER_EMAIL_PASSWORD  = "akey whwp pnul eskw"
	RECEIVER_EMAIL_ADDRESS = "atiabintiaziz@gmail.com"
	SEARCH_DELAY_IN_SEC    = 2
	SPECIFIC_TRAIN         = "PARJOTAK EXPRESS (816)"
	SEAT_TYPE              = "SNIGDHA"
	COACH_NUMB             = "SCHA"
	SEAT_ONE_NUMB          = "-51"
	SEAT_TWO_NUMB          = "-12"
)

var CoachMap = map[string]string{
	"KA":   "0",
	"TA":   "1",
	"THA":  "2",
	"DA":   "3",
	"SCHA": "4",
	"DANT": "5",
	"TO":   "6",
	"THO":  "7",
	"DOA":  "8",
}
