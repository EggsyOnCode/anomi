package api

import (
	"net/http"
	"time"

	"github.com/EggysOnCode/anomi/api/handlers"
	"github.com/EggysOnCode/anomi/api/middleware"
	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/EggysOnCode/anomi/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/nikolaydubina/fpdecimal"
	"go.uber.org/zap"
)

// Server represents a simplified API server
type Server struct {
	echo           *echo.Echo
	orderHandler   *handlers.OrderHandler
	tradeHandler   *handlers.TradeHandler
	receiptHandler *handlers.ReceiptHandler
}

// Config holds server configuration
type Config struct {
	Port        string
	Orderbooks  []*orderbook.OrderBook
	KvDB        *storage.KvDB
	MsgProducer handlers.MessageProducer
	Logger      *zap.Logger
}

// NewServer creates a new simplified API server
func NewServer(config *Config) *Server {
	e := echo.New()

	// Initialize handlers
	orderHandler := handlers.NewOrderHandler(config.Orderbooks, config.KvDB, config.MsgProducer, config.Logger)
	tradeHandler := handlers.NewTradeHandler(config.KvDB, config.MsgProducer, config.Logger)
	receiptHandler := handlers.NewReceiptHandler(config.KvDB, config.MsgProducer, config.Logger)

	// Setup middleware
	e.Use(middleware.LoggingMiddleware())
	e.Use(middleware.RequestIDMiddleware())
	e.Use(middleware.ValidationMiddleware())
	// e.Use(middleware.JSONBindingMiddleware()) // Commented out as it consumes request body
	e.Use(middleware.AuthMiddleware())

	// Setup routes
	setupRoutes(e, orderHandler, tradeHandler, receiptHandler)

	return &Server{
		echo:           e,
		orderHandler:   orderHandler,
		tradeHandler:   tradeHandler,
		receiptHandler: receiptHandler,
	}
}

// setupRoutes configures the HTTP routes
func setupRoutes(
	e *echo.Echo,
	orderHandler *handlers.OrderHandler,
	tradeHandler *handlers.TradeHandler,
	receiptHandler *handlers.ReceiptHandler,
) {
	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
		})
	})

	// API v1 routes
	v1 := e.Group("/api/v1")

	// Order routes
	orders := v1.Group("/orders")
	orders.POST("", func(c echo.Context) error {
		return handleOrderCreate(c, orderHandler)
	})
	orders.GET("/:id", func(c echo.Context) error {
		return handleOrderGet(c, orderHandler)
	})
	orders.PUT("/:id", func(c echo.Context) error {
		return handleOrderUpdate(c, orderHandler)
	})
	orders.DELETE("/:id", func(c echo.Context) error {
		return handleOrderCancel(c, orderHandler)
	})

	// Trade routes
	trades := v1.Group("/trades")
	trades.GET("/:id", func(c echo.Context) error {
		return handleTradeGet(c, tradeHandler)
	})

	// Receipt routes
	receipts := v1.Group("/receipts")
	receipts.GET("/:id", func(c echo.Context) error {
		return handleReceiptGet(c, receiptHandler)
	})
}

