package twelvelabs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testAPIKey = "test-api-key"

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := NewClient(Options{
		APIKey:     testAPIKey,
		BaseURL:    server.URL,
		MaxRetries: NoRetries,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		if err := json.NewEncoder(w).Encode(body); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}
}

func TestNewClientRequiresAPIKey(t *testing.T) {
	t.Setenv("TWELVE_LABS_API_KEY", "")
	if _, err := NewClient(Options{}); !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("expected ErrMissingAPIKey, got %v", err)
	}
}

func TestNewClientReadsEnvironment(t *testing.T) {
	t.Setenv("TWELVE_LABS_API_KEY", "from-env")
	client, err := NewClient(Options{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.BaseURL() != DefaultBaseURL {
		t.Fatalf("unexpected base URL %q", client.BaseURL())
	}
}

func TestRequestHeaders(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != testAPIKey {
			t.Errorf("x-api-key = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "twelvelabs-go-sdk/") {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"_id": "idx_1", "index_name": "demo"})
	})

	index, err := client.Indexes.Get(context.Background(), "idx_1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if index.ID != "idx_1" {
		t.Fatalf("unexpected index %+v", index)
	}
}

func TestErrorDecoding(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-request-id", "req_123")
		w.Header().Set("Retry-After", "3")
		writeJSON(t, w, http.StatusNotFound, map[string]any{
			"error": map[string]any{
				"code":    "resource_not_found",
				"message": "asset not found",
				"details": map[string]any{"asset_id": "a1"},
			},
		})
	})

	_, err := client.Assets.Get(context.Background(), "a1")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !IsNotFound(err) {
		t.Fatalf("expected IsNotFound, got %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.RequestID != "req_123" {
		t.Errorf("RequestID = %q", apiErr.RequestID)
	}
	if apiErr.RetryAfter != 3*time.Second {
		t.Errorf("RetryAfter = %v", apiErr.RetryAfter)
	}
	if apiErr.Details["asset_id"] != "a1" {
		t.Errorf("Details = %v", apiErr.Details)
	}
	if !strings.Contains(err.Error(), "asset not found") {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestRetriesRateLimitedRequests(t *testing.T) {
	var attempts int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			writeJSON(t, w, http.StatusTooManyRequests, map[string]any{
				"error": map[string]any{"code": "rate_limit_exceeded", "message": "slow down"},
			})
			return
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"_id": "idx_1"})
	})
	client.transport.maxRetries = 1

	if _, err := client.Indexes.Get(context.Background(), "idx_1"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestValidationDoesNotHitTheNetwork(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected request")
	})
	if _, err := client.Search.Query(context.Background(), &SearchRequest{}); err == nil {
		t.Fatal("expected a validation error")
	}
	if _, err := client.Indexes.Create(context.Background(), &IndexCreateRequest{}); err == nil {
		t.Fatal("expected a validation error")
	}
	if _, err := client.Analyze.Analyze(context.Background(), &AnalyzeRequest{}); err == nil {
		t.Fatal("expected a validation error")
	}
}

func TestIndexesCreateDefaultsModel(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/indexes" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body IndexCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if len(body.Models) != 1 || body.Models[0].ModelName != ModelMarengo30 {
			t.Errorf("models = %+v", body.Models)
		}
		writeJSON(t, w, http.StatusCreated, map[string]any{"_id": "idx_1"})
	})

	index, err := client.Indexes.Create(context.Background(), &IndexCreateRequest{IndexName: "demo"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if index.ID != "idx_1" || index.IndexName != "demo" {
		t.Fatalf("unexpected index %+v", index)
	}
}

func TestIndexedAssetsCreateAndWait(t *testing.T) {
	var polls int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/indexes/idx_1/indexed-assets":
			var body IndexedAssetCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
			}
			if body.AssetID != "asset_1" {
				t.Errorf("asset_id = %q", body.AssetID)
			}
			writeJSON(t, w, http.StatusAccepted, map[string]any{"_id": "ia_1", "asset_id": "asset_1"})
		case r.Method == http.MethodGet && r.URL.Path == "/indexes/idx_1/indexed-assets/ia_1":
			status := IndexedAssetStatusIndexing
			if atomic.AddInt32(&polls, 1) > 1 {
				status = IndexedAssetStatusReady
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"_id":      "ia_1",
				"asset_id": "asset_1",
				"status":   status,
			})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})

	created, err := client.IndexedAssets.Create(context.Background(), "idx_1", &IndexedAssetCreateRequest{AssetID: "asset_1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID != "ia_1" {
		t.Fatalf("unexpected indexed asset %+v", created)
	}

	ready, err := client.IndexedAssets.WaitForReady(context.Background(), "idx_1", "ia_1", &WaitOptions{PollInterval: time.Millisecond})
	if err != nil {
		t.Fatalf("WaitForReady: %v", err)
	}
	if ready.Status != IndexedAssetStatusReady {
		t.Fatalf("status = %q", ready.Status)
	}
}

