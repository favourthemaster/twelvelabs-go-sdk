package twelvelabs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://api.twelvelabs.io"
	defaultTimeout = 60 * time.Second
)

type Client struct {
	HTTPClient *http.Client
	BaseURL    string
	APIKey     string

	Tasks        *TasksService
	Indexes      *IndexesService
	Search       *SearchService
	Embed        *EmbedService
	ManageVideos *ManageVideosService
}

type ClientOptions struct {
	HTTPClient *http.Client
	BaseURL    string
	APIKey     string
	Timeout    time.Duration
}

func NewClient(options *ClientOptions) (*Client, error) {
	if options == nil {
		options = &ClientOptions{}
	}

	if options.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	hc := options.HTTPClient
	if hc == nil {
		hc = &http.Client{}
	}

	if options.Timeout != 0 {
		hc.Timeout = options.Timeout
	} else if hc.Timeout == 0 {
		hc.Timeout = defaultTimeout
	}

	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	c := &Client{
		HTTPClient: hc,
		BaseURL:    baseURL,
		APIKey:     options.APIKey,
	}

	c.Tasks = &TasksService{client: c}
	c.Indexes = &IndexesService{client: c}
	c.Search = &SearchService{client: c}
	c.Embed = &EmbedService{client: c}
	c.ManageVideos = &ManageVideosService{client: c}

	return c, nil
}

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	var reqBody io.Reader
	contentType := "application/json"

	if body != nil {
		if r, ok := body.(io.Reader); ok {
			reqBody = r
			contentType = "application/octet-stream" // Or detect based on file type
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

func (c *Client) do(req *http.Request, v interface{}) (*http.Response, error) {
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		apiErr := &APIError{
			StatusCode: res.StatusCode,
		}
		// Try to decode the error message from the response body
		var errBody struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(res.Body).Decode(&errBody); err == nil {
			apiErr.Message = errBody.Message
		} else {
			apiErr.Message = res.Status
		}

		switch res.StatusCode {
		case http.StatusBadRequest:
			return nil, &BadRequestError{*apiErr}
		case http.StatusUnauthorized:
			return nil, &UnauthorizedError{*apiErr}
		case http.StatusNotFound:
			return nil, &NotFoundError{*apiErr}
		case http.StatusTooManyRequests:
			return nil, &TooManyRequestsError{*apiErr}
		case http.StatusInternalServerError:
			return nil, &InternalServerError{*apiErr}
		default:
			return nil, apiErr
		}
	}

	if v != nil {
		if err := json.NewDecoder(res.Body).Decode(v); err != nil && err != io.EOF {
			return nil, err
		}
	}

	return res, nil
}
