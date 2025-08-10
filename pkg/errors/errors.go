package errors

import "fmt"

// APIError represents a generic API error
type APIError struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API Error %d: %s", e.StatusCode, e.Message)
}

// BadRequestError represents a 400 Bad Request error
type BadRequestError struct {
	APIError
}

func (e *BadRequestError) Error() string {
	return fmt.Sprintf("Bad Request (400): %s", e.Message)
}

// UnauthorizedError represents a 401 Unauthorized error
type UnauthorizedError struct {
	APIError
}

func (e *UnauthorizedError) Error() string {
	return fmt.Sprintf("Unauthorized (401): %s", e.Message)
}

// NotFoundError represents a 404 Not Found error
type NotFoundError struct {
	APIError
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("Not Found (404): %s", e.Message)
}

// TooManyRequestsError represents a 429 Too Many Requests error
type TooManyRequestsError struct {
	APIError
}

func (e *TooManyRequestsError) Error() string {
	return fmt.Sprintf("Too Many Requests (429): %s", e.Message)
}

// InternalServerError represents a 500 Internal Server Error
type InternalServerError struct {
	APIError
}

func (e *InternalServerError) Error() string {
	return fmt.Sprintf("Internal Server Error (500): %s", e.Message)
}
