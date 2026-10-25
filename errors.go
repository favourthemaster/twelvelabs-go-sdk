package twelvelabs

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIError is returned for every non-2xx response from the TwelveLabs API.
//
// The platform returns errors as {"error": {"code", "message", "details"}}.
// Some legacy endpoints return a flat {"code", "message"} object; both shapes
// are decoded.
type APIError struct {
	// StatusCode is the HTTP status code of the response.
	StatusCode int
	// Code is the machine-readable error code, for example "parameter_invalid".
	Code string
	// Message is the human-readable error message.
	Message string
	// Details carries any additional structured error data.
	Details map[string]any
	// RequestID is the value of the x-request-id response header, when present.
	RequestID string
	// RetryAfter is the delay requested by the server for 429 responses.
	RetryAfter time.Duration
	// RateLimitDimensions is the value of the X-Ratelimit-Dimensions header.
	RateLimitDimensions string
	// Body is the raw response body.
	Body []byte
}

// Error implements the error interface.
func (e *APIError) Error() string {
	var b strings.Builder
	b.WriteString("twelvelabs: ")
	if e.StatusCode != 0 {
		fmt.Fprintf(&b, "HTTP %d", e.StatusCode)
	}
	if e.Code != "" {
		b.WriteString(" ")
		b.WriteString(e.Code)
	}
	if e.Message != "" {
		b.WriteString(": ")
		b.WriteString(e.Message)
	}
	if e.RequestID != "" {
		fmt.Fprintf(&b, " (request id %s)", e.RequestID)
	}
	return b.String()
}

// ValidationError is returned when a request is rejected locally, before it is
// sent to the API.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return "twelvelabs: " + e.Message
	}
	return fmt.Sprintf("twelvelabs: %s: %s", e.Field, e.Message)
}

func validationError(field, format string, args ...any) error {
	return &ValidationError{Field: field, Message: fmt.Sprintf(format, args...)}
}

var (
	// ErrMissingAPIKey is returned by NewClient when no API key is configured.
	ErrMissingAPIKey = errors.New("twelvelabs: an API key is required; set Options.APIKey or the TWELVE_LABS_API_KEY environment variable")

	// ErrTimeout is returned by wait helpers when a timeout is exceeded.
	ErrTimeout = errors.New("twelvelabs: timed out waiting for the resource to become ready")

	// ErrInvalidWebhookSignature is returned when a webhook payload does not
	// match its signature.
	ErrInvalidWebhookSignature = errors.New("twelvelabs: invalid webhook signature")

	// ErrWebhookTimestampOutsideTolerance is returned when a webhook
	// signature's timestamp is too far from the current time.
	ErrWebhookTimestampOutsideTolerance = errors.New("twelvelabs: webhook timestamp is outside the allowed tolerance")

	// ErrMalformedWebhookSignature is returned when the TL-Signature header
	// cannot be parsed.
	ErrMalformedWebhookSignature = errors.New("twelvelabs: malformed TL-Signature header")

	// ErrMissingWebhookSecret is returned when a verifier has no secret.
	ErrMissingWebhookSecret = errors.New("twelvelabs: a webhook signing secret is required")

	// ErrResponseTooLarge is returned when a non-streaming response exceeds the
	// size the SDK is willing to buffer. It protects callers from unbounded
	// memory use if a proxy or server returns an unexpected payload.
	ErrResponseTooLarge = errors.New("twelvelabs: response body exceeds the maximum allowed size")
)

func errEmptyID(resource string) error {
	return fmt.Errorf("twelvelabs: the API did not return an ID for the created %s", resource)
}

func errTaskFailed(resource, id string, cause *ErrorBody) error {
	if cause == nil || cause.String() == "" {
		return fmt.Errorf("twelvelabs: %s %s failed", resource, id)
	}
	return fmt.Errorf("twelvelabs: %s %s failed: %s", resource, id, cause)
}

// IsNotFound reports whether err is an API error with status 404.
func IsNotFound(err error) bool { return statusIs(err, http.StatusNotFound) }

// IsUnauthorized reports whether err is an API error with status 401 or 403.
func IsUnauthorized(err error) bool {
	return statusIs(err, http.StatusUnauthorized) || statusIs(err, http.StatusForbidden)
}

// IsBadRequest reports whether err is an API error with status 400 or 422.
func IsBadRequest(err error) bool {
	return statusIs(err, http.StatusBadRequest) || statusIs(err, http.StatusUnprocessableEntity)
}

// IsRateLimited reports whether err is an API error with status 429.
func IsRateLimited(err error) bool { return statusIs(err, http.StatusTooManyRequests) }

// IsServerError reports whether err is an API error with a 5xx status.
func IsServerError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode >= 500
}

// AsAPIError returns the *APIError wrapped by err, if any.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	ok := errors.As(err, &apiErr)
	return apiErr, ok
}

func statusIs(err error, code int) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == code
}

func newAPIError(resp *http.Response, body []byte) *APIError {
	apiErr := &APIError{
		StatusCode:          resp.StatusCode,
		RequestID:           resp.Header.Get("x-request-id"),
		RateLimitDimensions: resp.Header.Get("X-Ratelimit-Dimensions"),
		Body:                body,
	}

	var envelope struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		switch {
		case envelope.Error.Code != "" || envelope.Error.Message != "":
			apiErr.Code = envelope.Error.Code
			apiErr.Message = envelope.Error.Message
			apiErr.Details = envelope.Error.Details
		case envelope.Code != "" || envelope.Message != "":
			apiErr.Code = envelope.Code
			apiErr.Message = envelope.Message
			apiErr.Details = envelope.Details
		}
	}

	if apiErr.Message == "" {
		apiErr.Message = strings.TrimSpace(string(body))
	}
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}

	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			apiErr.RetryAfter = time.Duration(seconds) * time.Second
		} else if when, err := http.ParseTime(retryAfter); err == nil {
			apiErr.RetryAfter = time.Until(when)
		}
	}

	return apiErr
}
