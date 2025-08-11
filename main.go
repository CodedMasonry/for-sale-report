package main

import (
	"fmt"
	"log"

	"github.com/joho/godotenv"
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

		/*
		 * Handle parsing a page of people
		 */

		// Fetch current people
		var currentPeople []Person
		currentPeople, isEnd, err = fub.GetPeoplePage(offset)
		if err != nil {
			log.Panic(err)
		}

		// Id's of people who have sold
		haveSoldIds := make([]int, 0)

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

			if status.hasSold {
				haveSoldIds = append(haveSoldIds, status.id)
			}
		}

		//Handle successful lookupResults
		handleLookupResults(&fub, haveSoldIds)

		// Increment
		offset += 50
	}

	fmt.Print("Finished Program\n")
}
