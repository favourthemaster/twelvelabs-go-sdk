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

// ValidationError represents a validation error for invalid parameters
type ValidationError struct {
	APIError
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("Validation Error: %s", e.Message)
}

// ServiceError represents a service-level error
type ServiceError struct {
	APIError
	ServiceName string
}

func (e *ServiceError) Error() string {
	return fmt.Sprintf("%s Service Error: %s", e.ServiceName, e.Message)
}

// RequestError represents an error creating or processing a request
type RequestError struct {
	APIError
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("Request Error: %s", e.Message)
}

// TimeoutError represents a timeout error
type TimeoutError struct {
	APIError
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("Timeout Error: %s", e.Message)
}

// NewBadRequestError creates a new BadRequestError
func NewBadRequestError(message string) *BadRequestError {
	return &BadRequestError{
		APIError: APIError{
			StatusCode: 400,
			Message:    message,
		},
	}
}

// NewUnauthorizedError creates a new UnauthorizedError
func NewUnauthorizedError(message string) *UnauthorizedError {
	return &UnauthorizedError{
		APIError: APIError{
			StatusCode: 401,
			Message:    message,
		},
	}
}

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(message string) *NotFoundError {
	return &NotFoundError{
		APIError: APIError{
			StatusCode: 404,
			Message:    message,
		},
	}
}

// NewTooManyRequestsError creates a new TooManyRequestsError
func NewTooManyRequestsError(message string) *TooManyRequestsError {
	return &TooManyRequestsError{
		APIError: APIError{
			StatusCode: 429,
			Message:    message,
		},
	}
}

// NewInternalServerError creates a new InternalServerError
func NewInternalServerError(message string) *InternalServerError {
	return &InternalServerError{
		APIError: APIError{
			StatusCode: 500,
			Message:    message,
		},
	}
}

// NewValidationError creates a new ValidationError
func NewValidationError(message string) *ValidationError {
	return &ValidationError{
		APIError: APIError{
			StatusCode: 400,
			Message:    message,
		},
	}
}

// NewServiceError creates a new ServiceError
func NewServiceError(serviceName, message string) *ServiceError {
	return &ServiceError{
		APIError: APIError{
			StatusCode: 500,
			Message:    message,
		},
		ServiceName: serviceName,
	}
}

// NewRequestError creates a new RequestError
func NewRequestError(message string) *RequestError {
	return &RequestError{
		APIError: APIError{
			StatusCode: 400,
			Message:    message,
		},
	}
}

// NewTimeoutError creates a new TimeoutError
func NewTimeoutError(message string) *TimeoutError {
	return &TimeoutError{
		APIError: APIError{
			StatusCode: 408,
			Message:    message,
		},
	}
}