func TestSearchSendsMultipartFields(t *testing.T) {
	var seen map[string][]string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "multipart/form-data" {
			t.Fatalf("content type = %q (%v)", r.Header.Get("Content-Type"), err)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if _, _, err := r.FormFile("query_media_file"); err != nil {
			t.Errorf("query_media_file missing: %v", err)
		}
		seen = r.MultipartForm.Value
		if params["boundary"] == "" {
			t.Error("missing boundary")
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"data": []map[string]any{{
				"start": 0.0, "end": 4.5, "video_id": "ia_1", "rank": 1,
			}},
			"page_info": map[string]any{
				"limit_per_page":  10,
				"total_results":   1,
				"next_page_token": "tok_2",
			},
		})
	})

	response, err := client.Search.Query(context.Background(), &SearchRequest{
		IndexID:         "idx_1",
		QueryText:       "a red car",
		SearchOptions:   []string{ModalityVisual, ModalityTranscription},
		QueryMediaType:  SearchMediaTypeImage,
		QueryMediaFiles: []*FileUpload{{Filename: "frame.jpg", Content: strings.NewReader("jpeg-bytes")}},
		GroupBy:         GroupByVideo,
		Operator:        OperatorAnd,
		PageLimit:       25,
		Filter:          map[string]any{"duration": map[string]any{"gte": 10}},
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].End != 4.5 {
		t.Fatalf("unexpected response %+v", response.Data)
	}
	if response.PageInfo.NextPageToken != "tok_2" {
		t.Fatalf("page info = %+v", response.PageInfo)
	}

	checks := map[string]string{
		"index_id":         "idx_1",
		"query_text":       "a red car",
		"query_media_type": SearchMediaTypeImage,
		"group_by":         GroupByVideo,
		"operator":         OperatorAnd,
		"page_limit":       "25",
		"filter":           `{"duration":{"gte":10}}`,
	}
	for key, want := range checks {
		if got := seen[key]; len(got) != 1 || got[0] != want {
			t.Errorf("%s = %v, want %q", key, got, want)
		}
	}
	if got := seen["search_options"]; len(got) != 2 {
		t.Errorf("search_options = %v", got)
	}
}

func TestSearchValidation(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected request")
	})
	cases := []*SearchRequest{
		{},
		{IndexID: "idx_1"},
		{IndexID: "idx_1", SearchOptions: []string{ModalityVisual}},
		{IndexID: "idx_1", SearchOptions: []string{ModalityVisual}, QueryText: "x", QueryMediaType: "video", QueryMediaURLs: []string{"https://example.com/a.jpg"}},
	}
	for i, request := range cases {
		if _, err := client.Search.Query(context.Background(), request); err == nil {
			t.Errorf("case %d: expected a validation error", i)
		}
	}
}

