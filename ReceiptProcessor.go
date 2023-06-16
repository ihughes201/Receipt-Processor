package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// map to contain all the point and uuids for receipts that have been processed
var receiptCache = make(map[string]int)

//region structs

// an item on the receipt
type PurchaseItem struct {
	ShortDescription string  `json:"shortDescription"`
	Price            float64 `json:"price,string"`
}

// the receipt to be processed
type Receipt struct {
	Retailer       string         `json:"retailer"`
	PurchaseDate   string         `json:"purchaseDate"`
	PurchaseTime   string         `json:"purchaseTime"`
	ItemsPurchased []PurchaseItem `json:"items"`
	PurchaseTotal  float64        `json:"total,string"`
}

type ResponseId struct {
	Id string `json:"id"`
}

type ResponsePoints struct {
	Points int `json:"points"`
}

//endregion structs

// Calculate the total number of points a receipt earned.
func CalculatePoints(receipt Receipt) (int, error) {
	points := 0

	// For each alpha numeric character in the retailer name, add 1 point
	// Note: does not include characters with accents

	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	// if there are no errors creating the regex criteria, process the retailer name
	if err != nil {
		return -1, errors.New("failed name parsing")
	}
	processedRetailerName := reg.ReplaceAllString(receipt.Retailer, "")
	points += len(processedRetailerName)

	// If the purchase total is a multiple of "1.00" (whole dollars, no cents), then add 50 points
	// Make sure the purchase is positive, not zero or negative
	if receipt.PurchaseTotal > 0 && math.Remainder(receipt.PurchaseTotal, 1) == 0 {
		points += 50
	}

	// If the purchase total is a multiple of "0.25", then add 25 points
	// Make sure the purchase is positive, not zero or negative
	if receipt.PurchaseTotal > 0 && math.Remainder(receipt.PurchaseTotal, 0.25) == 0 {
		points += 25
	}

	// For every 2 items on the receipt, add 5 points
	points += (len(receipt.ItemsPurchased) / 2) * 5

	// For each item purchased,
	// If the item description without leading or trailing whitespace,
	//		Add (item price * 0.2) rounded up number of points
	for i := 0; i < len(receipt.ItemsPurchased); i++ {
		item := receipt.ItemsPurchased[i]

		if len(strings.TrimSpace(item.ShortDescription))%3 == 0 {
			points += int(math.Ceil(item.Price * 0.2))
		}
	}

	// Parse the date and time strings into 1 datetime object
	// If the date portion was not included, skip it
	if receipt.PurchaseDate != "" {
		dateDatetime, err := time.Parse("2006-01-02", receipt.PurchaseDate)
		if err == nil {
			// If the purchase date is odd, then add 6 points
			if dateDatetime.Day()%2 == 1 {
				points += 6
			}
		}
	}

	// Parse the date and time strings into 1 datetime object
	// If the time portion was not included, skip it
	if receipt.PurchaseTime != "" {
		timeDatetime, err := time.Parse("15:04", receipt.PurchaseTime)
		if err == nil {
			// If the purchase time is after 14:00 and before 16:00, then add 10 points
			if (timeDatetime.Hour() >= 14 && timeDatetime.Minute() > 0) && (timeDatetime.Hour() < 16) {
				points += 10
			}
		}
	}

	return points, nil
}

// wrapper function for URL path requests starting with "/receipts"
func ReceiptsAPIProcessor(w http.ResponseWriter, r *http.Request) {

	// process receipt submissions
	if r.URL.Path == "/receipts/process" && r.Method == "POST" {
		var receipt Receipt

		// decode the json package and exit if there is an error
		err := json.NewDecoder(r.Body).Decode(&receipt)

		// continue if there were no errors decoding the json
		if err == nil {
			id, err := ProcessReceipt(receipt)

			// If there were no errors processing the receipt, return the id
			if err == nil {
				// Return the id for the processsed receipt
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(ResponseId{
					Id: id,
				})
				return
			}
		}

		// return bad request if the receipt could not be processed
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "The receipt is invalid\n")
		return
	}

	// Create the regex for the "get points for receipt"
	// if there are no errors creating the regex criteria, process the URL path
	pointsRegEx, err := regexp.Compile("/receipts/([a-zA-Z0-9-]+)/points")
	if err == nil {
		var urlmatch = pointsRegEx.FindStringSubmatch(r.URL.Path)

		// If the URL path matches the expected form, continue
		if urlmatch != nil && r.Method == "GET" {
			receiptUuid := urlmatch[1]

			points, err := GetReceiptPoints(receiptUuid)

			if err == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(ResponsePoints{
					Points: points,
				})
				return
			}

			// If the id is invalid or if no score was found, return not found
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "No receipt found for that id\n")
			return
		}
	}

	// If the URL path does not match anything, return not found
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "404 page not found\n")
}

// Process the submitted receipt
func ProcessReceipt(receipt Receipt) (string, error) {
	// Caclulate points for the receipt
	points, err := CalculatePoints(receipt)

	if err == nil {
		id := uuid.New().String()
		receiptCache[id] = points

		return id, nil
	}

	return "", err
}

// return the receipt points for an id or (0, error)
func GetReceiptPoints(receiptUuid string) (int, error) {
	points, found := receiptCache[receiptUuid]

	// If the uuid is found in the map, return the points
	if found {
		return points, nil
	}

	// if the uuid is not in the map, return 0 and an error
	return points, errors.New("uuid not found")
}

func main() {
	http.HandleFunc("/receipts/", ReceiptsAPIProcessor)

	http.ListenAndServe(":8080", nil)
}
