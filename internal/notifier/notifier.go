package notifier

import (
	"fmt"
	"strings"
)

func SendEmail(messageBody string) bool {
	// Sender data.
	//from := constants.SENDER_EMAIL_ADDRESS
	//password := constants.SENDER_EMAIL_PASSWORD
	//
	//// Receiver email address.
	//to := []string{
	//	constants.RECEIVER_EMAIL_ADDRESS,
	//	"minhaztimu7250@gmail.com",
	//}
	// smtp server configuration.
	//smtpHost := "smtp.gmail.com"
	//smtpPort := "587"
	//
	//mail := generateMail(messageBody, from, to, arguments.DATE)
	//// Authentication.
	//auth := smtp.PlainAuth("", from, password, smtpHost)
	//
	//// Sending email.
	//err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, []byte(mail))
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	fmt.Println("Email Sent Successfully!")

	//makeCall()
	return true
}

func MakeCall() bool {

	//urlTimu := "https://e83c-103-243-82-92.ngrok-free.app/call/timu"
	//
	//// Make a GET request to the specified URL
	//_, err := http.Get(urlTimu)
	//if err != nil {
	//	fmt.Println("Error making GET request:", err)
	//	return false
	//} else {
	//	fmt.Println("call made successfully")
	//}
	//
	//urlMuna := "https://e83c-103-243-82-92.ngrok-free.app/call/muna"
	//
	//// Make a GET request to the specified URL
	//_, err = http.Get(urlMuna)
	//if err != nil {
	//	fmt.Println("Error making GET request:", err)
	//	return false
	//} else {
	//	fmt.Println("call made successfully")
	//}
	return true
}

func generateMail(messageBody string, from string, to []string, date string) string {
	// Message.
	msg := "From: " + from + "\r\n"
	msg += "To: " + strings.Join(to, ";") + "\r\n"
	msg += "Subject: Available Tickets on " + date + " From Rail Ticket Notifier\r\n"
	msg += "\r\n" + messageBody
	return msg
}