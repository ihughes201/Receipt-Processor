package main

import (
	"encoding/json"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

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

type ReceiptPoints struct {
	Id     string `json:"id"`
	Points int    `json:"points"`
}

type ResponseId struct {
	Id string `json:"id"`
}

type ResponsePoints struct {
	Points int `json:"points"`
}

//endregion structs

//region Helper Functions

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

func SaveReceiptPoints(id string, score int) bool {
	return true
}

func GetPointsForUuid(uuid string) int {
	return -1
}

//endregion Helper Functions

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

			success := SaveReceiptPoints(id, points)
			if success {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(ResponseId{
					Id: id,
				})
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("The receipt is invalid")
		return
	}

}

func getReceiptPoints(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// For each alpha numeric character in the retailer name, add 1 point
		// Note: does not include characters with accents
		reg, err := regexp.Compile("/receipts/([a-zA-Z0-9]+)/points")
		// if there are no errors creating the regex criteria, process the retailer name
		if err == nil {
			receiptUuid := reg.FindStringSubmatch(r.URL.Path)[1]

			points := GetPointsForUuid(receiptUuid)
			if points != -1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(ResponsePoints{
					Points: points,
				})
				return
			}

		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode("No receipt found for that id")
		return
	}
}

func main() {
	http.HandleFunc("/receipts/process", processReceipt)
	http.HandleFunc("/receipts/", getReceiptPoints)

	http.ListenAndServe(":8090", nil)
}
