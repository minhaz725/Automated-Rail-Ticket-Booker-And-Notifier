package notifier

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/utils/constants"
	"fmt"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
)

func SendEmail(messageBody string) bool {
	//Sender data.

	// Receiver email address.
	to := []string{
		arguments.RECEIVER_EMAIL_ADDRESS,
		constants.OWNER_EMAIL_ADDRESS,
	}
	//smtp server configuration.
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	mail := generateMail(messageBody, to)
	// Authentication.
	auth := smtp.PlainAuth("", constants.SENDER_EMAIL_ADDRESS, constants.SENDER_EMAIL_PASSWORD, smtpHost)

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, constants.SENDER_EMAIL_ADDRESS, to, []byte(mail))
	if err != nil {
		fmt.Println(err)
		return false
	}
	fmt.Println("Email Sent Successfully!")
	return true
}

func MakeCall(number string) bool {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Calls.json", constants.TWILIO_ACCOUNT_SID)

	twiml := `<Response>
		<Say voice="alice">
			Alert! You have received a new ticket.
			This is a high priority notification.
			Please check your dashboard immediately.
		</Say>
		<Pause length="2"/>
		<Say>Thank you.</Say>
	</Response>`

	data := url.Values{}
	data.Set("From", constants.TWILIO_FROM_NUMBER)
	if number == "" {
		data.Set("To", constants.TWILIO_TO_NUMBER)
	} else {
		data.Set("To", number)
	}

	data.Set("Twiml", twiml)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return false
	}

	req.SetBasicAuth(constants.TWILIO_ACCOUNT_SID, constants.TWILIO_AUTH_TOKEN)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making call:", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("Call initiated successfully!")
		return true
	}

	fmt.Printf("Call failed with status: %d\n", resp.StatusCode)
	return false
}

func generateMail(messageBody string, to []string) string {
	// Message.
	msg := "From: " + constants.SENDER_EMAIL_NAME + " <" + arguments.FROM + ">\r\n"
	msg += "To: " + strings.Join(to, ";") + "\r\n"
	msg += "Subject: Available Tickets on " + arguments.DATE + "\r\n"
	msg += "\r\n" + messageBody
	return msg
}
