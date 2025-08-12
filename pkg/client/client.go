package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/errors"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/services"
)

const (
	DefaultBaseURL = "https://api.twelvelabs.io/v1.3"
	DefaultTimeout = 60 * time.Second
)

type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	APIKey     string
	Tasks      *services.TasksService
	Indexes    *services.IndexesService
	Embed      *services.EmbedService
	Search     *services.SearchService
	Analyze    *services.AnalyzeService
}

type Options struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

func NewClient(options *Options) *Client {
	if options.BaseURL == "" {
		options.BaseURL = DefaultBaseURL
	}

	httpClient := &http.Client{
		Timeout: options.Timeout,
	}

	client := &Client{
		HTTPClient: httpClient,
		BaseURL:    options.BaseURL,
		APIKey:     options.APIKey,
	}

	// Initialize services with client reference
	client.Tasks = &services.TasksService{Client: client}
	client.Indexes = &services.IndexesService{Client: client}
	client.Embed = &services.EmbedService{Client: client}
	client.Search = &services.SearchService{Client: client}
	client.Analyze = &services.AnalyzeService{Client: client}

	return client
}

func (c *Client) NewRequest(method, path string, body interface{}) (*http.Request, error) {
	var reqBody io.Reader
	contentType := "application/json"

	if body != nil {
		if r, ok := body.(io.Reader); ok {
			reqBody = r
			contentType = "application/octet-stream"
		} else {
			buf := new(bytes.Buffer)
			err := json.NewEncoder(buf).Encode(body)
			if err != nil {
				return nil, err
			}
			reqBody = buf
		}
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-API-KEY", c.APIKey)
	req.Header.Set("User-Agent", "twelvelabs-go-sdk/1.0.0")

	return req, nil
}

func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("failed to close response body: %v", err)
		}
	}(res.Body)

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Add debugging to see the raw JSON response
	fmt.Printf("DEBUG: Raw JSON response: %s\n", string(body))

	if res.StatusCode >= 400 {
		return nil, handleAPIError(res.StatusCode, body)
	}

	if v != nil && len(body) > 0 {
		if err := json.Unmarshal(body, v); err != nil {
			fmt.Printf("DEBUG: JSON unmarshal error: %v\n", err)
			fmt.Printf("DEBUG: Trying to unmarshal into type: %T\n", v)
			return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
		}
	}

	return res, nil
}

// DoRaw performs a raw HTTP request and returns the response without closing the body
// This is useful for streaming responses where the caller needs to handle the response body
func (c *Client) DoRaw(req *http.Request) (*http.Response, error) {
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode >= 400 {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Printf("failed to close error response body: %v", err)
			}
		}(res.Body)
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read error response body: %w", err)
		}
		return nil, handleAPIError(res.StatusCode, body)
	}

	return res, nil
}

func handleAPIError(statusCode int, body []byte) error {
	apiErr := &errors.APIError{
		StatusCode: statusCode,
	}

	var errBody struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errBody); err == nil {
		apiErr.Message = errBody.Message
	} else {
		apiErr.Message = fmt.Sprintf("HTTP %d", statusCode)
	}

	switch statusCode {
	case http.StatusBadRequest:
		return &errors.BadRequestError{APIError: *apiErr}
	case http.StatusUnauthorized:
		return &errors.UnauthorizedError{APIError: *apiErr}
	case http.StatusNotFound:
		return &errors.NotFoundError{APIError: *apiErr}
	case http.StatusTooManyRequests:
		return &errors.TooManyRequestsError{APIError: *apiErr}
	case http.StatusInternalServerError:
		return &errors.InternalServerError{APIError: *apiErr}
	default:
		return apiErr
	}
}
