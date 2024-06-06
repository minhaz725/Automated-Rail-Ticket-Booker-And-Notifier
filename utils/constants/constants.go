package constants

const (
	BASE_URL              = "https://eticket.railway.gov.bd/booking/train/search/?"
	FROM_KEY              = "fromcity="
	TO_KEY                = "&tocity="
	DATE_KEY              = "&doj="
	CLASS_KEY             = "&class="
	DEBUG_CHROME_URL      = "http://localhost:9222"
	DEBUG_MODE_CHECK_URL  = "http://127.0.0.1:9222/json/version"
	SENDER_EMAIL_ADDRESS  = "minhaztimu7250@gmail.com"
	SENDER_EMAIL_NAME     = "Automated Rail Ticket System by Minhaz"
	OWNER_EMAIL_ADDRESS   = "minhaz725@gmail.com"
	SENDER_EMAIL_PASSWORD = "yjia widg uwor uqfo"
	SEARCH_DELAY_IN_SEC   = 2
	WINDOWS_CHROME_PATH   = "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"
	MAC_CHROME_PATH       = "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	LINUX_CHROME_PATH     = "/usr/bin/google-chrome"
	INTRO_MSG             = "" +
		"\t\t\t\t\t **** PLEASE READ IF YOU'RE USING FIRST TIME **** \n\n" +
		//" -Close Chrome if already opened.\n" +
		//" -Press Windows key + R to open Run.\n" +
		//" -Paste this text:  chrome.exe --remote-debugging-port=9222 and hit enter. It will open Chrome.\n" +
		//" -At that opened chrome, login to ticket site. (https://eticket.railway.gov.bd/), if you're already logged in, skip.\n" +
		//" -Fill in the program's form data and hit search button to start searching. Don't close chrome.\n" +
		//" -You can minimize the program and continue your work. Don't close your computer.\n" +
		//" -If the program works then it will automatically open a chrome tab and you have to finish your purchase there.\n\n" +
		//"\t\t\t\t\t****** NOTES ****** \n\n" +
		" --Google Chrome must be installed in your Operating System's default install location.\n\n" +
		" --You need to log in to the ticket site(https://eticket.railway.gov.bd/), if you're already logged in, skip.\n\n" +
		" --There's no input validation currently, so check spellings carefully.\n\n" +
		" --Know the Seat Types: S_CHAIR-> Non AC Chair, SNIGDHA-> AC Chair, F_BERTH-> Non AC Bed (Cabin)\n" +
		"   AC_B-> AC Bed (Cabin), F_SEAT-> Non AC Chair (Cabin), AC_S-> AC Chair (Cabin), SHOVAN-> 2nd class Bench\n\n" +
		" --Write only one train name, if you want to search multiple trains together then run the app again and write another.\n\n" +
		" --The seat face option has Towards Dhaka and From Dhaka options. By choosing Towards Dhaka, app tries to find\n" +
		"  seats from seat 1 and choosing the opposite option, app tries to find seats from last seat. Usually if train goes to\n" +
		"  dhaka seat 1 to Half (ex. 1-30) is forward. And if goes out from dhaka seat last to Half (ex. 60-30)  is forward\n" +
		"  For other destinations, you need to know the train face info and select options accordingly.\n\n" +
		" --Cross out the 'Go to booking page' checkbox if you want to review seats before book.\n\n" +
		" --If you plan to purchase at ticket release day (10 days prior) at Eid time: \n" +
		"\t -Start app at 2.00PM you want to travel Purbanchal.\n" +
		"\t -Start app at 8.00AM you want to travel Poshchimanchal.\n" +
		"\t -Start app at anytime if it's between 9 to 1 day before journey.\n" +
		"\t -If you search once, your previous search will always be saved.\n" +
		"\t -If you see zero tickets, don't worry. Just keep it running (internet must be on), app will notify you.\n" +
		"\t -OTP might not come timely, seat might not be clickable. Here,just run the program again and again.\n" +
		"\t -Program will always automatically detect empty seats and book them, so don't lose hope.\n"
	OUTRO_SUCCESS_MSG = "" +
		"Operation completed successfully, an email has been sent to you.\n" +
		"Go to the opened tab and finish your purchase.\n" +
		"Application will automatically close in 10 seconds.\n" +
		"Thanks for using the app, have a nice journey!"
	OUTRO_FAILURE_MSG        = "Operation Failed. Please try again!"
	CHROME_SETUP_MSG         = "Setting up Chrome, Chrome will close and reopen automatically."
	CHROME_SETUP_SUCCESS_MSG = "Chrome Setup Successful! Fill the form and hit search button!\n " +
		"\t\tDon't close chrome till the application ends."
	CHROME_SETUP_FAILURE_MSG = "Chrome Setup Failed, check if chrome is installed on you os's default location (Closing in 5 seconds)"
)