func TestAnalyzeStream(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if stream, _ := body["stream"].(bool); !stream {
			t.Errorf("stream = %v, want true", body["stream"])
		}
		if body["model_name"] != ModelPegasus15 {
			t.Errorf("model_name = %v", body["model_name"])
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(http.StatusOK)
		lines := []string{
			`{"event_type":"stream_start","metadata":{"generation_id":"gen_1"}}`,
			`{"event_type":"text_generation","text":"Hello "}`,
			`{"event_type":"text_generation","text":"world"}`,
			`{"event_type":"stream_end","finish_reason":"stop","metadata":{"generation_id":"gen_1","usage":{"output_tokens":2,"input_tokens":5}}}`,
		}
		for _, line := range lines {
			fmt.Fprintln(w, line)
		}
	})

	var text strings.Builder
	var events int
	err := client.Analyze.Stream(context.Background(), &AnalyzeRequest{
		Video:  VideoFromAssetID("asset_1"),
		Prompt: "Describe the video.",
	}, func(event AnalyzeStreamEvent) error {
		events++
		if event.EventType == StreamEventText {
			text.WriteString(event.Text)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if events != 4 {
		t.Fatalf("events = %d", events)
	}
	if text.String() != "Hello world" {
		t.Fatalf("text = %q", text.String())
	}
}

func TestAnalyzeStreamTruncationIsAWarning(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"event_type":"stream_end","finish_reason":"length","error":{"code":"","message":"truncated"}}`)
	})

	var final AnalyzeStreamEvent
	err := client.Analyze.Stream(context.Background(), &AnalyzeRequest{Video: VideoFromURL("https://example.com/a.mp4"), Prompt: "x"}, func(event AnalyzeStreamEvent) error {
		final = event
		return nil
	})
	if err != nil {
		t.Fatalf("Stream returned %v, want nil for a truncation warning", err)
	}
	if final.FinishReason != "length" || final.Error == nil || final.Error.Message != "truncated" {
		t.Fatalf("unexpected final event %+v", final)
	}
}

func TestAnalyzeSyncSendsStreamFalse(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if stream, ok := body["stream"]; !ok || stream != false {
			t.Errorf("stream = %v, want false", body["stream"])
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"id":            "gen_1",
			"data":          "A cat.",
			"finish_reason": "stop",
			"usage":         map[string]any{"output_tokens": 3, "input_tokens": 10},
		})
	})

	response, err := client.Analyze.Analyze(context.Background(), &AnalyzeRequest{
		Video:  VideoFromURL("https://example.com/cat.mp4"),
		Prompt: "What is in the video?",
	})
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if response.Data != "A cat." || response.Usage.OutputTokens != 3 {
		t.Fatalf("unexpected response %+v", response)
	}
}

func TestAnalyzeTaskMarshalUnlimitedTokens(t *testing.T) {
	limited, err := json.Marshal(AnalyzeTaskCreateRequest{MaxTokens: 4096, Video: VideoFromAssetID("a1")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(limited), `"max_tokens":4096`) {
		t.Fatalf("limited = %s", limited)
	}
	if strings.Contains(string(limited), "UnlimitedTokens") {
		t.Fatalf("unexpected field in %s", limited)
	}

	unlimited, err := json.Marshal(AnalyzeTaskCreateRequest{UnlimitedTokens: true, Video: VideoFromAssetID("a1")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(unlimited), `"max_tokens":"unlimited"`) {
		t.Fatalf("unlimited = %s", unlimited)
	}
}

func TestAnalyzeBatchResultsJSONL(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analyze/batches/batch_1/results" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if accept := r.Header.Get("Accept"); accept != "application/jsonl, application/json" {
			t.Errorf("Accept = %q", accept)
		}
		w.Header().Set("Content-Type", "application/jsonl")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"task_id":"t1","custom_id":"c1","status":"ready","data":{"generation_id":"g1","data":"ok","finish_reason":"stop","usage":{"output_tokens":1}}}`)
		fmt.Fprintln(w, `{"task_id":"t2","custom_id":"c2","status":"failed","error":{"code":"asset_unavailable","message":"nope"}}`)
	})

	results, err := client.Analyze.BatchResults(context.Background(), "batch_1")
	if err != nil {
		t.Fatalf("BatchResults: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d", len(results))
	}
	if results[0].Data == nil || results[0].Data.Data != "ok" {
		t.Fatalf("first result = %+v", results[0])
	}
	if results[1].Error == nil || results[1].Error.Message != "nope" {
		t.Fatalf("second result = %+v", results[1])
	}
}

func TestIncompleteUploadsPagination(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("page = %q", got)
		}
		if got := r.URL.Query().Get("page_limit"); got != "5" {
			t.Errorf("page_limit = %q", got)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"data":      []map[string]any{{"upload_id": "up_1", "status": UploadStatusActive}},
			"page_info": map[string]any{"page": 2, "limit_per_page": 5, "total_results": 1},
		})
	})

	page, err := client.Assets.ListIncompleteUploads(context.Background(), &Pagination{Page: 2, PageLimit: 5})
	if err != nil {
		t.Fatalf("ListIncompleteUploads: %v", err)
	}
	if len(page.Data) != 1 || page.PageInfo.Page != 2 {
		t.Fatalf("unexpected page %+v", page)
	}
}

