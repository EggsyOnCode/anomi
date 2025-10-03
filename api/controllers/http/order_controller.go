package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/EggysOnCode/anomi/api/handlers"
	"github.com/EggysOnCode/anomi/core/orderbook/engine"
	"github.com/labstack/echo/v4"
)

// OrderHandler interface for dependency injection
type OrderHandler interface {
	CreateOrder(ctx context.Context, order *engine.Order) interface{}
	UpdateOrder(ctx context.Context, order *engine.Order) interface{}
	CancelOrder(ctx context.Context, orderID string) interface{}
	GetOrder(ctx context.Context, orderID string) interface{}
	GetOrdersByUser(ctx context.Context, userID string) interface{}
}

// OrderController handles HTTP requests for orders
type OrderController struct {
	orderHandler OrderHandler
}

// NewOrderController creates a new order controller
func NewOrderController(orderHandler OrderHandler) *OrderController {
	return &OrderController{
		orderHandler: orderHandler,
	}
}

// CreateOrder handles POST /orders
func (c *OrderController) CreateOrder(ctx echo.Context) error {
	// Parse request body
	var req CreateOrderRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, NewErrorResponse("Invalid request body", "Failed to parse request"))
	}

	// Validate required fields
	if err := c.validateCreateOrderRequest(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, NewErrorResponse(err.Error(), "Validation failed"))
	}

	// Convert to engine.Order (simplified - needs proper implementation)
	order, err := c.createOrderFromRequest(&req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewErrorResponse(err.Error(), "Invalid order data"))
	}

	// Call business handler
	result := c.orderHandler.CreateOrder(ctx.Request().Context(), order)

	// Type assert to get the HandlerResult
	if handlerResult, ok := result.(*handlers.HandlerResult); ok {
		if handlerResult.Error != nil {
			return ctx.JSON(http.StatusInternalServerError, NewErrorResponse(handlerResult.Error.Error(), handlerResult.Message))
		}
		return ctx.JSON(http.StatusCreated, NewSuccessResponse(handlerResult.Data, handlerResult.Message))
	}

	return ctx.JSON(http.StatusInternalServerError, NewErrorResponse("Invalid handler result", "Handler returned unexpected result type"))
}

// GetOrder handles GET /orders/:id
func (c *OrderController) GetOrder(ctx echo.Context) error {
	orderID := ctx.Param("id")
	if orderID == "" {
		return ctx.JSON(http.StatusBadRequest, NewErrorResponse("Order ID is required", "Missing order ID"))
	}

	// Call business handler
	result := c.orderHandler.GetOrder(ctx.Request().Context(), orderID)

	// Type assert to get the HandlerResult
	if handlerResult, ok := result.(*handlers.HandlerResult); ok {
		if handlerResult.Error != nil {
			return ctx.JSON(http.StatusNotFound, NewErrorResponse(handlerResult.Error.Error(), handlerResult.Message))
		}
		return ctx.JSON(http.StatusOK, NewSuccessResponse(handlerResult.Data, handlerResult.Message))
	}

	return ctx.JSON(http.StatusInternalServerError, NewErrorResponse("Invalid handler result", "Handler returned unexpected result type"))
}

// UpdateOrder handles PUT /orders/:id
func (c *OrderController) UpdateOrder(ctx echo.Context) error {
	orderID := ctx.Param("id")
	if orderID == "" {
		return ctx.JSON(http.StatusBadRequest, NewErrorResponse("Order ID is required", "Missing order ID"))
	}

	// Parse request body
	var req UpdateOrderRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, NewErrorResponse("Invalid request body", "Failed to parse request"))
	}

	// Validate required fields
	if err := c.validateUpdateOrderRequest(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, NewErrorResponse(err.Error(), "Validation failed"))
	}

	// Convert to engine.Order (simplified - needs proper implementation)
	order, err := c.updateOrderFromRequest(orderID, &req)
	if err != nil {
		return ctx.JSON(http.StatusBadRequest, NewErrorResponse(err.Error(), "Invalid order data"))
	}

	// Call business handler
	result := c.orderHandler.UpdateOrder(ctx.Request().Context(), order)

	// Type assert to get the HandlerResult
	if handlerResult, ok := result.(*handlers.HandlerResult); ok {
		if handlerResult.Error != nil {
			return ctx.JSON(http.StatusInternalServerError, NewErrorResponse(handlerResult.Error.Error(), handlerResult.Message))
		}
		return ctx.JSON(http.StatusOK, NewSuccessResponse(handlerResult.Data, handlerResult.Message))
	}

	return ctx.JSON(http.StatusInternalServerError, NewErrorResponse("Invalid handler result", "Handler returned unexpected result type"))
}

