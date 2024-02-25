package constants

const (
	BASE_URL               = "https://eticket.railway.gov.bd/booking/train/search/?"
	DEBUG_CHROME_URL       = "http://localhost:9222"
	FROM                   = "fromcity=Chattogram"
	TO                     = "&tocity=Dhaka"
	DATE                   = "&doj=26-Feb-2024"
	CLASS                  = "&class=S_CHAIR"
	SEAT_COUNT             = 2
	SENDER_EMAIL_ADDRESS   = "minhaz725@gmail.com"
	SENDER_EMAIL_PASSWORD  = "akey whwp pnul eskw"
	RECEIVER_EMAIL_ADDRESS = "atiabintiaziz@gmail.com"
	SEARCH_DELAY_IN_SEC    = 2
)

var (
	SEAT_TYPE_ARRAY      = []string{"S_CHAIR", "SNIGDHA"}
	SPECIFIC_TRAIN_ARRAY = []string{"TURNA", "PARJOTAK", "BAZAR"}
)
