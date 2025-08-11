package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
)

func handleLookupResults(fub *FUB, results []*PersonStatus) {
	for _, person := range results {
		// Don't do anything with people who have sold
		if !person.hasSold {
			continue
		}

		err := fub.SetPersonHasSold(person.id)
		if err != nil {
			log.Panic(err)
		}

		log.Printf("[INFO] %v: Stage updated - Has Sold", person.id)
	}
}

func main() {
	// load env file if exist
	godotenv.Load()
	// Set global varialbes from ENV
	initEnv()

	fub := NewFUB(FUBApiKey, FUBSellerSmartlistId)
	mls, err := BuildMLS(MLSUser, MLSPass)
	if err != nil {
		log.Panic(err)
	}
	defer mls.Close()

	// Loop Context
	isEnd := false
	offset := 0

	for {
		// Break context
		if isEnd {
			break
		}

		// Successful lookups to handle later
		lookupResults := make([]*PersonStatus, 0)

		/*
		 * Handle parsing a page of people
		 */

		// Fetch current people
		var currentPeople []Person
		currentPeople, isEnd, err = fub.GetPeoplePage(offset)
		if err != nil {
			log.Panic(err)
		}

		// Parse people from current list
		for _, person := range currentPeople {
			// Skip invalid people
			if len(person.Addresses) == 0 {
				log.Printf("[WARN] %v: Invalid User - No Addresses", person.ID)
				continue
			}

			// Skip excluded people
			if fub.PersonIsExcluded(&person) {
				continue
			}

			status, err := mls.PersonHasSoldSince(person, person.CreatedAt)
			if err != nil {
				log.Printf("[WARN] %v: %v", person.ID, err)
				continue
			}

			lookupResults = append(lookupResults, status)
		}

		//Handle successful lookupResults
		handleLookupResults(&fub, lookupResults)

		// Increment
		offset += 50
	}

	fmt.Print("Finished Program\n")
}
