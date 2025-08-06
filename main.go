package main

import (
	"fmt"
)

func getLocations() (locations []string) {
	locations = make([]string, 0)

	// Random lot from zillow
	locations = append(locations, "2074 Hard Rd Columbus OH 43235")

	return
}

func addressDetails(addr string) string {
	return ""
}

func fetchLocationDetails(locations []string) (details []string) {
	details = make([]string, len(locations))

	for _, addr := range locations {
		details = append(details, addressDetails(addr))
	}

	return
}

func main() {
	locations := getLocations()
	details := fetchLocationDetails(locations)

	for _, item := range details {
		fmt.Printf("item: %v\n", item)
	}
}
