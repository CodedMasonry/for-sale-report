package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// load env file if exist
	godotenv.Load()

	// ENV
	FUBApiKey := os.Getenv("FUB_KEY")
	FUBSellerSmartlistId := os.Getenv("FUB_SMARTLIST_SELLER_ID")
	MLSUser := os.Getenv("MLS_USER")
	MLSPass := os.Getenv("MLS_PASS")

	fub := NewFUB(FUBApiKey, FUBSellerSmartlistId)
	mls, err := BuildMLS(MLSUser, MLSPass)
	if err != nil {
		log.Panic(err)
	}
	defer mls.Close()

	// Loop Context
	var currentPeople []Person
	isEnd := false
	offset := 0

	// Loop result
	lookupResults := make([]*PersonStatus, 0)

	for {
		// Break context
		if isEnd || offset == 1 {
			break
		}

		// Fetch current people
		currentPeople, isEnd, err = fub.GetPeoplePage(offset)
		if err != nil {
			log.Panic(err)
		}

		// Parse people from current list
		for _, person := range currentPeople {
			// Skip invalid people
			if len(person.Addresses) == 0 {
				log.Printf("[WARN] Invalid User: %v - No Addresses", person.ID)
				continue
			}

			status, err := mls.LookupPerson(person)
			if err != nil {
				log.Printf("[WARN] %v", person.ID, err)
				continue
			}
			lookupResults = append(lookupResults, status)
		}

		// Increment
		offset++
	}

	for _, v := range lookupResults {
		fmt.Printf("%v - %v\n", v.id, v.status)
	}
}