// Handler functions
func handleOrderCreate(c echo.Context, handler *handlers.OrderHandler) error {
	var req CreateOrderRequest

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			"Invalid request format",
			"Failed to parse request body",
		))
	}

	// Basic validation
	if req.OrderType == "" || req.UserID == "" || req.Quantity == "" || req.Symbol == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			"Missing required fields",
			"orderType, userID, quantity, and symbol are required",
		))
	}

	// Generate order ID
	orderID := uuid.New().String()

	// Parse quantity
	quantity, err := fpdecimal.FromString(req.Quantity)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			"Invalid quantity format",
			"Quantity must be a valid decimal number",
		))
	}

	// Convert side
	var side engine.Side
	if req.Side == 0 {
		side = engine.Buy
	} else {
		side = engine.Sell
	}

	// Create order based on type
	var order *engine.Order
	switch req.OrderType {
	case "MARKET":
		if req.IsQuote {
			order = engine.NewMarketQuoteOrder(orderID, side, quantity, req.UserID)
		} else {
			order = engine.NewMarketOrder(orderID, side, quantity, req.UserID)
		}

	case "LIMIT":
		// Validate required fields for limit orders
		if req.Price == "" {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(
				"Missing required field",
				"Price is required for LIMIT orders",
			))
		}
		if req.TIF == "" {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(
				"Missing required field",
				"TIF is required for LIMIT orders",
			))
		}

		// Parse price
		price, err := fpdecimal.FromString(req.Price)
		if err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(
				"Invalid price format",
				"Price must be a valid decimal number",
			))
		}

		// Parse TIF
		tif := engine.TIF(req.TIF)
		if tif != engine.GTC && tif != engine.FOK && tif != engine.IOC {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(
				"Invalid TIF value",
				"TIF must be one of: GTC, FOK, IOC",
			))
		}

		order = engine.NewLimitOrder(orderID, side, quantity, price, tif, req.OCO, req.UserID)
		if req.IsQuote {
			// Note: For quote orders, we need to handle this differently
			// This is a simplified implementation
		}

	case "STOP-LIMIT":
		// Validate required fields for stop-limit orders
		if req.Price == "" {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(
				"Missing required field",
				"Price is required for STOP-LIMIT orders",
			))
		}
		if req.Stop == "" {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(
				"Missing required field",
				"Stop price is required for STOP-LIMIT orders",
			))
		}

		// Parse price and stop price
		price, err := fpdecimal.FromString(req.Price)
		if err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(
				"Invalid price format",
				"Price must be a valid decimal number",
			))
		}

		stop, err := fpdecimal.FromString(req.Stop)
		if err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponse(
				"Invalid stop price format",
				"Stop price must be a valid decimal number",
			))
		}

		order = engine.NewStopLimitOrder(orderID, side, quantity, price, stop, req.OCO, req.UserID)

	default:
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			"Invalid order type",
			"Order type must be one of: MARKET, LIMIT, STOP-LIMIT",
		))
	}

	// Process order through handler
	// Symbol is required now
	if req.Symbol == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			"Missing symbol",
			"Symbol (BASE/QUOTE) is required",
		))
	}

	result := handler.CreateOrder(c.Request().Context(), req.Symbol, order)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(
			"Order creation failed",
			result.Error.Error(),
		))
	}

	// Convert order to response format
	orderResp := convertOrderToResponse(order)

	return c.JSON(http.StatusCreated, NewSuccessResponse(orderResp, result.Message))
}

func handleOrderGet(c echo.Context, handler *handlers.OrderHandler) error {
	orderID := c.Param("id")
	if orderID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Order ID is required",
		})
	}

	symbol := c.QueryParam("symbol")
	if symbol == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Symbol query parameter is required",
		})
	}
	result := handler.GetOrder(c.Request().Context(), symbol, orderID)
	if result.Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": result.Error.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result.Data,
		"message": result.Message,
	})
}

func handleOrderUpdate(c echo.Context, handler *handlers.OrderHandler) error {
	orderID := c.Param("id")
	if orderID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			"Missing order ID",
			"Order ID is required",
		))
	}

	var req UpdateOrderRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			"Invalid request format",
			"Failed to parse request body",
		))
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			"Validation failed",
			err.Error(),
		))
	}

	// Check if at least one field is provided for update
	if req.Quantity == "" && req.Price == "" && req.Stop == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			"No fields to update",
			"At least one field (quantity, price, stop) must be provided",
		))
	}

	// Get existing order
	if req.Symbol == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			"Missing symbol",
			"Symbol (BASE/QUOTE) is required",
		))
	}
	getResult := handler.GetOrder(c.Request().Context(), req.Symbol, orderID)
	if getResult.Error != nil {
		return c.JSON(http.StatusNotFound, NewErrorResponse(
			"Order not found",
			getResult.Error.Error(),
		))
	}

	existingOrder, ok := getResult.Data.(*engine.Order)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(
			"Invalid order data",
			"Failed to retrieve order data",
		))
	}

	// Check if order is canceled
	if existingOrder.IsCanceled() {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			"Cannot update canceled order",
			"Order has been canceled and cannot be updated",
		))
	}

	// Create updated order with new values
	updatedOrder := createUpdatedOrder(existingOrder, &req)
	if updatedOrder == nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			"Invalid update parameters",
			"Failed to create updated order with provided parameters",
		))
	}

	// Process order update through handler
	result := handler.UpdateOrder(c.Request().Context(), req.Symbol, updatedOrder)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(
			"Order update failed",
			result.Error.Error(),
		))
	}

	// Convert order to response format
	orderResp := convertOrderToResponse(updatedOrder)

	return c.JSON(http.StatusOK, NewSuccessResponse(orderResp, result.Message))
}

