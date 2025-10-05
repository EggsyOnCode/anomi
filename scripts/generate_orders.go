package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

// OrderRequest represents the structure for creating orders
type OrderRequest struct {
	OrderType string `json:"orderType"`
	UserID    string `json:"userID"`
	Side      int    `json:"side"`
	Symbol    string `json:"symbol"`
	IsQuote   bool   `json:"isQuote"`
	Quantity  string `json:"quantity"`
	Price     string `json:"price"`
	Stop      string `json:"stop"`
	TIF       string `json:"tif"`
	OCO       string `json:"oco"`
}

// OrderResponse represents the response from the API
type OrderResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
}

const (
	// Order types
	MARKET     = "MARKET"
	LIMIT      = "LIMIT"
	STOP_LIMIT = "STOP-LIMIT"

	// Sides
	BUY  = 0
	SELL = 1

	// TIF types
	GTC = "GTC"
	FOK = "FOK"
	IOC = "IOC"

	// Symbols
	BTC_USD = "BTC/USD"
	ETH_USD = "ETH/USD"
)

var symbols = []string{BTC_USD, ETH_USD}
var orderTypes = []string{MARKET, LIMIT, STOP_LIMIT}
var tifTypes = []string{GTC, FOK, IOC}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run generate_orders.go <node_id>")
		os.Exit(1)
	}

	nodeID := os.Args[1]

	// Determine the API port based on node ID
	var apiPort string
	switch nodeID {
	case "node1":
		apiPort = "8081"
	case "node2":
		apiPort = "8082"
	default:
		apiPort = "8080"
	}

	baseURL := fmt.Sprintf("http://localhost:%s", apiPort)

	fmt.Printf("Starting order generation for node: %s on port: %s\n", nodeID, apiPort)

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// Generate orders continuously
	for {
		order := generateRandomOrder(nodeID)

		// Send order to the API
		if err := sendOrder(baseURL, order); err != nil {
			fmt.Printf("Error sending order: %v\n", err)
		} else {
			fmt.Printf("Order sent successfully: %s %s %s %s @ %s\n",
				order.OrderType,
				getSideString(order.Side),
				order.Quantity,
				order.Symbol,
				order.Price)
		}

		// Wait between orders (1-5 seconds)
		waitTime := time.Duration(rand.Intn(4)+1) * time.Second
		time.Sleep(waitTime)
	}
}

func generateRandomOrder(nodeID string) OrderRequest {
	// Random symbol
	symbol := symbols[rand.Intn(len(symbols))]

	// Random order type
	orderType := orderTypes[rand.Intn(len(orderTypes))]

	// Random side (buy/sell)
	side := rand.Intn(2)

	// Random quantity (0.1 to 10.0)
	quantity := fmt.Sprintf("%.2f", rand.Float64()*9.9+0.1)

	// Random price based on symbol
	var price string
	if symbol == BTC_USD {
		price = fmt.Sprintf("%.2f", rand.Float64()*20000+30000) // $30k-$50k
	} else {
		price = fmt.Sprintf("%.2f", rand.Float64()*2000+1000) // $1k-$3k
	}

	// Random TIF for limit orders
	var tif string
	if orderType == LIMIT {
		tif = tifTypes[rand.Intn(len(tifTypes))]
	}

	// Random stop price for stop-limit orders
	var stop string
	if orderType == STOP_LIMIT {
		basePrice, _ := strconv.ParseFloat(price, 64)
		stop = fmt.Sprintf("%.2f", basePrice*0.95) // 5% below limit price
	}

	return OrderRequest{
		OrderType: orderType,
		UserID:    fmt.Sprintf("user_%s_%d", nodeID, rand.Intn(1000)),
		Side:      side,
		Symbol:    symbol,
		IsQuote:   false,
		Quantity:  quantity,
		Price:     price,
		Stop:      stop,
		TIF:       tif,
		OCO:       "",
	}
}

func sendOrder(baseURL string, order OrderRequest) error {
	jsonData, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %v", err)
	}

	// Debug: print the JSON being sent
	fmt.Printf("Sending JSON: %s\n", string(jsonData))

	url := fmt.Sprintf("%s/api/v1/orders", baseURL)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Debug: print response body for errors
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		fmt.Printf("Error response: %s\n", string(body[:n]))
		return fmt.Errorf("API returned status code: %d", resp.StatusCode)
	}

	return nil
}

func getSideString(side int) string {
	if side == BUY {
		return "BUY"
	}
	return "SELL"
}
