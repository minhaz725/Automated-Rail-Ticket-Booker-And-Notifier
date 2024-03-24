package constants

const (
	BASE_URL              = "https://eticket.railway.gov.bd/booking/train/search/?"
	FROM_KEY              = "fromcity="
	TO_KEY                = "&tocity="
	DATE_KEY              = "&doj="
	CLASS_KEY             = "&class="
	DEBUG_CHROME_URL      = "http://localhost:9222"
	SENDER_EMAIL_ADDRESS  = "minhaztimu7250@gmail.com"
	SENDER_EMAIL_NAME     = "Automated Rail Ticket System by Minhaz"
	OWNER_EMAIL_ADDRESS   = "minhaz725@gmail.com"
	SENDER_EMAIL_PASSWORD = "yjia widg uwor uqfo"
	SEARCH_DELAY_IN_SEC   = 2
	INTRO_MSG             = "" +
		"\t\t\t\t\t\t\t\t**** INSTRUCTIONS **** \n" +
		"\t\t\t\t\t -Press Ctrl + R to open Run.\n" +
		"\t\t\t\t\t -Paste chrome.exe --remote-debugging-port=9222 and hit enter.Chrome will open.\n" +
		"\t\t\t\t\t -Login to ticket site. (https://eticket.railway.gov.bd/)\n" +
		"\t\t\t\t\t -Fill below form and hit search button to start searching.\n" +
		"\t\t\t\t\t -Please note that there's no input validation currently, so check spellings carefully.\n" +
		"\t\t\t\t\t -Choose only one train, if you want options then run the app again and choose another train.\n" +
		"\t\t\t\t\t -Cross out the 'Go to booking page' checkbox if you want to review seat before book."
	OUTRO_SUCCESS_MSG = "" +
		"Operation completed successfully, an email has been sent to you.\n" +
		"Go to the opened tab and finish your purchase.\n" +
		"Application will automatically close in 10 seconds.\n" +
		"Thanks for using the app, have a nice journey!"
	OUTRO_FAILURE_MSG = "Operation Failed. Please try again!"
)