func TestAssetUploadChunks(t *testing.T) {
	putBodies := map[string]string{}
	var putMu sync.Mutex
	var reports []CompletedChunk

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/assets/multipart-uploads":
			var body CreateAssetUploadRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			if body.TotalSize != 8 || body.Type != "video" {
				t.Errorf("create body = %+v", body)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{
				"upload_id":      "up_1",
				"asset_id":       "asset_1",
				"chunk_size":     4,
				"total_chunks":   2,
				"upload_headers": map[string]string{"x-amz-meta-token": "abc"},
				"upload_urls": []map[string]any{
					{"chunk_index": 1, "url": serverURL(r) + "/upload/1"},
					{"chunk_index": 2, "url": serverURL(r) + "/upload/2"},
				},
			})
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/upload/"):
			if got := r.Header.Get("x-amz-meta-token"); got != "abc" {
				t.Errorf("upload header = %q", got)
			}
			body, _ := io.ReadAll(r.Body)
			putMu.Lock()
			putBodies[r.URL.Path] = string(body)
			putMu.Unlock()
			w.Header().Set("ETag", `"etag-`+strings.TrimPrefix(r.URL.Path, "/upload/")+`"`)
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/assets/multipart-uploads/up_1":
			var body ReportChunksRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode report body: %v", err)
			}
			reports = body.CompletedChunks
			writeJSON(t, w, http.StatusOK, map[string]any{"asset_id": "asset_1"})
		case r.Method == http.MethodGet && r.URL.Path == "/assets/asset_1":
			writeJSON(t, w, http.StatusOK, map[string]any{"_id": "asset_1", "status": AssetStatusReady})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})

	asset, err := client.Assets.Upload(context.Background(), &MultipartUploadRequest{
		File: &FileUpload{Filename: "clip.mp4", Content: bytes.NewReader([]byte("abcdefgh"))},
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if asset.ID != "asset_1" || asset.Status != AssetStatusReady {
		t.Fatalf("unexpected asset %+v", asset)
	}
	putMu.Lock()
	defer putMu.Unlock()
	if len(putBodies) != 2 || putBodies["/upload/1"] != "abcd" || putBodies["/upload/2"] != "efgh" {
		t.Fatalf("chunk bodies = %q", putBodies)
	}
	if len(reports) != 2 {
		t.Fatalf("reports = %+v", reports)
	}
	proofs := map[int]string{}
	for _, report := range reports {
		proofs[report.ChunkIndex] = report.Proof
	}
	if proofs[1] != "etag-1" || proofs[2] != "etag-2" {
		t.Fatalf("proofs = %v", proofs)
	}
}

func TestAssetUploadReportsShortContent(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/assets/multipart-uploads":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"upload_id":    "up_1",
				"asset_id":     "asset_1",
				"chunk_size":   4,
				"total_chunks": 2,
				"upload_urls": []map[string]any{
					{"chunk_index": 1, "url": serverURL(r) + "/upload/1"},
					{"chunk_index": 2, "url": serverURL(r) + "/upload/2"},
				},
			})
		case r.Method == http.MethodPut && (r.URL.Path == "/upload/1" || r.URL.Path == "/upload/2"):
			w.Header().Set("ETag", `"etag"`)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})

	_, err := client.Assets.Upload(context.Background(), &MultipartUploadRequest{
		File:      &FileUpload{Filename: "clip.mp4", Content: strings.NewReader("abcd")},
		TotalSize: 8,
	})
	if err == nil || !strings.Contains(err.Error(), "check TotalSize") {
		t.Fatalf("expected a short-content error, got %v", err)
	}
}

func serverURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func TestFileUploadInfersContentType(t *testing.T) {
	file := &FileUpload{Filename: "clip.mp4", Content: strings.NewReader("x")}
	if got := file.contentType(); got != "video/mp4" {
		t.Fatalf("contentType = %q", got)
	}
	file.Filename = "unknown.bin"
	if got := file.contentType(); got != "application/octet-stream" {
		t.Fatalf("contentType = %q", got)
	}
}

