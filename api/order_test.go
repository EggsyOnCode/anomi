package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/EggysOnCode/anomi/storage"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestOrderCreation(t *testing.T) {
	// Create a test orderbook and kvdb
	orderbook := engine.NewOrderBook()
	kvdb := &storage.KvDB{} // You might need to mock this properly

	// Create server
	config := &Config{
		Port:      "8080",
		Orderbook: orderbook,
		KvDB:      kvdb,
	}
	server := NewServer(config)

	// Test cases
	tests := []struct {
		name           string
		request        CreateOrderRequest
		expectedStatus int
	}{
		{
			name: "Valid Market Order",
			request: CreateOrderRequest{
				OrderType: "MARKET",
				UserID:    "user123",
				Side:      0, // BUY
				Quantity:  "100.50",
				IsQuote:   false,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Valid Limit Order",
			request: CreateOrderRequest{
				OrderType: "LIMIT",
				UserID:    "user123",
				Side:      1, // SELL
				Quantity:  "50.25",
				Price:     "1000.00",
				TIF:       "GTC",
				IsQuote:   false,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Invalid Order - Missing UserID",
			request: CreateOrderRequest{
				OrderType: "MARKET",
				UserID:    "",
				Side:      0,
				Quantity:  "100.50",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid Order - Missing Price for Limit",
			request: CreateOrderRequest{
				OrderType: "LIMIT",
				UserID:    "user123",
				Side:      0,
				Quantity:  "100.50",
				TIF:       "GTC",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			reqBody, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Create echo context
			e := echo.New()
			c := e.NewContext(req, rec)

			// Call handler
			err := handleOrderCreate(c, server.orderHandler)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestOrderUpdate(t *testing.T) {
	// Create a test orderbook and kvdb
	orderbook := engine.NewOrderBook()
	kvdb := &storage.KvDB{} // You might need to mock this properly

	// Create server
	config := &Config{
		Port:      "8080",
		Orderbook: orderbook,
		KvDB:      kvdb,
	}
	server := NewServer(config)

	// Test cases
	tests := []struct {
		name           string
		orderID        string
		request        UpdateOrderRequest
		expectedStatus int
	}{
		{
			name:    "Valid Update - Quantity Only",
			orderID: "test-order-123",
			request: UpdateOrderRequest{
				Quantity: "200.00",
			},
			expectedStatus: http.StatusNotFound, // Order doesn't exist
		},
		{
			name:           "Invalid Update - No Fields",
			orderID:        "test-order-123",
			request:        UpdateOrderRequest{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			reqBody, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/orders/"+tt.orderID, bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// Create echo context
			e := echo.New()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.orderID)

			// Call handler
			err := handleOrderUpdate(c, server.orderHandler)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}
