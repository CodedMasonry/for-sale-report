package main

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// VerifySMTPAuth checks if the SMTP server can be accessed and authenticated
func VerifySMTPAuth(host, port, username, password string) error {
	addr := fmt.Sprintf("%s:%s", host, port)

	// Connect to the SMTP server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer client.Close()

	// Upgrade connection to TLS
	if err = client.StartTLS(&tls.Config{ServerName: host}); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Check if server supports AUTH extension
	if ok, auths := client.Extension("AUTH"); ok {
		fmt.Printf("✓ SMTP server supports authentication: %s\n", auths)
	} else {
		return fmt.Errorf("SMTP server does not support authentication")
	}

	// Authenticate using PlainAuth
	auth := smtp.PlainAuth("", username, password, host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Printf("✓ Successfully authenticated with SMTP server\n")
	return nil
}

// Build HTML body from Person slice
func buildHTMLBody(people []Person) string {
	var sb strings.Builder
	sb.WriteString(`<html><body>`)
	sb.WriteString(`<h2>Listings Report</h2>`)
	sb.WriteString(`<p>The following individuals have been marked as expired leads in FUB.</p>`)
	sb.WriteString(`<table border="1" cellpadding="5" cellspacing="0" style="border-collapse: collapse; width: 100%;">`)
	sb.WriteString(`<tr style="background-color: #dddddd;"><th>#</th><th>Name</th><th>ID</th><th>Addresses</th></tr>`)

	for i, p := range people {
		// Alternate row background color
		rowColor := "#ffffff"
		if i%2 == 1 {
			rowColor = "#f2f2f2"
		}

		addresses := []string{}
		for _, a := range p.Addresses {
			addresses = append(addresses, fmt.Sprintf("%s, %s, %s %s", a.Street, a.City, a.State, a.Code))
		}

		sb.WriteString(fmt.Sprintf(
			`<tr style="background-color: %s;">
				<td>%d</td>
				<td><b>%s</b></td>
				<td>%d</td>
				<td>%s</td>
			</tr>`,
			rowColor,
			i+1,
			p.Name,
			p.ID,
			strings.Join(addresses, "<br>"),
		))
	}

	sb.WriteString(`</table></body></html>`)
	return sb.String()
}

// SendEmailReport sends an HTML email to multiple recipients
func SendEmailReport(subject string, people []Person) error {
	if AppConfig.SMTP.User == "" || len(AppConfig.SMTP.To) == 0 {
		return fmt.Errorf("SMTP config not initialized properly")
	}

	host := AppConfig.SMTP.Host
	port := AppConfig.SMTP.Port
	addr := fmt.Sprintf("%s:%s", host, port)

	body := buildHTMLBody(people)

	// Construct MIME email with HTML
	msg := fmt.Sprintf("From: %s\r\n", AppConfig.SMTP.From)
	msg += fmt.Sprintf("To: %s\r\n", strings.Join(AppConfig.SMTP.To, ","))
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "MIME-Version: 1.0\r\n"
	msg += "Content-Type: text/html; charset=\"UTF-8\"\r\n"
	msg += "\r\n" + body

	auth := smtp.PlainAuth("", AppConfig.SMTP.User, AppConfig.SMTP.Pass, host)

	// Connect
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Upgrade to TLS
	if err = client.StartTLS(&tls.Config{ServerName: host}); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Send to multiple recipients
	if err = client.Mail(AppConfig.SMTP.From); err != nil {
		return err
	}
	for _, recipient := range AppConfig.SMTP.To {
		if err = client.Rcpt(recipient); err != nil {
			return err
		}
	}

	wc, err := client.Data()
	if err != nil {
		return err
	}
	_, err = wc.Write([]byte(msg))
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}

	fmt.Printf("✓ HTML email sent successfully to: %s\n", strings.Join(AppConfig.SMTP.To, ", "))
	return nil
}
