package constants

const (
	BASE_URL               = "https://eticket.railway.gov.bd/booking/train/search/?"
	FROM_KEY               = "fromcity="
	TO_KEY                 = "&tocity="
	DATE_KEY               = "&doj="
	CLASS_KEY              = "&class="
	DEBUG_CHROME_URL       = "http://localhost:9222"
	SENDER_EMAIL_ADDRESS   = "minhaz725@gmail.com"
	SENDER_EMAIL_PASSWORD  = "akey whwp pnul eskw"
	RECEIVER_EMAIL_ADDRESS = "atiabintiaziz@gmail.com"
	SEARCH_DELAY_IN_SEC    = 2
	INTRO_MSG              = "	Press Ctrl + R to open Run.\n" +
		" 	Paste chrome.exe --remote-debugging-port=9222 and hit enter.\n" +
		" 	Chrome will open.\n        " +
		"   Login to ticket site. (https://eticket.railway.gov.bd/)\n        " +
		"   Fill below form and hit search button to start searching."
)
