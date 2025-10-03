package http

import (
	"net/http"
	"strconv"

	"github.com/EggysOnCode/anomi/api"
	"github.com/EggysOnCode/anomi/api/handlers"
	"github.com/EggysOnCode/anomi/core/orderbook"
	"github.com/labstack/echo/v4"
)

// ReceiptController handles HTTP requests for receipts
type ReceiptController struct {
	receiptHandler *handlers.ReceiptHandler
}

// NewReceiptController creates a new receipt controller
func NewReceiptController(receiptHandler *handlers.ReceiptHandler) *ReceiptController {
	return &ReceiptController{
		receiptHandler: receiptHandler,
	}
}

// GetReceipt handles GET /receipts/:id
func (c *ReceiptController) GetReceipt(ctx echo.Context) error {
	receiptID := ctx.Param("id")
	if receiptID == "" {
		return ctx.JSON(http.StatusBadRequest, api.NewErrorResponse("Receipt ID is required", "Missing receipt ID"))
	}

	// Call business handler
	result := c.receiptHandler.GetReceipt(ctx.Request().Context(), receiptID)
	if result.Error != nil {
		return ctx.JSON(http.StatusNotFound, api.NewErrorResponse(result.Error.Error(), result.Message))
	}

	// Return success response
	return ctx.JSON(http.StatusOK, api.NewSuccessResponse(result.Data, result.Message))
}

// GetReceiptsByUser handles GET /users/:user_id/receipts
func (c *ReceiptController) GetReceiptsByUser(ctx echo.Context) error {
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
	result := c.receiptHandler.GetReceiptsByUser(ctx.Request().Context(), userID)
	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewErrorResponse(result.Error.Error(), result.Message))
	}

	// Apply pagination (simplified)
	receipts := result.Data.([]*orderbook.Receipt)
	if offset >= len(receipts) {
		receipts = []*orderbook.Receipt{}
	} else if offset+limit > len(receipts) {
		receipts = receipts[offset:]
	} else {
		receipts = receipts[offset : offset+limit]
	}

	// Return success response
	return ctx.JSON(http.StatusOK, api.NewSuccessResponse(receipts, result.Message))
}

// GetReceiptsByOrder handles GET /orders/:order_id/receipts
func (c *ReceiptController) GetReceiptsByOrder(ctx echo.Context) error {
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
	result := c.receiptHandler.GetReceiptsByOrder(ctx.Request().Context(), orderID)
	if result.Error != nil {
		return ctx.JSON(http.StatusInternalServerError, api.NewErrorResponse(result.Error.Error(), result.Message))
	}

	// Apply pagination (simplified)
	receipts := result.Data.([]*orderbook.Receipt)
	if offset >= len(receipts) {
		receipts = []*orderbook.Receipt{}
	} else if offset+limit > len(receipts) {
		receipts = receipts[offset:]
	} else {
		receipts = receipts[offset : offset+limit]
	}

	// Return success response
	return ctx.JSON(http.StatusOK, api.NewSuccessResponse(receipts, result.Message))
}