// CancelOrder handles DELETE /orders/:id
func (c *OrderController) CancelOrder(ctx echo.Context) error {
	orderID := ctx.Param("id")
	if orderID == "" {
		return ctx.JSON(http.StatusBadRequest, NewErrorResponse("Order ID is required", "Missing order ID"))
	}

	// Call business handler
	result := c.orderHandler.CancelOrder(ctx.Request().Context(), orderID)

	// Type assert to get the HandlerResult
	if handlerResult, ok := result.(*handlers.HandlerResult); ok {
		if handlerResult.Error != nil {
			return ctx.JSON(http.StatusInternalServerError, NewErrorResponse(handlerResult.Error.Error(), handlerResult.Message))
		}
		return ctx.JSON(http.StatusOK, NewSuccessResponse(handlerResult.Data, handlerResult.Message))
	}

	return ctx.JSON(http.StatusInternalServerError, NewErrorResponse("Invalid handler result", "Handler returned unexpected result type"))
}

// GetOrdersByUser handles GET /users/:user_id/orders
func (c *OrderController) GetOrdersByUser(ctx echo.Context) error {
	userID := ctx.Param("user_id")
	if userID == "" {
		return ctx.JSON(http.StatusBadRequest, NewErrorResponse("User ID is required", "Missing user ID"))
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
	result := c.orderHandler.GetOrdersByUser(ctx.Request().Context(), userID)

	// Type assert to get the HandlerResult
	if handlerResult, ok := result.(*handlers.HandlerResult); ok {
		if handlerResult.Error != nil {
			return ctx.JSON(http.StatusInternalServerError, NewErrorResponse(handlerResult.Error.Error(), handlerResult.Message))
		}

		// Apply pagination (simplified)
		orders := handlerResult.Data.([]*engine.Order)
		if offset >= len(orders) {
			orders = []*engine.Order{}
		} else if offset+limit > len(orders) {
			orders = orders[offset:]
		} else {
			orders = orders[offset : offset+limit]
		}

		return ctx.JSON(http.StatusOK, NewSuccessResponse(orders, handlerResult.Message))
	}

	return ctx.JSON(http.StatusInternalServerError, NewErrorResponse("Invalid handler result", "Handler returned unexpected result type"))
}

// Validation methods

func (c *OrderController) validateCreateOrderRequest(req *CreateOrderRequest) error {
	if req.OrderType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Order type is required")
	}
	if req.UserID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "User ID is required")
	}
	if req.Quantity == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Quantity is required")
	}
	if req.Price == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Price is required")
	}
	return nil
}

func (c *OrderController) validateUpdateOrderRequest(req *UpdateOrderRequest) error {
	if req.Quantity == "" && req.Price == "" && req.Stop == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "At least one field must be provided for update")
	}
	return nil
}

// Conversion methods (simplified - needs proper implementation)

func (c *OrderController) createOrderFromRequest(req *CreateOrderRequest) (*engine.Order, error) {
	// This is a simplified conversion - you'll need to implement proper decimal parsing
	// and order creation based on your engine.Order constructors
	return nil, echo.NewHTTPError(http.StatusNotImplemented, "Order creation not yet implemented")
}

func (c *OrderController) updateOrderFromRequest(orderID string, req *UpdateOrderRequest) (*engine.Order, error) {
	// This is a simplified conversion - you'll need to implement proper order updates
	return nil, echo.NewHTTPError(http.StatusNotImplemented, "Order update not yet implemented")
}
