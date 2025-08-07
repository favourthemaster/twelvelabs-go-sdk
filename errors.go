package twelvelabs

import "fmt"

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error: status %d: %s", e.StatusCode, e.Message)
}

type BadRequestError struct {
	APIError
}

type UnauthorizedError struct {
	APIError
}

type NotFoundError struct {
	APIError
}

type TooManyRequestsError struct {
	APIError
}

type InternalServerError struct {
	APIError
}
