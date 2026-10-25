package twelvelabs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// Multipart upload status values.
const (
	UploadStatusActive    = "active"
	UploadStatusCompleted = "completed"
	UploadStatusFailed    = "failed"
	UploadStatusExpired   = "expired"
)

const (
	// maxUploadConcurrency bounds parallel chunk uploads.
	maxUploadConcurrency = 16
	// maxPresignedURLBatch is the largest number of URLs the API issues per call.
	maxPresignedURLBatch = 50
	// reportChunksBatchSize is the number of proofs sent per report call.
	reportChunksBatchSize = 100
)

// MultipartUploadRequest starts a resumable multipart upload. Either set File
// (Upload then streams it) or call CreateMultipartUpload / ReportChunks
// manually.
type MultipartUploadRequest struct {
	// AssetType must be AssetTypeVideo, AssetTypeAudio, or AssetTypeImage. When
	// empty it is inferred from the filename.
	AssetType string
	// Filename is the name the platform stores.
	Filename string
	// TotalSize is the size of the file in bytes. Required when File is nil,
	// and inferred from File when it is an *os.File.
	TotalSize int64
	// File is the content to upload.
	File *FileUpload
	// EnableHLS generates an HLS playlist. The API defaults to false.
	EnableHLS *bool
	// EnableThumbnail generates thumbnails. The API defaults to false.
	EnableThumbnail *bool
	// UserMetadata is arbitrary key/value metadata.
	UserMetadata map[string]any
	// Concurrency is the number of chunks uploaded in parallel. Defaults to 3
	// and is capped at 16.
	Concurrency int
	// OnProgress, when set, is called after each chunk is uploaded. Calls are
	// serialized.
	OnProgress func(uploadedChunks, totalChunks int)
}

// CreateAssetUploadRequest opens a multipart upload session.
type CreateAssetUploadRequest struct {
	Filename  string `json:"filename"`
	Type      string `json:"type"`
	TotalSize int64  `json:"total_size"`
	// EnableHLS generates an HLS playlist. The API defaults to false.
	EnableHLS *bool `json:"enable_hls,omitempty"`
	// EnableThumbnail generates thumbnails. The API defaults to false.
	EnableThumbnail *bool          `json:"enable_thumbnail,omitempty"`
	UserMetadata    map[string]any `json:"user_metadata,omitempty"`
}

// PresignedChunk is one chunk's presigned upload URL.
type PresignedChunk struct {
	ChunkIndex int    `json:"chunk_index"`
	URL        string `json:"url"`
	ExpiresAt  string `json:"expires_at,omitempty"`
}

// CreateAssetUploadResponse describes a newly created upload session. The
// response may contain URLs for only some chunks; Upload requests the rest
// automatically.
type CreateAssetUploadResponse struct {
	UploadID      string            `json:"upload_id"`
	AssetID       string            `json:"asset_id"`
	UploadURLs    []PresignedChunk  `json:"upload_urls"`
	UploadHeaders map[string]string `json:"upload_headers"`
	ChunkSize     int64             `json:"chunk_size"`
	TotalChunks   int               `json:"total_chunks"`
	ExpiresAt     string            `json:"expires_at"`
}

// ChunkInfo is the status of one chunk in an upload session.
type ChunkInfo struct {
	Index      int    `json:"index"`
	Status     string `json:"status"`
	UploadedAt string `json:"uploaded_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	Error      string `json:"error,omitempty"`
}

// UploadStatus describes an upload session.
type UploadStatus struct {
	UploadID        string      `json:"upload_id"`
	Status          string      `json:"status"`
	UploadedChunks  []ChunkInfo `json:"uploaded_chunks"`
	ChunksCompleted int         `json:"chunks_completed"`
	ChunksFailed    int         `json:"chunks_failed"`
	ChunksPending   int         `json:"chunks_pending"`
	PageInfo        *PageInfo   `json:"page_info,omitempty"`
	TotalSize       int64       `json:"total_size"`
	UploadedSize    int64       `json:"uploaded_size"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
}

