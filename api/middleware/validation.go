package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
)

// ValidationMiddleware provides request validation
func ValidationMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Validate request method
			if !isValidMethod(c.Request().Method) {
				return c.JSON(http.StatusMethodNotAllowed, map[string]string{
					"error": "Method not allowed",
				})
			}

			// Validate content type for POST/PUT requests
			if c.Request().Method == "POST" || c.Request().Method == "PUT" {
				contentType := c.Request().Header.Get("Content-Type")
				if contentType != "application/json" {
					return c.JSON(http.StatusBadRequest, map[string]string{
						"error": "Content-Type must be application/json",
					})
				}
			}

			return next(c)
		}
	}
}

// JSONBindingMiddleware provides JSON binding with validation
func JSONBindingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Only bind JSON for POST/PUT requests
			if c.Request().Method == "POST" || c.Request().Method == "PUT" {
				// Validate JSON syntax
				var jsonData map[string]interface{}
				if err := c.Bind(&jsonData); err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{
						"error": "Invalid JSON format",
					})
				}
			}

			return next(c)
		}
	}
}

// isValidMethod checks if the HTTP method is allowed
func isValidMethod(method string) bool {
	allowedMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	for _, m := range allowedMethods {
		if method == m {
			return true
		}
	}
	return false
}

// ValidateJSON validates JSON data
func ValidateJSON(data []byte) error {
	var jsonData interface{}
	return json.Unmarshal(data, &jsonData)
}

// ValidateRequiredFields validates that required fields are present
func ValidateRequiredFields(data map[string]interface{}, requiredFields []string) error {
	for _, field := range requiredFields {
		if _, exists := data[field]; !exists {
			return echo.NewHTTPError(http.StatusBadRequest, "Missing required field: "+field)
		}
	}
	return nil
}
