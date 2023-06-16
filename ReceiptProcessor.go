package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var receiptCache = make(map[string]int)

//region structs

type PurchaseItem struct {
	ShortDescription string  `json:"shortDescription"`
	Price            float64 `json:"price,string"`
}

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
func CalculatePoints(receipt Receipt) int {
	points := 0

	// For each alpha numeric character in the retailer name, add 1 point
	// Note: does not include characters with accents
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	// if there are no errors creating the regex criteria, process the retailer name
	if err == nil {
		processedRetailerName := reg.ReplaceAllString(receipt.Retailer, "")
		points += len(processedRetailerName)
	}

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

	datetime, err := time.Parse("2006-01-02 15:04", receipt.PurchaseDate+" "+receipt.PurchaseTime)

	if err == nil {
		// If the purchase date is odd, then add 6 points
		if datetime.Day()%2 == 1 {
			points += 6
		}

		// If the purchase time is after 14:00 and before 16:00, then add 10 points
		if (datetime.Hour() >= 14 && datetime.Minute() > 0) && (datetime.Hour() < 16) {
			points += 10
		}
	}

	return points
}

// Validate the URL and process the submitted receipt
func processReceipt(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/receipts/process" {
		if r.Method == "POST" {
			var receipt Receipt

			// decode the json package and exit if there is an error
			err := json.NewDecoder(r.Body).Decode(&receipt)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				// http.Error(w, "The receipt is invalid", http.StatusBadRequest)
				return
			}

			// Caclulate points for the receipt
			points := CalculatePoints(receipt)
			id := uuid.New().String()
			receiptCache[id] = points

			// Return the id for the processsed receipt
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(ResponseId{
				Id: id,
			})
			return
		}

		// return bad request if the receipt could not be processed
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "The receipt is invalid\n")
		return
	}
}

// return the receipt points for an id or Not Found
func getReceiptPoints(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Check if the id passed in matches the form we expect.
		// alphanum and a "-"
		reg, err := regexp.Compile("/receipts/([a-zA-Z0-9-]+)/points")
		// if there are no errors creating the regex criteria, process the retailer name
		if err == nil {
			var urlmatch = reg.FindStringSubmatch(r.URL.Path)

			// If the url matches the expected form
			if urlmatch != nil {
				receiptUuid := urlmatch[1]
				points, found := receiptCache[receiptUuid]

				// If the id does exist in the map, return the points
				if found {
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
	}

	// If the id is invalid or if no score was found, return page not found
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "404 page not found\n")
}

func main() {
	http.HandleFunc("/receipts/process", processReceipt)
	http.HandleFunc("/receipts/", getReceiptPoints)

	http.ListenAndServe(":8090", nil)
}