func TestNewFileUpload(t *testing.T) {
	path := t.TempDir() + "/sample.txt"
	if err := os.WriteFile(path, []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := NewFileUpload(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if file.Filename != "sample.txt" {
		t.Fatalf("filename = %q", file.Filename)
	}
	if got := file.contentType(); got != "text/plain; charset=utf-8" {
		t.Fatalf("contentType = %q", got)
	}
}

func TestParseAnalyzeSegments(t *testing.T) {
	segments, err := ParseAnalyzeSegments(`{"highlight":[{"start_time":1.5,"end_time":4,"metadata":{"team":"red"}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(segments["highlight"]) != 1 {
		t.Fatalf("segments = %+v", segments)
	}
	if segments["highlight"][0].Metadata["team"] != "red" {
		t.Fatalf("metadata = %+v", segments["highlight"][0].Metadata)
	}
}

func TestParseAnalyzeSegmentsEmpty(t *testing.T) {
	segments, err := ParseAnalyzeSegments("  ")
	if err != nil || segments != nil {
		t.Fatalf("segments = %+v, err = %v", segments, err)
	}
}

func TestWebhookSignatureRoundTrip(t *testing.T) {
	payload := []byte(`{"id":"whe_1","created_at":"2026-09-14T10:29:30.123Z","type":"index.task.ready","data":{"id":"t1","metadata":{"duration":30},"status":"ready","models":[{"name":"marengo3.0","options":["visual","audio"]}],"tags":["demo"]}}`)
	now := time.Now()
	header := SignWebhookPayload("secret", now, payload)

	verifier := NewWebhookVerifier("secret")
	event, err := verifier.VerifyAndParse(payload, header)
	if err != nil {
		t.Fatalf("VerifyAndParse: %v", err)
	}
	if event.Type != WebhookEventIndexTaskReady {
		t.Fatalf("event type = %q", event.Type)
	}
	data, err := event.TaskData()
	if err != nil {
		t.Fatal(err)
	}
	if data.ID != "t1" || data.Status != "ready" {
		t.Fatalf("task data = %+v", data)
	}
	if data.Metadata == nil || data.Metadata.Duration != 30 {
		t.Fatalf("metadata = %+v", data.Metadata)
	}
	if len(data.Models) != 1 || data.Models[0].Name != "marengo3.0" {
		t.Fatalf("models = %+v", data.Models)
	}
}

func TestWebhookAnalyzeTaskData(t *testing.T) {
	event, err := ParseWebhookEvent([]byte(`{
		"id": "whe_2",
		"created_at": "2026-09-14T10:29:30.123Z",
		"type": "analyze.task.failed",
		"data": {
			"id": "t2",
			"custom_id": null,
			"status": "failed",
			"created_at": "2026-09-14T10:29:19.968Z",
			"video_source": {"type": "url", "url": "https://example.com/v.mp4", "system_metadata": {"duration": 120.5}},
			"error": {"message": "boom"}
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	data, err := event.TaskData()
	if err != nil {
		t.Fatal(err)
	}
	if data.ID != "t2" || data.CustomID != nil || data.Status != "failed" {
		t.Fatalf("task data = %+v", data)
	}
	if data.VideoSource == nil || data.VideoSource.Type != "url" || data.VideoSource.URL == "" {
		t.Fatalf("video source = %+v", data.VideoSource)
	}
	if data.Error == nil || data.Error.Message != "boom" {
		t.Fatalf("error = %+v", data.Error)
	}
}

func TestWebhookRejectsTamperedPayload(t *testing.T) {
	payload := []byte(`{"type":"index.task.ready"}`)
	header := SignWebhookPayload("secret", time.Now(), payload)

	verifier := NewWebhookVerifier("secret")
	err := verifier.Verify([]byte(`{"type":"index.task.failed"}`), header)
	if !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Fatalf("expected ErrInvalidWebhookSignature, got %v", err)
	}
}

func TestWebhookRejectsStaleSignature(t *testing.T) {
	payload := []byte(`{"type":"index.task.ready"}`)
	header := SignWebhookPayload("secret", time.Now().Add(-10*time.Minute), payload)

	err := NewWebhookVerifier("secret").Verify(payload, header)
	if !errors.Is(err, ErrWebhookTimestampOutsideTolerance) {
		t.Fatalf("expected a tolerance error, got %v", err)
	}
}

func TestWebhookAcceptsMultipleSignatures(t *testing.T) {
	payload := []byte(`{"type":"analyze.task.ready"}`)
	valid := SignWebhookPayload("new-secret", time.Now(), payload)
	header := strings.Replace(valid, "v1=", "v1=deadbeef,v1=", 1)

	if err := NewWebhookVerifier("new-secret").Verify(payload, header); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestWebhookMalformedHeader(t *testing.T) {
	cases := []string{"", "garbage", "t=abc,v1=ff", "t=123", "v1=ff"}
	for _, header := range cases {
		if err := NewWebhookVerifier("s").Verify([]byte("{}"), header); !errors.Is(err, ErrMalformedWebhookSignature) {
			t.Errorf("header %q: expected a malformed error, got %v", header, err)
		}
	}
}

func TestIndexedAssetsListFiltersAndGetOptions(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/indexes/idx_1/indexed-assets":
			query := r.URL.Query()
			if got := query.Get("duration"); got != `{"gte":10,"lte":20}` {
				t.Errorf("duration = %q", got)
			}
			if got := query.Get("width"); got != `{"gte":640}` {
				t.Errorf("width = %q", got)
			}
			if got := query.Get("status"); got != "ready" {
				t.Errorf("status = %q", got)
			}
			if got := query.Get("category"); got != "recentlyAdded" {
				t.Errorf("category = %q", got)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{"data": []any{}, "page_info": map[string]any{}})
		case "/indexes/idx_1/indexed-assets/ia_1":
			query := r.URL.Query()
			if got := query["embedding_option"]; len(got) != 2 || got[0] != EmbeddingOptionVisual || got[1] != EmbeddingOptionAudio {
				t.Errorf("embedding_option = %v", got)
			}
			if got := query.Get("transcription"); got != "true" {
				t.Errorf("transcription = %q", got)
			}
			writeJSON(t, w, http.StatusOK, map[string]any{"_id": "ia_1", "asset_id": "asset_1", "status": "ready"})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})

	if _, err := client.IndexedAssets.List(context.Background(), "idx_1", &IndexedAssetListOptions{
		Status:       []string{IndexedAssetStatusReady},
		Duration:     Range[float64]{GTE: ptrFloat(10), LTE: ptrFloat(20)},
		Width:        Range[int]{GTE: ptrInt(640)},
		UserMetadata: map[string]any{"category": "recentlyAdded", "inStock": true},
	}); err != nil {
		t.Fatalf("List: %v", err)
	}

	transcription := true
	if _, err := client.IndexedAssets.Get(context.Background(), "idx_1", "ia_1", &IndexedAssetGetOptions{
		EmbeddingOptions: []string{EmbeddingOptionVisual, EmbeddingOptionAudio},
		Transcription:    &transcription,
	}); err != nil {
		t.Fatalf("Get: %v", err)
	}
}

func TestIndexedAssetValidation(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected request")
	})
	if _, err := client.IndexedAssets.List(context.Background(), "idx_1", &IndexedAssetListOptions{
		Duration: Range[float64]{GTE: ptrFloat(20), LTE: ptrFloat(10)},
	}); err == nil {
		t.Error("expected an error for an inverted range")
	}
	if _, err := client.IndexedAssets.List(context.Background(), "idx_1", &IndexedAssetListOptions{
		Status: []string{"unknown"},
	}); err == nil {
		t.Error("expected an error for an unknown status")
	}
	if _, err := client.IndexedAssets.Get(context.Background(), "idx_1", "ia_1", &IndexedAssetGetOptions{
		EmbeddingOptions: []string{"unknown"},
	}); err == nil {
		t.Error("expected an error for an unknown embedding option")
	}
}

func TestEntitiesAssetMethodsReturnUpdatedEntity(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/entity-collections/ec_1/entities/en_1/assets" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var body EntityAssetsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if len(body.AssetIDs) != 1 || body.AssetIDs[0] != "asset_1" {
			t.Errorf("asset_ids = %v", body.AssetIDs)
		}
		status := EntityStatusProcessing
		if r.Method == http.MethodDelete {
			status = EntityStatusReady
		}
		writeJSON(t, w, http.StatusOK, map[string]any{
			"_id":       "en_1",
			"name":      "Ada",
			"asset_ids": body.AssetIDs,
			"status":    status,
		})
	})

	added, err := client.Entities.AddEntityAssets(context.Background(), "ec_1", "en_1", "asset_1")
	if err != nil {
		t.Fatalf("AddEntityAssets: %v", err)
	}
	if added.ID != "en_1" || added.Status != EntityStatusProcessing {
		t.Fatalf("unexpected entity %+v", added)
	}

	removed, err := client.Entities.RemoveEntityAssets(context.Background(), "ec_1", "en_1", "asset_1")
	if err != nil {
		t.Fatalf("RemoveEntityAssets: %v", err)
	}
	if removed.ID != "en_1" || removed.Status != EntityStatusReady {
		t.Fatalf("unexpected entity %+v", removed)
	}
}

