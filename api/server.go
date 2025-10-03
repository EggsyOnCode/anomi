package api

import (
	"net/http"

	"github.com/EggysOnCode/anomi/api/handlers"
	"github.com/EggysOnCode/anomi/api/middleware"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/EggysOnCode/anomi/storage"
	"github.com/labstack/echo/v4"
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
	Orderbook   *engine.OrderBook
	KvDB        *storage.KvDB
	MsgProducer handlers.MessageProducer
}

// NewServer creates a new simplified API server
func NewServer(config *Config) *Server {
	e := echo.New()

	// Initialize handlers
	orderHandler := handlers.NewOrderHandler(config.Orderbook, config.KvDB, config.MsgProducer)
	tradeHandler := handlers.NewTradeHandler(config.KvDB, config.MsgProducer)
	receiptHandler := handlers.NewReceiptHandler(config.KvDB, config.MsgProducer)

	// Setup middleware
	e.Use(middleware.LoggingMiddleware())
	e.Use(middleware.RequestIDMiddleware())
	e.Use(middleware.ValidationMiddleware())
	e.Use(middleware.JSONBindingMiddleware())
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
	// TODO: Implement order creation
	return c.JSON(http.StatusNotImplemented, map[string]string{
		"error": "Order creation not yet implemented",
	})
}

func handleOrderGet(c echo.Context, handler *handlers.OrderHandler) error {
	orderID := c.Param("id")
	if orderID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Order ID is required",
		})
	}

	result := handler.GetOrder(c.Request().Context(), orderID)
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
	// TODO: Implement order update
	return c.JSON(http.StatusNotImplemented, map[string]string{
		"error": "Order update not yet implemented",
	})
}

func handleOrderCancel(c echo.Context, handler *handlers.OrderHandler) error {
	orderID := c.Param("id")
	if orderID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Order ID is required",
		})
	}

	result := handler.CancelOrder(c.Request().Context(), orderID)
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
