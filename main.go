package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Receipt structure to hold the receipt data
type Receipt struct {
	Retailer     string  `json:"retailer"`
	PurchaseDate string  `json:"purchaseDate"`
	PurchaseTime string  `json:"purchaseTime"`
	Items        []Item  `json:"items"`
	Total        float64 `json:"total,string"`
}

type Item struct {
	ShortDescription string  `json:"shortDescription"`
	Price            float64 `json:"price,string"`
}

var (
	receiptStore = make(map[string]Receipt)
	scoreStore   = make(map[string]int)
	mu           sync.Mutex
)

func main() {
	http.HandleFunc("/receipts/process", processReceiptHandler)
	http.HandleFunc("/receipts/", getPointsHandler)

	fmt.Println("Server is running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func processReceiptHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var receipt Receipt
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		http.Error(w, "Invalid receipt format", http.StatusBadRequest)
		return
	}

	 // Generate a unique ID using timestamp and random number
	 id := generateUniqueID()

	mu.Lock()
	receiptStore[id] = receipt
	scoreStore[id] = calculatePoints(receipt)
	mu.Unlock()

	response := map[string]string{"id": id}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func generateUniqueID() string {
    rand.Seed(time.Now().UnixNano())
    timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
    randomNum := strconv.Itoa(rand.Intn(10000))
    return timestamp + randomNum
}


func calculatePoints(receipt Receipt) int {
	points := 0

	// 1. One point for every alphanumeric character in the retailer name.
	for _, char := range receipt.Retailer {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			points++
		}
	}

	// 2. 50 points if the total is a round dollar amount with no cents.
	if receipt.Total == float64(int(receipt.Total)) {
		points += 50
	}

	// 3. 25 points if the total is a multiple of 0.25.
	if math.Mod(receipt.Total, 0.25) == 0 {
		points += 25
	}

	// 4. 5 points for every two items on the receipt.
	points += (len(receipt.Items) / 2) * 5

	// 5. If the trimmed length of the item description is a multiple of 3, multiply the price by 0.2 and round up to the nearest integer.
	for _, item := range receipt.Items {
		descLength := len(strings.TrimSpace(item.ShortDescription))
		if descLength%3 == 0 {
			points += int(math.Ceil(item.Price * 0.2))
		}
	}

	// 6. 6 points if the day in the purchase date is odd.
	if day, err := strconv.Atoi(strings.Split(receipt.PurchaseDate, "-")[2]); err == nil && day%2 != 0 {
		points += 6
	}

	// 7. 10 points if the time of purchase is after 2:00pm and before 4:00pm.
	if purchaseTime, err := time.Parse("15:04", receipt.PurchaseTime); err == nil {
		if purchaseTime.After(time.Date(0, 1, 1, 14, 0, 0, 0, time.UTC)) && purchaseTime.Before(time.Date(0, 1, 1, 16, 0, 0, 0, time.UTC)) {
			points += 10
		}
	}

	return points
}

func getPointsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Path[len("/receipts/"):len(r.URL.Path)-len("/points")]

	mu.Lock()
	points, exists := scoreStore[id]
	mu.Unlock()

	if !exists {
		http.Error(w, "Receipt not found", http.StatusNotFound)
		return
	}

	response := map[string]int{"points": points}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
