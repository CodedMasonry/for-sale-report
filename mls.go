package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

type MLS struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type AddrStatus int

const (
	addrError      AddrStatus = iota // 0
	addrClosed                       // 1
	addrComingSoon                   // 2
	addrActive                       // 3
)

type PersonStatus struct {
	id     int
	status AddrStatus
}

func BuildMLS(user string, pass string) (mls *MLS, err error) {
	// Create context
	ctx, cancel := chromedp.NewContext(context.Background())

	// Set a timeout
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)

	// Login with chromedp and return the context to control it
	err = loginAndGetCookies(ctx, user, pass)
	if err != nil {
		cancel()
		return nil, err
	}

	return &MLS{
		ctx,
		cancel,
	}, nil
}

func loginAndGetCookies(ctx context.Context, user string, pass string) error {
	// Get all the necessary cookies so ctx can be used later
	err := chromedp.Run(ctx,
		// Navigate to the login page
		chromedp.Navigate(MLS_LOGIN_URL),

		// Wait for the page to load
		chromedp.WaitVisible(`input[name="username"]`, chromedp.ByQuery),

		// Fill in the username field
		chromedp.SendKeys(`input[name="username"]`, user, chromedp.ByQuery),

		// Fill in the password field
		chromedp.SendKeys(`input[name="password"]`, pass, chromedp.ByQuery),

		// Submit the form (either click submit button or press Enter)
		chromedp.Click(`input[type="submit"]`, chromedp.ByQuery),

		// Wait for navigation after form submission
		chromedp.Sleep(3*time.Second), // Wait for 3 seconds
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
	)

	return err
}

func (mls *MLS) AddressStatus(addr string) (AddrStatus, error) {
	// setup URL
	addr = strings.ReplaceAll(addr, " ", "+")
	addr = strings.ReplaceAll(addr, ",", "")
	searchURL := MLS_SEARCH_URL_BASE + addr

	var jsonString string

	// fetch raw json
	err := chromedp.Run(mls.ctx,
		// Navigate to the next URL
		chromedp.Navigate(searchURL),

		// Wait for the body of the next page to be visible
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		// Extract inner JSON text
		chromedp.Text(`pre`, &jsonString, chromedp.ByQuery),
	)
	if err != nil {
		return addrError, err
	}

	// Strip out the `lookupCallback(...)` wrapper to extract the raw json
	prefix := "lookupCallback("
	suffix := ")"

	start := strings.Index(jsonString, prefix)
	end := strings.LastIndex(jsonString, suffix)
	if start == -1 || end == -1 {
		return addrError, fmt.Errorf("Invalid JSON wrapper")
	}

	cleanJSON := jsonString[start+len(prefix) : end]

	// Expected json
	type LookupResponse struct {
		D struct {
			Results []struct {
				Name string `json:"Name"`
			} `json:"Results"`
		} `json:"D"`
	}

	var data LookupResponse
	err = json.Unmarshal([]byte(cleanJSON), &data)
	if err != nil {
		return addrError, fmt.Errorf("Failed to parse JSON: %v", err)
	}

	// If no results, it failed
	if len(data.D.Results) == 0 {
		return addrError, fmt.Errorf("No results found - %s", addr)
	}

	// Get the status
	status := strings.Split(data.D.Results[0].Name, "(")[1]
	status = status[0 : len(status)-1]
	status = strings.ToLower(status)

	// Handle status
	switch status {
	case "active":
		return addrActive, nil
	case "coming soon":
		return addrComingSoon, nil
	default:
		return addrClosed, nil
	}
}

func (mls *MLS) LookupPerson(person Person) (*PersonStatus, error) {
	finalStatus := addrClosed
	var finalErr error
	for _, addr := range person.Addresses {
		status, err := mls.AddressStatus(addr.ToString())

		// Is active, so we know it is live
		if status == addrActive {
			finalStatus = status
			break
		}
		// Coming soon means it is just recently listed, good to know
		if status == addrComingSoon {
			finalStatus = status
			continue
		}
		// Handle error
		if err != nil {
			finalErr = err
		}
	}

	// one of the addresses could fail, but if none work then it is invalid address
	if finalErr != nil {
		return nil, finalErr
	}

	return &PersonStatus{
		id:     person.ID,
		status: finalStatus,
	}, nil
}

func (mls *MLS) Close() {
	mls.cancel()
}
