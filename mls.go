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

type PersonStatus struct {
	id      int
	hasSold bool
}

func BuildMLS(user string, pass string) (mls *MLS, err error) {
	// Create context
	ctx, cancel := chromedp.NewContext(context.Background())

	// Set a timeout
	ctx, cancel = context.WithTimeout(ctx, 600*time.Second)

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

// Gets the list of dates the address has been listed
func (mls *MLS) mostRecentlySold(id string, mlsId string) (time.Time, error) {
	url := MLS_SEARCH_HISTORY_URL_BASE
	url = strings.Replace(url, "{id}", id, 1)
	url = strings.Replace(url, "{mlsid}", mlsId, 1)

	var date string
	err := chromedp.Run(mls.ctx,
		chromedp.Navigate(url),

		// Wait for table to load (adjust selector if needed)
		chromedp.WaitVisible(`tbody`, chromedp.ByQuery),

		// Extract the text of the first date in the table
		chromedp.Text(`tbody tr td.date`, &date, chromedp.ByQuery),
	)

	if err != nil {
		return time.Time{}, err
	}

	// Parse date into time.Time
	return time.Parse("01/02/2006", date)
}

func (mls *MLS) AddressHasSoldSince(addr string, time time.Time) (bool, error) {
	/*
	 * First, get the Id & MlsId from the address
	 */

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
		return false, err
	}

	// Strip out the `lookupCallback(...)` wrapper to extract the raw json
	prefix := "lookupCallback("
	suffix := ")"

	start := strings.Index(jsonString, prefix)
	end := strings.LastIndex(jsonString, suffix)
	if start == -1 || end == -1 {
		return false, fmt.Errorf("Invalid JSON wrapper")
	}

	cleanJSON := jsonString[start+len(prefix) : end]

	// Expected json
	type LookupResponse struct {
		D struct {
			Results []struct {
				Id    string `json:"Id"`
				MlsId string `json:"MlsId"`
			} `json:"Results"`
		} `json:"D"`
	}

	var data LookupResponse
	err = json.Unmarshal([]byte(cleanJSON), &data)
	if err != nil {
		return false, fmt.Errorf("Failed to parse JSON: %v", err)
	}

	// If no results, it failed
	if len(data.D.Results) == 0 {
		return false, fmt.Errorf("No results found - %s", addr)
	}

	/*
	 * Second, use Id & MlsId to see whether the address has been listed since [time]
	 */

	// If address has been sold after [time]
	recentlySold, err := mls.mostRecentlySold(data.D.Results[0].Id, data.D.Results[0].MlsId)
	if err != nil {
		return false, err
	}

	if recentlySold.After(time) {
		return true, nil
	} else {
		return false, nil
	}
}

func (mls *MLS) PersonHasSoldSince(person Person, time time.Time) (*PersonStatus, error) {

	hasSold, err := mls.AddressHasSoldSince(person.Addresses[0].ToString(), time)
	if err != nil {
		return nil, err
	}

	return &PersonStatus{
		id:      person.ID,
		hasSold: hasSold,
	}, nil
}

func (mls *MLS) Close() {
	mls.cancel()
}
