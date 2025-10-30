package main

import (
	"fmt"
	"log"
	"time"
)

func handleLookupResults(fub *FUB, results []int) {
	for _, person := range results {
		err := fub.SetPersonHasSold(person)
		if err != nil {
			log.Panic(err)
		}
		log.Printf("[INFO] %v: Stage updated - Has Sold", person)
	}
}

func main() {
	initConfig()

	// Confirm SMTP server is reachable
	err := VerifySMTPAuth(AppConfig.SMTP.Host, AppConfig.SMTP.Port, AppConfig.SMTP.User, AppConfig.SMTP.Pass)
	if err != nil {
		log.Fatal(err)
	}

	// Init services used in main loop
	fub := NewFUB(AppConfig.FUB.APIKey, AppConfig.FUB.SellerSmartlistIDs)
	mls, err := BuildMLS(AppConfig.MLS.User, AppConfig.MLS.Pass)
	if err != nil {
		log.Panic(err)
	}
	defer mls.Close()

	// Final context for sending out email
	updatedPeople := make([]Person, 0)

	// Iterate through each smart list ID
	for _, smartListId := range fub.sellerListIds {
		log.Printf("[INFO] Processing Smart List ID: %v", smartListId)

		// Loop Context for this smart list
		isEnd := false
		offset := 0

		for {
			// Break context
			if isEnd {
				break
			}

			/*
			 * Handle parsing a page of people
			 */
			// Fetch current people for this specific smart list
			var currentPeople []Person
			currentPeople, isEnd, err = fub.GetPeoplePage(smartListId, offset)
			if err != nil {
				log.Panic(err)
			}

			haveSoldIds := make([]int, 0)

			for _, person := range currentPeople {
				// Skip invalid people
				if len(person.Addresses) == 0 {
					log.Printf("[WARN] %v: Invalid User - No Addresses", person.ID)
					continue
				}

				// Skip excluded stages
				if fub.PersonIsExcluded(&person) {
					continue
				}

				status, err := mls.PersonHasSoldSince(person, person.CreatedAt)
				if err != nil {
					log.Printf("[WARN] %v: %v", person.ID, err)
					continue
				}

				if status.hasSold {
					haveSoldIds = append(haveSoldIds, status.id)
					updatedPeople = append(updatedPeople, person)
				}
			}

			// Handle successful lookupResults
			// Use small subset instead of doing them all at the end to avoid flooding FUB
			handleLookupResults(&fub, haveSoldIds)

			// Increment
			offset += FUB_BUFFFER_AMOUNT
		}

		log.Printf("[INFO] Completed processing Smart List ID: %v", smartListId)
	}

	// Send out email report
	title := fmt.Sprintf("Sold Listings - %s", time.Now().Format(time.DateOnly))
	if err = SendEmailReport(title, updatedPeople); err != nil {
		log.Fatalf("Failed to send email report: %v", err)
	}

	fmt.Print("Finished Program\n")
}
