package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
)

type FUB struct {
	token        string
	sellerListId int
	client       *http.Client
}

type PeopleMetadata struct {
	Collection string `json:"collection"`
	Offset     int    `json:"offset"`
	Limit      int    `json:"limit"`
	Total      int    `json:"total"`
	Next       string `json:"next"`
	NextLink   string `json:"nextLink"`
	Notice     string `json:"notice"`
}

type PersonAddress struct {
	Type    string `json:"type"`
	Street  string `json:"street"`
	City    string `json:"city"`
	State   string `json:"state"`
	Code    string `json:"code"`
	Country string `json:"country"`
}

type Person struct {
	ID        int             `json:"id"`
	Addresses []PersonAddress `json:"addresses"`
}

type PeopleResponse struct {
	Metadata PeopleMetadata `json:"_metadata"`
	People   []Person       `json:"people"`
}

func NewFUB(token string, smartListId string) FUB {
	client := &http.Client{}

	sellerListId, err := strconv.Atoi(smartListId)
	if err != nil {
		log.Panic(err)
	}

	return FUB{
		token,
		sellerListId,
		client,
	}
}

// Offset allows for recursion internally if response is paginated.
// Internal function for GetPeople
func (f *FUB) GetPeoplePage(offset int) (people []Person, isEnd bool, err error) {
	url := "https://api.followupboss.com/v1/people?sort=created&limit=10&offset=" + strconv.Itoa(offset) + "&includeTrash=false&includeUnclaimed=true&smartListId=" + strconv.Itoa(f.sellerListId)

	req, err := f.newRequest("GET", url, nil)
	if err != nil {
		return nil, false, err
	}

	res, err := f.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()

	var jsonRes PeopleResponse
	err = json.NewDecoder(res.Body).Decode(&jsonRes)
	if err != nil {
		return nil, false, err
	}

	people = jsonRes.People
	isEnd = jsonRes.Metadata.Next == "null"
	return people, isEnd, nil
}

func (f *FUB) newRequest(method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Basic auth requires Base64 encoding of API key
	auth := base64.StdEncoding.EncodeToString([]byte(f.token + ":"))

	req.Header.Add("X-System", FUB_SYSTEM_HEADER)
	req.Header.Add("X-System-Key", FUB_SYSTEM_KEY)
	req.Header.Add("accept", "application/json")
	req.Header.Add("authorization", "Basic "+auth)

	return req, nil
}

func (addr *PersonAddress) ToString() string {
	return fmt.Sprintf("%s, %s", addr.Street, addr.City)
}
