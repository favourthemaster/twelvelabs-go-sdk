package twelvelabs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// maxRetryDelay caps the exponential backoff between retries.
	maxRetryDelay = 10 * time.Second
	// baseRetryDelay is the delay before the first retry.
	baseRetryDelay = 500 * time.Millisecond
)

type transport struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	maxRetries int
	userAgent  string
}

type request struct {
	method    string
	path      string
	query     url.Values
	jsonBody  []byte
	multipart *multipartBody
	// accept overrides the Accept header. It defaults to application/json and
	// is set to the streaming media type for NDJSON and JSON Lines responses.
	accept string
	out    any
}

// multipartBody describes a streamed multipart/form-data request. The write
// function is called once per attempt; bodies built from a stream cannot be
// replayed, so such requests are never retried.
type multipartBody struct {
	write func(*multipart.Writer) error
}

func (t *transport) doJSON(ctx context.Context, method, path string, query url.Values, in, out any) error {
	req := &request{method: method, path: path, query: query, out: out}
	if in != nil {
		encoded, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("twelvelabs: encode request body: %w", err)
		}
		req.jsonBody = encoded
	}
	return t.do(ctx, req)
}

func (t *transport) doMultipart(ctx context.Context, method, path string, query url.Values, body *multipartBody, out any) error {
	return t.do(ctx, &request{method: method, path: path, query: query, multipart: body, out: out})
}

func (t *transport) do(ctx context.Context, req *request) error {
	attempts := t.maxRetries + 1
	if req.multipart != nil {
		// The body is consumed while it is sent, so it cannot be replayed.
		attempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			if err := sleepContext(ctx, retryDelay(attempt, lastErr)); err != nil {
				return err
			}
		}

		resp, err := t.roundTrip(ctx, req)
		if err == nil {
			err = decodeResponse(resp, req.out)
		}
		if err == nil {
			return nil
		}
		lastErr = err

		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if !shouldRetry(req.method, err) {
			return err
		}
	}
	return lastErr
}

func (t *transport) roundTrip(ctx context.Context, req *request) (*http.Response, error) {
	endpoint := t.baseURL + req.path
	if len(req.query) > 0 {
		endpoint += "?" + req.query.Encode()
	}

	var (
		body        io.Reader
		contentType string
		pipeReader  *io.PipeReader
	)

	switch {
	case req.multipart != nil:
		pr, pw := io.Pipe()
		pipeReader = pr
		mw := multipart.NewWriter(pw)
		contentType = mw.FormDataContentType()
		write := req.multipart.write
		go func() {
			err := write(mw)
			if err == nil {
				err = mw.Close()
			}
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			_ = pw.Close()
		}()
		body = pr
	case req.jsonBody != nil:
		body = bytes.NewReader(req.jsonBody)
		contentType = "application/json"
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.method, endpoint, body)
	if err != nil {
		if pipeReader != nil {
			_ = pipeReader.Close()
		}
		return nil, fmt.Errorf("twelvelabs: build request: %w", err)
	}

	accept := req.accept
	if accept == "" {
		accept = "application/json"
	}
	httpReq.Header.Set("x-api-key", t.apiKey)
	httpReq.Header.Set("User-Agent", t.userAgent)
	httpReq.Header.Set("Accept", accept)
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	if req.multipart != nil {
		// Force chunked encoding; the size is not known up front.
		httpReq.ContentLength = -1
	}

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("twelvelabs: %w", err)
	}
	return resp, nil
}

// stream performs a request and returns the response body for incremental
// decoding. The caller must close the returned reader.
func (t *transport) stream(ctx context.Context, req *request) (io.ReadCloser, error) {
	resp, err := t.roundTrip(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, newAPIError(resp, body)
	}
	return resp.Body, nil
}

func decodeResponse(resp *http.Response, out any) error {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("twelvelabs: read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return newAPIError(resp, body)
	}

	if out == nil || len(bytes.TrimSpace(body)) == 0 {
		return nil
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("twelvelabs: decode response body: %w", err)
	}
	return nil
}

func shouldRetry(method string, err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == http.StatusTooManyRequests {
			return true
		}
		if apiErr.StatusCode >= 500 && isIdempotent(method) {
			return true
		}
		return false
	}

	// Network failures are only safe to replay for idempotent methods.
	var netErr net.Error
	if errors.As(err, &netErr) {
		return isIdempotent(method)
	}
	return false
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

func retryDelay(attempt int, err error) time.Duration {
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
		return apiErr.RetryAfter
	}

	delay := baseRetryDelay << uint(attempt-1)
	if delay > maxRetryDelay || delay <= 0 {
		delay = maxRetryDelay
	}
	// Full jitter on the upper half of the interval.
	jitter := time.Duration(rand.Int63n(int64(delay)/2 + 1))
	return delay/2 + jitter
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// escapePath escapes a single path segment.
func escapePath(segment string) string { return url.PathEscape(segment) }

func encodeQuery(options *Pagination) url.Values {
	query := url.Values{}
	if options != nil {
		if options.Page > 0 {
			query.Set("page", strconv.Itoa(options.Page))
		}
		if options.PageLimit > 0 {
			query.Set("page_limit", strconv.Itoa(options.PageLimit))
		}
	}
	return query
}

func addStrings(query url.Values, key string, values []string) {
	for _, value := range values {
		if value != "" {
			query.Add(key, value)
		}
	}
}

func trimQuotes(s string) string { return strings.Trim(s, `"`) }
