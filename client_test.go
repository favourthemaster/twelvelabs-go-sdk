package twelvelabs

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	// Test case 1: Missing API Key
	_, err := NewClient(&ClientOptions{})
	if err == nil || err.Error() != "API key is required" {
		t.Errorf("Expected 'API key is required' error, got %v", err)
	}

	// Test case 2: Valid client creation
	client, err := NewClient(&ClientOptions{
		APIKey: "test-api-key",
	})
	if err != nil {
		t.Fatalf("Error creating client: %v", err)
	}

	if client.APIKey != "test-api-key" {
		t.Errorf("Expected APIKey to be 'test-api-key', got %s", client.APIKey)
	}
	if client.BaseURL != defaultBaseURL {
		t.Errorf("Expected BaseURL to be '%s', got %s", defaultBaseURL, client.BaseURL)
	}
	if client.HTTPClient.Timeout != defaultTimeout {
		t.Errorf("Expected HTTPClient timeout to be '%s', got %s", defaultTimeout, client.HTTPClient.Timeout)
	}

	// Test case 3: Custom options
	customClient, err := NewClient(&ClientOptions{
		APIKey:     "custom-api-key",
		BaseURL:    "http://custom.url",
		Timeout:    10 * time.Second,
		HTTPClient: &http.Client{},
	})
	if err != nil {
		t.Fatalf("Error creating custom client: %v", err)
	}

	if customClient.APIKey != "custom-api-key" {
		t.Errorf("Expected APIKey to be 'custom-api-key', got %s", customClient.APIKey)
	}
	if customClient.BaseURL != "http://custom.url" {
		t.Errorf("Expected BaseURL to be 'http://custom.url', got %s", customClient.BaseURL)
	}
	if customClient.HTTPClient.Timeout != 10*time.Second {
		t.Errorf("Expected HTTPClient timeout to be '10s', got %s", customClient.HTTPClient.Timeout)
	}
}

func TestNewRequest(t *testing.T) {
	client, _ := NewClient(&ClientOptions{APIKey: "test-api-key"})

	// Test case 1: GET request with no body
	req, err := client.newRequest("GET", "/test", nil)
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}

	if req.Method != "GET" {
		t.Errorf("Expected GET method, got %s", req.Method)
	}
	if req.URL.String() != defaultBaseURL+"/test" {
		t.Errorf("Expected URL %s, got %s", defaultBaseURL+"/test", req.URL.String())
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", req.Header.Get("Content-Type"))
	}
	if req.Header.Get("X-API-KEY") != "test-api-key" {
		t.Errorf("Expected X-API-KEY 'test-api-key', got %s", req.Header.Get("X-API-KEY"))
	}

	// Test case 2: POST request with JSON body
	body := map[string]string{"key": "value"}
	req, err = client.newRequest("POST", "/test", body)
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("Expected POST method, got %s", req.Method)
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got %s", req.Header.Get("Content-Type"))
	}

	// Test case 3: POST request with io.Reader body (e.g., file upload)
	fileContent := "test file content"
	readerBody := bytes.NewReader([]byte(fileContent))
	req, err = client.newRequest("POST", "/upload", readerBody)
	if err != nil {
		t.Fatalf("Error creating request: %v", err)
	}

	if req.Method != "POST" {
		t.Errorf("Expected POST method, got %s", req.Method)
	}
	// Content-Type should be application/octet-stream for io.Reader body
	if req.Header.Get("Content-Type") != "application/octet-stream" {
		t.Errorf("Expected Content-Type 'application/octet-stream', got %s", req.Header.Get("Content-Type"))
	}
}

func TestDo(t *testing.T) {
	// Mock server for testing HTTP requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/success":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"message": "success"}`)
		case "/error":
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"message": "bad request"}`)
		case "/unauthorized":
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"message": "unauthorized"}`)
		case "/notfound":
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message": "not found"}`)
		case "/toomanyrequests":
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprint(w, `{"message": "too many requests"}`)
		case "/internalservererror":
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"message": "internal server error"}`)
		default:
			w.WriteHeader(http.StatusTeapot)
			fmt.Fprint(w, `{"message": "unknown error"}`)
		}
	}))
	defer server.Close()

	client, _ := NewClient(&ClientOptions{
		APIKey:     "test-api-key",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	})

	// Test case 1: Successful request
	var successRes map[string]string
	req, _ := client.newRequest("GET", "/success", nil)
	_, err := client.do(req, &successRes)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if successRes["message"] != "success" {
		t.Errorf("Expected success message 'success', got %s", successRes["message"])
	}

	// Test case 2: Bad Request error
	req, _ = client.newRequest("GET", "/error", nil)
	_, err = client.do(req, nil)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if _, ok := err.(*BadRequestError); !ok {
		t.Errorf("Expected BadRequestError, got %T", err)
	}

	// Test case 3: Unauthorized error
	req, _ = client.newRequest("GET", "/unauthorized", nil)
	_, err = client.do(req, nil)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if _, ok := err.(*UnauthorizedError); !ok {
		t.Errorf("Expected UnauthorizedError, got %T", err)
	}

	// Test case 4: Not Found error
	req, _ = client.newRequest("GET", "/notfound", nil)
	_, err = client.do(req, nil)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if _, ok := err.(*NotFoundError); !ok {
		t.Errorf("Expected NotFoundError, got %T", err)
	}

	// Test case 5: Too Many Requests error
	req, _ = client.newRequest("GET", "/toomanyrequests", nil)
	_, err = client.do(req, nil)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if _, ok := err.(*TooManyRequestsError); !ok {
		t.Errorf("Expected TooManyRequestsError, got %T", err)
	}

	// Test case 6: Internal Server Error
	req, _ = client.newRequest("GET", "/internalservererror", nil)
	_, err = client.do(req, nil)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if _, ok := err.(*InternalServerError); !ok {
		t.Errorf("Expected InternalServerError, got %T", err)
	}

	// Test case 7: Unknown error
	req, _ = client.newRequest("GET", "/unknown", nil)
	_, err = client.do(req, nil)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if _, ok := err.(*APIError); !ok {
		t.Errorf("Expected APIError, got %T", err)
	}
}
