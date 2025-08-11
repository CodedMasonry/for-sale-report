package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

type FUB struct {
	token        string
	sellerListId int
	client       *http.Client
}

// Only next is cared about because offset context is handled internally. Communicates end of list
type PeopleMetadata struct {
	Next  *string `json:"next"`
	Total int     `json:"total"`
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
	CreatedAt time.Time       `json:"created"`
	Stage     string          `json:"stage"`
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

	if body != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

// Offset allows for recursion internally if response is paginated.
// Internal function for GetPeople
func (f *FUB) GetPeoplePage(offset int) (people []Person, isEnd bool, err error) {
	url := "https://api.followupboss.com/v1/people?sort=created&limit=50&offset=" + strconv.Itoa(offset) + "&includeTrash=false&includeUnclaimed=true&fields=id%2Ccreated%2Cstage%2Caddresses&smartListId=" + strconv.Itoa(f.sellerListId)

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
	fmt.Printf("[INFO] %v / %v people", offset, jsonRes.Metadata.Total)
	isEnd = jsonRes.Metadata.Next == nil
	return people, isEnd, nil
}

// internal version of [SetPersonHasSold] for handling zillow tag
func (f *FUB) setZillowPersonHasSold(id int) error {
	url := "https://api.followupboss.com/v1/people/" + strconv.Itoa(id) + "?mergeTags=true"
	payload := strings.NewReader("{\"tags\":[\"Expired Lead\"]}")

	req, err := f.newRequest("PUT", url, payload)
	if err != nil {
		return err
	}

	res, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		return nil
	}
	// If neither original or zillow tag works, bigger problems
	body, err := io.ReadAll(res.Body)
	return fmt.Errorf("%v: Failed to set tag %v", id, body)
}

// Sets [id]'s stage to [FUBExpiredStageName]
func (f *FUB) SetPersonHasSold(id int) error {
	url := "https://api.followupboss.com/v1/people/" + strconv.Itoa(id) + "?mergeTags=true"
	payload := strings.NewReader("{\"tags\":[\"Expired Lead\"]}")

	req, err := f.newRequest("PUT", url, payload)
	if err != nil {
		return err
	}

	res, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		return nil
	}
	// If not a success, see if zillow lead
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	if strings.Contains(string(body), "Zillow") {
		log.Printf("%v: Trying to handle Zillow Lead", id)
		return f.setZillowPersonHasSold(id)
	} else {
		return fmt.Errorf("%v: Failed to set tag - %v", id, body)
	}
}

func (fub *FUB) PersonIsExcluded(person *Person) bool {
	// If person.stage is an excluded stage
	return slices.Contains(AppConfig.FUB.ExcludedStages, person.Stage)
}

func (addr *PersonAddress) ToString() string {
	// Invalid address
	if addr.Street == "" {
		return ""
	}

	return fmt.Sprintf("%s, %s", addr.Street, addr.City)
}