func handleOrderCancel(c echo.Context, handler *handlers.OrderHandler) error {
	orderID := c.Param("id")
	if orderID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Order ID is required",
		})
	}

	symbol := c.QueryParam("symbol")
	if symbol == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Symbol query parameter is required",
		})
	}
	result := handler.CancelOrder(c.Request().Context(), symbol, orderID)
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": result.Error.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result.Data,
		"message": result.Message,
	})
}

func handleTradeGet(c echo.Context, handler *handlers.TradeHandler) error {
	tradeID := c.Param("id")
	if tradeID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Trade ID is required",
		})
	}

	result := handler.GetTrade(c.Request().Context(), tradeID)
	if result.Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": result.Error.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result.Data,
		"message": result.Message,
	})
}

func handleReceiptGet(c echo.Context, handler *handlers.ReceiptHandler) error {
	receiptID := c.Param("id")
	if receiptID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Receipt ID is required",
		})
	}

	result := handler.GetReceipt(c.Request().Context(), receiptID)
	if result.Error != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": result.Error.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    result.Data,
		"message": result.Message,
	})
}

// Start starts the HTTP server
func (s *Server) Start(port string) error {
	return s.echo.Start(":" + port)
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	return s.echo.Close()
}

// GetEcho returns the Echo instance for advanced configuration
func (s *Server) GetEcho() *echo.Echo {
	return s.echo
}

// Helper functions

// convertOrderToResponse converts an engine.Order to OrderResponse
func convertOrderToResponse(order *engine.Order) *OrderResponse {
	return &OrderResponse{
		ID:          order.ID(),
		OrderType:   order.OrderType(),
		UserID:      order.UserID(),
		Side:        order.Side().String(),
		IsQuote:     order.IsQuote(),
		Quantity:    order.Quantity().String(),
		OriginalQty: order.OriginalQty().String(),
		Price:       order.Price().String(),
		Stop:        order.StopPrice().String(),
		Canceled:    order.IsCanceled(),
		Role:        string(order.Role()),
		TIF:         string(order.TIF()),
		OCO:         order.OCO(),
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}
}

// createUpdatedOrder creates a new order with updated fields
func createUpdatedOrder(existingOrder *engine.Order, req *UpdateOrderRequest) *engine.Order {
	// Start with existing order values
	quantity := existingOrder.Quantity()
	price := existingOrder.Price()
	stop := existingOrder.StopPrice()

	// Update quantity if provided
	if req.Quantity != "" {
		newQuantity, err := fpdecimal.FromString(req.Quantity)
		if err != nil {
			return nil
		}
		quantity = newQuantity
	}

	// Update price if provided
	if req.Price != "" {
		newPrice, err := fpdecimal.FromString(req.Price)
		if err != nil {
			return nil
		}
		price = newPrice
	}

	// Update stop price if provided
	if req.Stop != "" {
		newStop, err := fpdecimal.FromString(req.Stop)
		if err != nil {
			return nil
		}
		stop = newStop
	}

	// Create new order based on existing order type
	switch existingOrder.OrderType() {
	case "MARKET":
		if existingOrder.IsQuote() {
			return engine.NewMarketQuoteOrder(existingOrder.ID(), existingOrder.Side(), quantity, existingOrder.UserID())
		}
		return engine.NewMarketOrder(existingOrder.ID(), existingOrder.Side(), quantity, existingOrder.UserID())

	case "LIMIT":
		return engine.NewLimitOrder(existingOrder.ID(), existingOrder.Side(), quantity, price, existingOrder.TIF(), existingOrder.OCO(), existingOrder.UserID())

	case "STOP-LIMIT":
		return engine.NewStopLimitOrder(existingOrder.ID(), existingOrder.Side(), quantity, price, stop, existingOrder.OCO(), existingOrder.UserID())

	default:
		return nil
	}
}
