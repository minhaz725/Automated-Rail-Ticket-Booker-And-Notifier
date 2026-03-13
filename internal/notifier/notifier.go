package notifier

import (
	"Rail-Ticket-Notifier/internal/arguments"
	"Rail-Ticket-Notifier/utils/constants"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

var (
	resolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}
	dialer = &net.Dialer{
		Timeout:  10 * time.Second,
		Resolver: resolver,
	}
	httpClient = &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
		},
	}
)

func SendEmail(messageBody string) bool {
	to := []string{
		arguments.RECEIVER_EMAIL_ADDRESS,
		constants.OWNER_EMAIL_ADDRESS,
	}

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	mail, sanitizedTo, sanitizedFrom := generateMail(messageBody, to)
	auth := smtp.PlainAuth("", constants.SENDER_EMAIL_ADDRESS, constants.SENDER_EMAIL_PASSWORD, smtpHost)

	err := sendMailWithCustomDNS(smtpHost, smtpPort, auth, sanitizedFrom, sanitizedTo, []byte(mail))
	if err != nil {
		fmt.Println(err)
		return false
	}
	fmt.Println("Email Sent Successfully!")
	return true
}

func sendMailWithCustomDNS(host, port string, auth smtp.Auth, from string, to []string, msg []byte) error {
	addr := host + ":" + port
	conn, err := dialer.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial error: %w", err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("smtp client error: %w", err)
	}
	defer c.Close()

	if err = c.StartTLS(&tls.Config{ServerName: host}); err != nil {
		return fmt.Errorf("starttls error: %w", err)
	}
	if err = c.Auth(auth); err != nil {
		return fmt.Errorf("auth error: %w", err)
	}
	if err = c.Mail(from); err != nil {
		return fmt.Errorf("mail from error: %w", err)
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return fmt.Errorf("rcpt error: %w", err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data error: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	if err = w.Close(); err != nil {
		return fmt.Errorf("close data error: %w", err)
	}
	return c.Quit()
}

func MakeCall() bool {
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
	data.Set("To", constants.TWILIO_TO_NUMBER)
	data.Set("From", constants.TWILIO_FROM_NUMBER)
	data.Set("Twiml", twiml)

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return false
	}

	req.SetBasicAuth(constants.TWILIO_ACCOUNT_SID, constants.TWILIO_AUTH_TOKEN)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
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

func generateMail(messageBody string, to []string) (string, []string, string) {
	// Sanitize function - remove CR/LF and trim whitespace
	sanitize := func(s string) string {
		s = strings.ReplaceAll(s, "\r", "")
		s = strings.ReplaceAll(s, "\n", "")
		return strings.TrimSpace(s)
	}

	// Sanitize recipients and filter empty ones
	var sanitizedTo []string
	for _, email := range to {
		if clean := sanitize(email); clean != "" {
			sanitizedTo = append(sanitizedTo, clean)
		}
	}

	// Sanitize sender
	sanitizedFrom := sanitize(constants.SENDER_EMAIL_ADDRESS)

	// Sanitize the message body - convert to CRLF for SMTP
	sanitizedBody := strings.ReplaceAll(messageBody, "\r\n", "\n")
	sanitizedBody = strings.ReplaceAll(sanitizedBody, "\n", "\r\n")

	// Build headers with sanitized values
	msg := "From: " + sanitize(constants.SENDER_EMAIL_NAME) + " <" + sanitizedFrom + ">\r\n"
	msg += "To: " + strings.Join(sanitizedTo, ", ") + "\r\n"
	msg += "Subject: Available Tickets on " + sanitize(arguments.DATE) + "\r\n"
	msg += "MIME-Version: 1.0\r\n"
	msg += "Content-Type: text/plain; charset=\"utf-8\"\r\n"
	msg += "\r\n" + sanitizedBody

	return msg, sanitizedTo, sanitizedFrom
}
