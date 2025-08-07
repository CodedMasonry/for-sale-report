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

	// setup MLS
	MLSUser := os.Getenv("MLS_USER")
	MLSPass := os.Getenv("MLS_PASS")
	mls := BuildMLS(MLSUser, MLSPass)
	defer mls.Close()

	result, err := mls.AddressStatus("6690 Mooney St #H1-13, Dublin, OH 43017")
	if err != nil {
		log.Panic(err)
	}

	if result == 3 {
		fmt.Printf("Open")
	} else {
		fmt.Printf("Closed")
	}
}
