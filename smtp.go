package main

import (
	"fmt"
	"net/smtp"
)

// VerifySMTPAuth checks if the SMTP server is running and accepts authentication
func VerifySMTPAuth(host, port, username, password string) error {
	// Create SMTP client
	client, err := smtp.Dial(fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Check if server supports AUTH
	if ok, auths := client.Extension("AUTH"); ok {
		fmt.Printf("✓ SMTP server supports authentication: %s\n", auths)
	} else {
		return fmt.Errorf("SMTP server does not support authentication")
	}

	// Try to authenticate
	auth := smtp.PlainAuth("", username, password, host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Printf("✓ Successfully authenticated with SMTP server\n")
	return nil
}

func SendEmailReport(subject, body string) error {
	// Validate that we have SMTP config
	if AppConfig == nil {
		return fmt.Errorf("configuration not initialized")
	}

	// Compose the email message
	message := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"\r\n"+
			"%s\r\n",
		AppConfig.SMTP.From,
		AppConfig.SMTP.To,
		subject,
		body,
	)

	// Set up SMTP authentication
	auth := smtp.PlainAuth(
		"",
		AppConfig.SMTP.User,
		AppConfig.SMTP.Pass,
		AppConfig.SMTP.Host,
	)

	// Send the email
	smtpAddr := fmt.Sprintf("%s:%s", AppConfig.SMTP.Host, AppConfig.SMTP.Port)
	err := smtp.SendMail(
		smtpAddr,
		auth,
		AppConfig.SMTP.From,
		[]string{AppConfig.SMTP.To},
		[]byte(message),
	)

	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	fmt.Printf("Email sent successfully from %s to %s\n",
		AppConfig.SMTP.From,
		AppConfig.SMTP.To)
	return nil
}
