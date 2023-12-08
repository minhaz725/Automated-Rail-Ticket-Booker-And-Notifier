package main

import (
	"Rail-Ticket-Notifier/utils/constants"
	"fmt"
	"net/smtp"
	"strings"
)

func sendEmail(messageBody string, date string) {
	// Sender data.
	from := constants.SENDER_EMAIL_ADDRESS
	password := constants.SENDER_EMAIL_PASSWORD

	// Receiver email address.
	to := []string{
		constants.RECEIVER_EMAIL_ADDRESS,
	}
	// smtp server configuration.
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	mail := generateMail(messageBody, from, to, date)
	// Authentication.
	auth := smtp.PlainAuth("", from, password, smtpHost)

	// Sending email.
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, []byte(mail))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Email Sent Successfully!")
}

func generateMail(messageBody string, from string, to []string, date string) string {
	// Message.
	msg := "From: " + from + "\r\n"
	msg += "To: " + strings.Join(to, ";") + "\r\n"
	msg += "Subject: Available Tickets on " + date + " From Rail Ticket Notifier\r\n"
	msg += "\r\n" + messageBody
	return msg
}
