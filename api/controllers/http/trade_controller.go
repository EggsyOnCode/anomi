package http

import (
	"net/http"
	"strconv"

	"github.com/EggysOnCode/anomi/api"
	"github.com/EggysOnCode/anomi/api/handlers"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/labstack/echo/v4"
)

// TradeController handles HTTP requests for trades
type TradeController struct {
	tradeHandler *handlers.TradeHandler
}

// NewTradeController creates a new trade controller
func NewTradeController(tradeHandler *handlers.TradeHandler) *TradeController {
	return &TradeController{
		tradeHandler: tradeHandler,
	}
}

// GetTrade handles GET /trades/:id
func (c *TradeController) GetTrade(ctx echo.Context) error {
	tradeID := ctx.Param("id")
	if tradeID == "" {
		return ctx.JSON(http.StatusBadRequest, api.NewErrorResponse("Trade ID is required", "Missing trade ID"))
	}

	// Call business handler
	result := c.tradeHandler.GetTrade(ctx.Request().Context(), tradeID)
	if result.Error != nil {
		return ctx.JSON(http.StatusNotFound, api.NewErrorResponse(result.Error.Error(), result.Message))
	}

	// Return success response
	return ctx.JSON(http.StatusOK, api.NewSuccessResponse(result.Data, result.Message))
}

// GetTradesByUser handles GET /users/:user_id/trades
func (c *TradeController) GetTradesByUser(ctx echo.Context) error {
	userID := ctx.Param("user_id")
	if userID == "" {
		return ctx.JSON(http.StatusBadRequest, api.NewErrorResponse("User ID is required", "Missing user ID"))
	}

	// Get query parameters
	limitStr := ctx.QueryParam("limit")
	offsetStr := ctx.QueryParam("offset")
	
	limit := 50 // default limit
	offset := 0 // default offset
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Call business handler
	result := c.tradeHandler.GetTradesByUser(ctx.Request().Context(), userID)
	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewErrorResponse(result.Error.Error(), result.Message))
	}

	// Apply pagination (simplified)
	trades := result.Data.([]*engine.TradeOrder)
	if offset >= len(trades) {
		trades = []*engine.TradeOrder{}
	} else if offset+limit > len(trades) {
		trades = trades[offset:]
	} else {
		trades = trades[offset : offset+limit]
	}

	// Return success response
	return ctx.JSON(http.StatusOK, api.NewSuccessResponse(trades, result.Message))
}

// GetTradesByOrder handles GET /orders/:order_id/trades
func (c *TradeController) GetTradesByOrder(ctx echo.Context) error {
	orderID := ctx.Param("order_id")
	if orderID == "" {
		return ctx.JSON(http.StatusBadRequest, api.NewErrorResponse("Order ID is required", "Missing order ID"))
	}

	// Get query parameters
	limitStr := ctx.QueryParam("limit")
	offsetStr := ctx.QueryParam("offset")
	
	limit := 50 // default limit
	offset := 0 // default offset
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Call business handler
	result := c.tradeHandler.GetTradesByOrder(ctx.Request().Context(), orderID)
	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewErrorResponse(result.Error.Error(), result.Message))
	}

	// Apply pagination (simplified)
	trades := result.Data.([]*engine.TradeOrder)
	if offset >= len(trades) {
		trades = []*engine.TradeOrder{}
	} else if offset+limit > len(trades) {
		trades = trades[offset:]
	} else {
		trades = trades[offset : offset+limit]
	}

	// Return success response
	return ctx.JSON(http.StatusOK, api.NewSuccessResponse(trades, result.Message))
}
