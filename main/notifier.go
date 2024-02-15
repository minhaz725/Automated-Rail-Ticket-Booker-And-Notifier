package main

import (
	"Rail-Ticket-Notifier/utils/constants"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
)

func sendEmail(messageBody string) {
	// Sender data.
	from := constants.SENDER_EMAIL_ADDRESS
	password := constants.SENDER_EMAIL_PASSWORD

	// Receiver email address.
	to := []string{
		constants.RECEIVER_EMAIL_ADDRESS,
		"minhaztimu7250@gmail.com",
	}
	// smtp server configuration.
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	mail := generateMail(messageBody, from, to, constants.DATE)
	// Authentication.
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, []byte(mail))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Email Sent Successfully!")

	makeCall()
}

func makeCall() {
	urlTimu := "https://ece9-103-72-212-129.ngrok-free.app/call/timu"

	// Make a GET request to the specified URL
	_, err := http.Get(urlTimu)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return
	} else {
		fmt.Println("call made successfully")
	}

	urlMuna := "https://ece9-103-72-212-129.ngrok-free.app/call/muna"

	// Make a GET request to the specified URL
	_, err = http.Get(urlMuna)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return
	} else {
		fmt.Println("call made successfully")
	}
}

func generateMail(messageBody string, from string, to []string, date string) string {
	// Message.
	msg := "From: " + from + "\r\n"
	msg += "To: " + strings.Join(to, ";") + "\r\n"
	msg += "Subject: Available Tickets on " + date + " From Rail Ticket Notifier\r\n"
	msg += "\r\n" + messageBody
	return msg
}
