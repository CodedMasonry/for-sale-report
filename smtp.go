package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// createTLSConfig creates a TLS configuration using the certificate from config
func createTLSConfig() (*tls.Config, error) {
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM([]byte(AppConfig.SMTP.Cert)) {
		return nil, fmt.Errorf("failed to parse certificate")
	}

	return &tls.Config{
		ServerName: AppConfig.SMTP.Host,
		RootCAs:    certPool,
	}, nil
}

// createAuth creates SMTP authentication
func createAuth() smtp.Auth {
	return smtp.PlainAuth("", AppConfig.SMTP.User, AppConfig.SMTP.Pass, AppConfig.SMTP.Host)
}

// smtpAddr returns the formatted SMTP server address
func smtpAddr() string {
	return fmt.Sprintf("%s:%s", AppConfig.SMTP.Host, AppConfig.SMTP.Port)
}

// checkServerReady verifies the server is listening and ready
func checkServerReady(client *smtp.Client) error {
	// Just check if server supports AUTH (indicates server is ready)
	if ok, _ := client.Extension("AUTH"); ok {
		return nil
	}
	return fmt.Errorf("SMTP server does not support authentication")
}

// sendMessageData handles the common message sending logic
func sendMessageData(client *smtp.Client, message []byte) error {
	// Set sender
	if err := client.Mail(AppConfig.SMTP.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err := client.Rcpt(AppConfig.SMTP.To); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send message data
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer writer.Close()

	if _, err := writer.Write(message); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// connectWithSTARTTLS establishes SMTP connection with STARTTLS
func connectWithSTARTTLS() (*smtp.Client, error) {
	client, err := smtp.Dial(smtpAddr())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	// Check if STARTTLS is supported
	if ok, _ := client.Extension("STARTTLS"); !ok {
		client.Close()
		return nil, fmt.Errorf("SMTP server does not support STARTTLS")
	}

	tlsConfig, err := createTLSConfig()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	if err := client.StartTLS(tlsConfig); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to start TLS: %w", err)
	}

	return client, nil
}

// VerifySMTPAuth checks if the SMTP server is listening and ready
func VerifySMTPAuth(host, port, username, password string) error {
	client, err := connectWithSTARTTLS()
	if err != nil {
		return err
	}
	defer client.Close()

	if err := checkServerReady(client); err != nil {
		return err
	}

	fmt.Printf("✓ SMTP server is ready at %s:%s\n", host, port)
	return nil
}

var emailTemplate = `<html>
<body>
%s
<i>For Sale Report</i>
</body>
</html>`

var emailListItem = `<p>%v) <span style="color: red;">%s</span> - <b>%s</b></p>
<ul>
<li>%s</li>
<li>%s</li>
</ul>
`

func GenerateEmailReportBody(people []Person) (subject string, body string) {
	currentTime := time.Now().Format(time.Stamp)
	// Handle no expired people
	if len(people) == 0 {
		subject = fmt.Sprintf("No Expired Leads - %s", currentTime)
		body = fmt.Sprintf(emailTemplate, "<p>No expired leads as of "+currentTime+"</p>")
		return
	}
	listItems := make([]string, len(people))
	for index, person := range people {
		item := fmt.Sprintf(emailListItem, index, person.Name, person.Addresses[0].ToString())
		listItems = append(listItems, item)
	}
	subject = fmt.Sprintf("%v Expired Leads - %s", len(people), currentTime)
	body = fmt.Sprintf(emailTemplate, strings.Join(listItems, ""))
	return
}

// formatMessage creates the email message with headers
func formatMessage(subject, body string) []byte {
	message := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"Content-Type: text/html; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n",
		AppConfig.SMTP.From,
		AppConfig.SMTP.To,
		subject,
		body,
	)
	return []byte(message)
}

func SendEmailReport(subject, body string) error {
	if AppConfig == nil {
		return fmt.Errorf("configuration not initialized")
	}

	client, err := connectWithSTARTTLS()
	if err != nil {
		return fmt.Errorf("failed to establish STARTTLS connection: %w", err)
	}
	defer client.Close()

	if err := client.Auth(createAuth()); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	message := formatMessage(subject, body)
	if err := sendMessageData(client, message); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Printf("Email sent successfully via STARTTLS from %s to %s\n",
		AppConfig.SMTP.From, AppConfig.SMTP.To)
	return nil
}