// IncompleteUploadSummary describes an upload session that is still open.
type IncompleteUploadSummary struct {
	UploadID    string `json:"upload_id"`
	Status      string `json:"status"`
	TotalSize   int64  `json:"total_size"`
	ChunkSize   int64  `json:"chunk_size"`
	TotalChunks int    `json:"total_chunks"`
	CreatedAt   string `json:"created_at,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

// CompletedChunk reports a successfully uploaded chunk.
type CompletedChunk struct {
	ChunkIndex int    `json:"chunk_index"`
	Proof      string `json:"proof"`
	ProofType  string `json:"proof_type"`
	ChunkSize  int64  `json:"chunk_size"`
}

// ReportChunksRequest reports uploaded chunks.
type ReportChunksRequest struct {
	CompletedChunks []CompletedChunk `json:"completed_chunks"`
}

// ReportChunksResponse is returned when reporting uploaded chunks.
type ReportChunksResponse struct {
	URL             string `json:"url,omitempty"`
	AssetID         string `json:"asset_id,omitempty"`
	ProcessedChunks int    `json:"processed_chunks,omitempty"`
	DuplicateChunks int    `json:"duplicate_chunks,omitempty"`
	TotalCompleted  int    `json:"total_completed,omitempty"`
}

// AdditionalPresignedURLsRequest asks for more presigned URLs.
type AdditionalPresignedURLsRequest struct {
	// Start is the 1-based index of the first chunk.
	Start int `json:"start"`
	// Count is the number of URLs to request. The API allows 1 to 50.
	Count int `json:"count"`
}

// AdditionalPresignedURLsResponse contains the requested presigned URLs.
type AdditionalPresignedURLsResponse struct {
	UploadID    string           `json:"upload_id"`
	StartIndex  int              `json:"start_index"`
	Count       int              `json:"count"`
	UploadURLs  []PresignedChunk `json:"upload_urls"`
	GeneratedAt string           `json:"generated_at"`
	ExpiresAt   string           `json:"expires_at"`
}

// CreateMultipartUpload opens a multipart upload session.
func (s *AssetsService) CreateMultipartUpload(ctx context.Context, request *CreateAssetUploadRequest) (*CreateAssetUploadResponse, error) {
	ctx = ensureContext(ctx)
	if request == nil {
		return nil, validationError("request", "is required")
	}
	if request.Filename == "" {
		return nil, validationError("Filename", "is required")
	}

	body := *request
	switch body.Type {
	case AssetTypeVideo, AssetTypeAudio, AssetTypeImage:
	case "":
		inferred, err := assetTypeFromFilename(body.Filename)
		if err != nil {
			return nil, err
		}
		body.Type = inferred
	default:
		return nil, validationError("Type", "must be %q, %q, or %q", AssetTypeVideo, AssetTypeAudio, AssetTypeImage)
	}
	if body.TotalSize <= 0 {
		return nil, validationError("TotalSize", "must be greater than zero")
	}

	var response CreateAssetUploadResponse
	if err := s.client.transport.doJSON(ctx, http.MethodPost, "/assets/multipart-uploads", nil, &body, &response); err != nil {
		return nil, err
	}
	if response.UploadID == "" {
		return nil, errEmptyID("upload session")
	}
	if response.TotalChunks <= 0 {
		return nil, fmt.Errorf("twelvelabs: upload session returned an invalid chunk count %d", response.TotalChunks)
	}
	if response.ChunkSize <= 0 {
		return nil, fmt.Errorf("twelvelabs: upload session returned an invalid chunk size %d", response.ChunkSize)
	}
	return &response, nil
}

// GetUploadStatus returns the status of an upload session.
func (s *AssetsService) GetUploadStatus(ctx context.Context, uploadID string, options *Pagination) (*UploadStatus, error) {
	ctx = ensureContext(ctx)
	if uploadID == "" {
		return nil, validationError("uploadID", "is required")
	}
	var status UploadStatus
	path := "/assets/multipart-uploads/" + escapePath(uploadID)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, encodeQuery(options), nil, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// ListIncompleteUploads lists upload sessions that have not completed.
func (s *AssetsService) ListIncompleteUploads(ctx context.Context, options *Pagination) (*Page[IncompleteUploadSummary], error) {
	ctx = ensureContext(ctx)
	var page Page[IncompleteUploadSummary]
	if err := s.client.transport.doJSON(ctx, http.MethodGet, "/assets/multipart-uploads", encodeQuery(options), nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// ReportChunks reports successfully uploaded chunks. Reports larger than the
// SDK batch size are split into several requests; the returned counts are
// aggregated.
func (s *AssetsService) ReportChunks(ctx context.Context, uploadID string, request *ReportChunksRequest) (*ReportChunksResponse, error) {
	ctx = ensureContext(ctx)
	if uploadID == "" {
		return nil, validationError("uploadID", "is required")
	}
	if request == nil || len(request.CompletedChunks) == 0 {
		return nil, validationError("CompletedChunks", "at least one chunk is required")
	}

	chunks := make([]CompletedChunk, len(request.CompletedChunks))
	for i, chunk := range request.CompletedChunks {
		if chunk.ChunkIndex <= 0 {
			return nil, validationError("CompletedChunks", "chunk %d has an invalid index; chunks are numbered starting from 1", i)
		}
		if chunk.Proof == "" {
			return nil, validationError("CompletedChunks", "chunk %d has no proof; an ETag is required", chunk.ChunkIndex)
		}
		if chunk.ChunkSize <= 0 {
			return nil, validationError("CompletedChunks", "chunk %d has an invalid size %d", chunk.ChunkIndex, chunk.ChunkSize)
		}
		switch chunk.ProofType {
		case "", "etag":
			chunk.ProofType = "etag"
		default:
			return nil, validationError("CompletedChunks", "chunk %d uses unsupported proof type %q", chunk.ChunkIndex, chunk.ProofType)
		}
		chunks[i] = chunk
	}

	var combined ReportChunksResponse
	for start := 0; start < len(chunks); start += reportChunksBatchSize {
		end := start + reportChunksBatchSize
		if end > len(chunks) {
			end = len(chunks)
		}
		var response ReportChunksResponse
		path := "/assets/multipart-uploads/" + escapePath(uploadID)
		body := &ReportChunksRequest{CompletedChunks: chunks[start:end]}
		if err := s.client.transport.doJSON(ctx, http.MethodPost, path, nil, body, &response); err != nil {
			return nil, err
		}
		if response.URL != "" {
			combined.URL = response.URL
		}
		if response.AssetID != "" {
			combined.AssetID = response.AssetID
		}
		combined.ProcessedChunks += response.ProcessedChunks
		combined.DuplicateChunks += response.DuplicateChunks
		if response.TotalCompleted > combined.TotalCompleted {
			combined.TotalCompleted = response.TotalCompleted
		}
	}
	return &combined, nil
}

// RequestPresignedURLs asks for additional presigned upload URLs. Start is
// 1-based and Count must be between 1 and 50.
func (s *AssetsService) RequestPresignedURLs(ctx context.Context, uploadID string, request *AdditionalPresignedURLsRequest) (*AdditionalPresignedURLsResponse, error) {
	ctx = ensureContext(ctx)
	if uploadID == "" {
		return nil, validationError("uploadID", "is required")
	}
	if request == nil {
		return nil, validationError("request", "is required")
	}
	if request.Start < 1 {
		return nil, validationError("Start", "must be at least 1; chunks are numbered starting from 1")
	}
	if request.Count < 1 || request.Count > maxPresignedURLBatch {
		return nil, validationError("Count", "must be between 1 and %d", maxPresignedURLBatch)
	}

	var response AdditionalPresignedURLsResponse
	path := "/assets/multipart-uploads/" + escapePath(uploadID) + "/presigned-urls"
	if err := s.client.transport.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Upload streams a file using the multipart upload flow: it opens a session,
// uploads every chunk in parallel, reports them, and returns the created asset.
// The asset is still processing; call WaitForReady before using it.
//
// Presigned URLs included in the create response are reused. Any chunk without
// a URL is fetched on demand in batches of at most 50. URLs are valid for one
// hour; an interrupted upload can be resumed with the manual methods.
func (s *AssetsService) Upload(ctx context.Context, request *MultipartUploadRequest) (*AssetDetail, error) {
	ctx = ensureContext(ctx)
	if request == nil {
		return nil, validationError("request", "is required")
	}
	if request.File == nil {
		return nil, validationError("File", "is required; use CreateMultipartUpload for manual uploads")
	}
	if err := request.File.validate("File"); err != nil {
		return nil, err
	}

	filename := request.Filename
	if filename == "" {
		filename = request.File.Filename
	}

	assetType := request.AssetType
	if assetType == "" {
		inferred, err := assetTypeFromFilename(filename)
		if err != nil {
			return nil, err
		}
		assetType = inferred
	}

	totalSize := request.TotalSize
	if seeker, ok := request.File.Content.(io.Seeker); ok && totalSize <= 0 {
		if size, err := seeker.Seek(0, io.SeekEnd); err == nil {
			if _, err := seeker.Seek(0, io.SeekStart); err == nil {
				totalSize = size
			}
		}
	}
	if totalSize <= 0 {
		return nil, validationError("TotalSize", "is required when the file size cannot be determined")
	}

	session, err := s.CreateMultipartUpload(ctx, &CreateAssetUploadRequest{
		Filename:        filename,
		Type:            assetType,
		TotalSize:       totalSize,
		EnableHLS:       request.EnableHLS,
		EnableThumbnail: request.EnableThumbnail,
		UserMetadata:    request.UserMetadata,
	})
	if err != nil {
		return nil, err
	}

	completed, err := s.uploadChunks(ctx, request.File.Content, session, totalSize, request.Concurrency, request.OnProgress)
	if err != nil {
		return nil, err
	}

	report, err := s.ReportChunks(ctx, session.UploadID, &ReportChunksRequest{CompletedChunks: completed})
	if err != nil {
		return nil, err
	}

	assetID := session.AssetID
	if assetID == "" {
		assetID = report.AssetID
	}
	if assetID == "" {
		return nil, fmt.Errorf("twelvelabs: upload completed but no asset ID was returned")
	}
	return s.Get(ctx, assetID)
}

func (s *AssetsService) uploadChunks(ctx context.Context, content io.Reader, session *CreateAssetUploadResponse, totalSize int64, concurrency int, onProgress func(int, int)) ([]CompletedChunk, error) {
	if session == nil {
		return nil, validationError("session", "is required")
	}
	if session.ChunkSize <= 0 {
		return nil, fmt.Errorf("twelvelabs: upload session returned an invalid chunk size")
	}
	if session.TotalChunks <= 0 {
		return nil, fmt.Errorf("twelvelabs: upload session returned an invalid chunk count")
	}
	if totalSize <= 0 {
		return nil, validationError("TotalSize", "must be greater than zero")
	}
	expectedChunks := (totalSize + session.ChunkSize - 1) / session.ChunkSize
	if int64(session.TotalChunks) != expectedChunks {
		return nil, fmt.Errorf("twelvelabs: upload session returned %d chunks for %d bytes at %d bytes per chunk", session.TotalChunks, totalSize, session.ChunkSize)
	}
	lastChunkSize := totalSize - session.ChunkSize*int64(session.TotalChunks-1)
	if lastChunkSize <= 0 || lastChunkSize > session.ChunkSize {
		return nil, fmt.Errorf("twelvelabs: upload session has an inconsistent size")
	}

	if concurrency <= 0 {
		concurrency = 3
	}
	if concurrency > maxUploadConcurrency {
		concurrency = maxUploadConcurrency
	}
	if concurrency > session.TotalChunks {
		concurrency = session.TotalChunks
	}

	urls := make(map[int]PresignedChunk, len(session.UploadURLs))
	for _, chunk := range session.UploadURLs {
		if chunk.ChunkIndex >= 1 && chunk.ChunkIndex <= session.TotalChunks && chunk.URL != "" {
			urls[chunk.ChunkIndex] = chunk
		}
	}
	if err := s.ensurePresignedURLs(ctx, session, urls); err != nil {
		return nil, err
	}

	type job struct {
		index int
		data  []byte
	}
	type result struct {
		chunk CompletedChunk
		err   error
	}

	jobs := make(chan job)
	results := make(chan result)
	readErr := make(chan error, 1)
	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				presigned := urls[j.index]
				etag, err := s.putChunk(ctx, presigned.URL, session.UploadHeaders, j.data)
				if err != nil {
					results <- result{err: fmt.Errorf("twelvelabs: chunk %d: %w", j.index, err)}
					continue
				}
				results <- result{chunk: CompletedChunk{
					ChunkIndex: j.index,
					Proof:      etag,
					ProofType:  "etag",
					ChunkSize:  int64(len(j.data)),
				}}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for index := 1; index <= session.TotalChunks; index++ {
			size := session.ChunkSize
			if index == session.TotalChunks {
				size = lastChunkSize
			}
			buffer := make([]byte, size)
			n, err := io.ReadFull(content, buffer)
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
					readErr <- fmt.Errorf("twelvelabs: the upload ended after fewer than the declared %d bytes; check TotalSize: %w", totalSize, err)
					return
				}
				readErr <- fmt.Errorf("twelvelabs: read upload content for chunk %d of %d: %w", index, session.TotalChunks, err)
				return
			}
			select {
			case jobs <- job{index: index, data: buffer[:n]}:
			case <-ctx.Done():
				return
			}
		}

		// The declared size must match the actual content; otherwise the
		// final chunk would be silently truncated or the upload would hang.
		var extra [1]byte
		n, err := io.ReadFull(content, extra[:])
		if n > 0 {
			readErr <- fmt.Errorf("twelvelabs: the upload contains more than the declared %d bytes; check TotalSize", totalSize)
			return
		}
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
			readErr <- fmt.Errorf("twelvelabs: read upload content: %w", err)
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var (
		completed []CompletedChunk
		firstErr  error
	)
	for item := range results {
		if item.err != nil {
			if firstErr == nil {
				firstErr = item.err
				cancel()
			}
			continue
		}
		completed = append(completed, item.chunk)
		if onProgress != nil {
			onProgress(len(completed), session.TotalChunks)
		}
	}
	select {
	case err := <-readErr:
		if firstErr == nil {
			firstErr = err
		}
	default:
	}
	if firstErr != nil {
		return nil, firstErr
	}
	if len(completed) != session.TotalChunks {
		return nil, fmt.Errorf("twelvelabs: uploaded %d of %d chunks", len(completed), session.TotalChunks)
	}

	return completed, nil
}

// ensurePresignedURLs requests URLs for every chunk that does not have one.
// The API issues URLs for contiguous ranges of at most 50 chunks.
func (s *AssetsService) ensurePresignedURLs(ctx context.Context, session *CreateAssetUploadResponse, urls map[int]PresignedChunk) error {
	var missing []int
	for index := 1; index <= session.TotalChunks; index++ {
		if _, ok := urls[index]; !ok {
			missing = append(missing, index)
		}
	}
	for position := 0; position < len(missing); {
		start := missing[position]
		end := position + 1
		for end < len(missing) && end-position < maxPresignedURLBatch && missing[end] == start+(end-position) {
			end++
		}
		count := end - position

		response, err := s.RequestPresignedURLs(ctx, session.UploadID, &AdditionalPresignedURLsRequest{
			Start: start,
			Count: count,
		})
		if err != nil {
			return err
		}
		for _, chunk := range response.UploadURLs {
			if chunk.ChunkIndex < start || chunk.ChunkIndex >= start+count {
				return fmt.Errorf("twelvelabs: presigned URL response contains unexpected chunk index %d", chunk.ChunkIndex)
			}
			if chunk.URL == "" {
				return fmt.Errorf("twelvelabs: presigned URL response for chunk %d is empty", chunk.ChunkIndex)
			}
			urls[chunk.ChunkIndex] = chunk
		}
		for index := start; index < start+count; index++ {
			if _, ok := urls[index]; !ok {
				return fmt.Errorf("twelvelabs: the API did not return a presigned URL for chunk %d", index)
			}
		}
		position = end
	}
	return nil
}

func (s *AssetsService) putChunk(ctx context.Context, presignedURL string, headers map[string]string, data []byte) (string, error) {
	parsed, err := url.Parse(presignedURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return "", fmt.Errorf("invalid presigned URL")
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, presignedURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	if request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/octet-stream")
	}
	request.ContentLength = int64(len(data))

	client := s.client.transport.uploadClient
	if client == nil {
		client = s.client.transport.httpClient
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("upload chunk: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _, _ := readLimited(response.Body, 512)
		detail := strings.TrimSpace(string(body))
		if detail != "" {
			return "", fmt.Errorf("upload chunk: HTTP %d: %s", response.StatusCode, detail)
		}
		return "", fmt.Errorf("upload chunk: HTTP %d", response.StatusCode)
	}
	_, _ = io.Copy(io.Discard, response.Body)

	etag := normalizeETag(response.Header.Get("ETag"))
	if etag == "" {
		return "", fmt.Errorf("upload chunk: the storage service returned no ETag")
	}
	return etag, nil
}

func normalizeETag(etag string) string {
	etag = strings.TrimSpace(etag)
	etag = strings.TrimPrefix(etag, "W/")
	return strings.Trim(etag, `"`)
}