func TestTaskListFilters(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if got := query.Get("duration"); got != "12.5" {
			t.Errorf("duration = %q", got)
		}
		if got := query.Get("width"); got != "1920" {
			t.Errorf("width = %q", got)
		}
		if got := query.Get("height"); got != "1080" {
			t.Errorf("height = %q", got)
		}
		if got := query.Get("status"); got != TaskStatusReady {
			t.Errorf("status = %q", got)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"data": []any{}, "page_info": map[string]any{}})
	})

	if _, err := client.Tasks.List(context.Background(), &TaskListOptions{
		Status:   []string{TaskStatusReady},
		Duration: ptrFloat(12.5),
		Width:    ptrInt(1920),
		Height:   ptrInt(1080),
	}); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestIndexCreateValidation(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected request")
	})
	cases := []*IndexCreateRequest{
		{IndexName: "x", Models: []IndexModel{{ModelName: ModelPegasus15, ModelOptions: []string{ModalityVisual}}}},
		{IndexName: "x", Models: []IndexModel{{ModelName: ModelMarengo30, ModelOptions: []string{"conversation"}}}},
		{IndexName: "x", Models: []IndexModel{{ModelName: ModelMarengo30, ModelOptions: []string{ModalityVisual, ModalityVisual}}}},
		{IndexName: "x", Addons: []string{"unknown"}},
	}
	for i, request := range cases {
		if _, err := client.Indexes.Create(context.Background(), request); err == nil {
			t.Errorf("case %d: expected a validation error", i)
		}
	}
}

