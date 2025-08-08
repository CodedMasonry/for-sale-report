package main

import (
	"log"
	"os"
)

// MLS
const MLS_LOGIN_URL = "https://cr.flexmls.com/"
const MLS_SEARCH_URL_BASE = "https://apps.flexmls.com/quick_launch/herald?callback=lookupCallback&_filter="

// Replace {id} with Id and {mlsid} with MLS Id from MLS_SEARCH_URL result
const MLS_SEARCH_HISTORY_URL_BASE = "https://cr.flexmls.com/cgi-bin/mainmenu.cgi?cmd=srv%20srch_rs/detail/addr_hist.html&list_tech_id=x%27{id}%27&srch=Y&ma_search_list=x%27{mlsid}%27"

// FollowUpBoss
const FUB_SYSTEM_HEADER = "ForSaleReport"                 // X-System
const FUB_SYSTEM_KEY = "e50150b78203e92245f6407fdea50dab" // X-System-Key

// Runtime ENV variables
var (
	FUBApiKey                 string
	FUBSellerSmartlistId      string
	FUBExpiredStageName       string
	FUBExpiredZillowStageName string
	MLSUser                   string
	MLSPass                   string
)

func getEnv(key string) string {
	str := os.Getenv(key)
	if str == "" {
		log.Fatalf("Expected ENV variable: %v", key)
	}

	return str
}

func initEnv() {
	FUBApiKey = getEnv("FUB_KEY")
	FUBSellerSmartlistId = getEnv("FUB_SMARTLIST_SELLER_ID")
	FUBExpiredStageName = getEnv("FUB_EXPIRED_STAGE_NAME")
	FUBExpiredZillowStageName = getEnv("FUB_EXPIRED_ZILLOW_STAGE_NAME")
	MLSUser = getEnv("MLS_USER")
	MLSPass = getEnv("MLS_PASS")
}
