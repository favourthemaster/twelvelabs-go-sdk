package twelvelabs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// Multipart upload status values.
const (
	UploadStatusActive    = "active"
	UploadStatusCompleted = "completed"
	UploadStatusFailed    = "failed"
	UploadStatusExpired   = "expired"
)

// MultipartUploadRequest starts a resumable multipart upload. Either set File
// (Upload then streams it) or call CreateMultipartUpload / ReportChunks
// manually.
type MultipartUploadRequest struct {
	// AssetType must be "video", "audio", or "image". When empty it is inferred
	// from the filename.
	AssetType string
	// Filename is the name the platform stores.
	Filename string
	// TotalSize is the size of the file in bytes. Required when File is nil,
	// and inferred from File when it is an *os.File.
	TotalSize int64
	// File is the content to upload.
	File *FileUpload
	// EnableHLS generates an HLS playlist. The API defaults to true.
	EnableHLS *bool
	// EnableThumbnail generates thumbnails. The API defaults to true.
	EnableThumbnail *bool
	// UserMetadata is arbitrary key/value metadata.
	UserMetadata map[string]any
	// Concurrency is the number of chunks uploaded in parallel. Defaults to 3.
	Concurrency int
	// OnProgress, when set, is called after each chunk is uploaded.
	OnProgress func(uploadedChunks, totalChunks int)
}

// CreateAssetUploadRequest opens a multipart upload session.
type CreateAssetUploadRequest struct {
	Filename        string         `json:"filename"`
	Type            string         `json:"type"`
	TotalSize       int64          `json:"total_size"`
	EnableHLS       *bool          `json:"enable_hls,omitempty"`
	EnableThumbnail *bool          `json:"enable_thumbnail,omitempty"`
	UserMetadata    map[string]any `json:"user_metadata,omitempty"`
}

// PresignedChunk is one chunk's presigned upload URL.
type PresignedChunk struct {
	ChunkIndex int    `json:"chunk_index"`
	URL        string `json:"url"`
	ExpiresAt  string `json:"expires_at,omitempty"`
}

// CreateAssetUploadResponse describes a newly created upload session.
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
	UploadID     string `json:"upload_id"`
	Filename     string `json:"filename"`
	AssetID      string `json:"asset_id"`
	Status       string `json:"status"`
	TotalSize    int64  `json:"total_size"`
	UploadedSize int64  `json:"uploaded_size"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
	ExpiresAt    string `json:"expires_at"`
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
	Start int `json:"start"`
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
	switch request.Type {
	case "video", "audio", "image":
	case "":
		inferred, err := assetTypeFromFilename(request.Filename)
		if err != nil {
			return nil, err
		}
		request.Type = inferred
	default:
		return nil, validationError("Type", "must be video, audio, or image")
	}
	if request.TotalSize <= 0 {
		return nil, validationError("TotalSize", "must be greater than zero")
	}

	var response CreateAssetUploadResponse
	if err := s.client.transport.doJSON(ctx, http.MethodPost, "/assets/multipart-uploads", nil, request, &response); err != nil {
		return nil, err
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

// ReportChunks reports successfully uploaded chunks.
func (s *AssetsService) ReportChunks(ctx context.Context, uploadID string, request *ReportChunksRequest) (*ReportChunksResponse, error) {
	ctx = ensureContext(ctx)
	if uploadID == "" {
		return nil, validationError("uploadID", "is required")
	}
	if request == nil || len(request.CompletedChunks) == 0 {
		return nil, validationError("CompletedChunks", "at least one chunk is required")
	}
	for i := range request.CompletedChunks {
		chunk := &request.CompletedChunks[i]
		if chunk.ChunkIndex <= 0 {
			return nil, validationError("ChunkIndex", "chunks are numbered starting from 1")
		}
		if chunk.Proof == "" {
			return nil, validationError("Proof", "an ETag is required")
		}
		if chunk.ProofType == "" {
			chunk.ProofType = "etag"
		}
	}
	var response ReportChunksResponse
	path := "/assets/multipart-uploads/" + escapePath(uploadID)
	if err := s.client.transport.doJSON(ctx, http.MethodPost, path, nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// RequestPresignedURLs asks for additional presigned upload URLs.
func (s *AssetsService) RequestPresignedURLs(ctx context.Context, uploadID string, request *AdditionalPresignedURLsRequest) (*AdditionalPresignedURLsResponse, error) {
	ctx = ensureContext(ctx)
	if uploadID == "" {
		return nil, validationError("uploadID", "is required")
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

	completed, err := s.uploadChunks(ctx, request.File.Content, session, request.Concurrency, request.OnProgress)
	if err != nil {
		return nil, err
	}

	if _, err := s.ReportChunks(ctx, session.UploadID, &ReportChunksRequest{CompletedChunks: completed}); err != nil {
		return nil, err
	}

	if session.AssetID == "" {
		return nil, fmt.Errorf("twelvelabs: upload completed but no asset ID was returned")
	}
	return s.Get(ctx, session.AssetID)
}

func (s *AssetsService) uploadChunks(ctx context.Context, content io.Reader, session *CreateAssetUploadResponse, concurrency int, onProgress func(int, int)) ([]CompletedChunk, error) {
	if session.ChunkSize <= 0 {
		return nil, fmt.Errorf("twelvelabs: upload session returned an invalid chunk size")
	}
	if concurrency <= 0 {
		concurrency = 3
	}

	urls := make(map[int]PresignedChunk, len(session.UploadURLs))
	for _, chunk := range session.UploadURLs {
		urls[chunk.ChunkIndex] = chunk
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
				presigned, ok := urls[j.index]
				if !ok {
					results <- result{err: fmt.Errorf("twelvelabs: no presigned URL for chunk %d", j.index)}
					continue
				}
				etag, err := s.putChunk(ctx, presigned.URL, session.UploadHeaders, j.data)
				if err != nil {
					results <- result{err: err}
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
			buffer := make([]byte, session.ChunkSize)
			n, err := io.ReadFull(content, buffer)
			switch err {
			case nil:
			case io.ErrUnexpectedEOF:
				if n == 0 {
					readErr <- fmt.Errorf("twelvelabs: the upload ended after %d of %d chunks; check TotalSize", index-1, session.TotalChunks)
					return
				}
			case io.EOF:
				readErr <- fmt.Errorf("twelvelabs: the upload ended after %d of %d chunks; check TotalSize", index-1, session.TotalChunks)
				return
			default:
				readErr <- fmt.Errorf("twelvelabs: read upload content: %w", err)
				return
			}
			select {
			case jobs <- job{index: index, data: buffer[:n]}:
			case <-ctx.Done():
				return
			}
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

func (s *AssetsService) putChunk(ctx context.Context, presignedURL string, headers map[string]string, data []byte) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, presignedURL, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("twelvelabs: build chunk request: %w", err)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	if request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", "application/octet-stream")
	}
	request.ContentLength = int64(len(data))

	response, err := s.client.transport.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("twelvelabs: upload chunk: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)

	if response.StatusCode >= 400 {
		return "", fmt.Errorf("twelvelabs: upload chunk: HTTP %d", response.StatusCode)
	}
	return trimQuotes(response.Header.Get("ETag")), nil
}