func TestIndexListJoinsModelOptions(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("model_options"); got != "visual,audio" {
			t.Errorf("model_options = %q", got)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"data": []any{}, "page_info": map[string]any{}})
	})
	if _, err := client.Indexes.List(context.Background(), &IndexListOptions{
		ModelOptions: []string{ModalityVisual, ModalityAudio},
	}); err != nil {
		t.Fatalf("List: %v", err)
	}
}

func TestSearchRejectsInvalidOptions(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected request")
	})
	cases := []*SearchRequest{
		{IndexID: "idx_1", SearchOptions: []string{"unknown"}, QueryText: "x"},
		{IndexID: "idx_1", SearchOptions: []string{ModalityVisual, ModalityVisual}, QueryText: "x"},
		{IndexID: "idx_1", SearchOptions: []string{ModalityVisual}, QueryText: "x", TranscriptionOptions: []string{TranscriptionOptionLexical}},
		{IndexID: "idx_1", SearchOptions: []string{ModalityVisual}, QueryText: "x", GroupBy: "scene"},
		{IndexID: "idx_1", SearchOptions: []string{ModalityVisual}, QueryText: "x", Operator: "xor"},
		{IndexID: "idx_1", SearchOptions: []string{ModalityVisual}, QueryText: "x", PageLimit: 51},
	}
	for i, request := range cases {
		if _, err := client.Search.Query(context.Background(), request); err == nil {
			t.Errorf("case %d: expected a validation error", i)
		}
	}
}

func TestAnalyzeSyncTruncationIsAWarning(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]any{
			"data":          "partial",
			"finish_reason": "length",
			"error":         map[string]any{"message": "truncated"},
		})
	})
	response, err := client.Analyze.Analyze(context.Background(), &AnalyzeRequest{
		Video:  VideoFromAssetID("asset_1"),
		Prompt: "x",
	})
	if err != nil {
		t.Fatalf("Analyze returned %v, want nil for a truncation warning", err)
	}
	if response.Data != "partial" || response.Error == nil {
		t.Fatalf("unexpected response %+v", response)
	}
}

func ptrFloat(value float64) *float64 { return &value }
func ptrInt(value int) *int           { return &value }

func TestAssetListValidatesAssetTypes(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query()["asset_types"]; len(got) != 2 || got[0] != AssetTypeVideo || got[1] != AssetTypeDocument {
			t.Errorf("asset_types = %v", got)
		}
		writeJSON(t, w, http.StatusOK, map[string]any{"data": []any{}, "page_info": map[string]any{}})
	})

	if _, err := client.Assets.List(context.Background(), &AssetListOptions{
		AssetTypes: []string{AssetTypeVideo, AssetTypeDocument},
	}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, err := client.Assets.List(context.Background(), &AssetListOptions{
		AssetTypes: []string{"spreadsheet"},
	}); err == nil {
		t.Fatal("expected an error for an unknown asset type")
	}
}
