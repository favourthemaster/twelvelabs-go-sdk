package twelvelabs

import (
	"context"
	"io"
	"mime"
	"os"
	"path/filepath"
	"time"
)

// Pagination controls list endpoints that page by page number.
type Pagination struct {
	Page      int
	PageLimit int
}

// Range filters a numeric field by inclusive bounds. A nil bound is not sent,
// so the zero value applies no filter. Setting both bounds to the same value
// filters for an exact match.
type Range[T int | float64] struct {
	GTE *T `json:"gte,omitempty"`
	LTE *T `json:"lte,omitempty"`
}

// IsZero reports whether the range has no bounds.
func (r Range[T]) IsZero() bool {
	return r.GTE == nil && r.LTE == nil
}

// PageInfo describes a page of results.
type PageInfo struct {
	LimitPerPage int `json:"limit_per_page"`
	Page         int `json:"page"`
	TotalPage    int `json:"total_page"`
	TotalResults int `json:"total_results"`
	// TotalInnerMatches is returned by grouped search responses.
	TotalInnerMatches int `json:"total_inner_matches"`
	// PageExpiresAt is set by search, whose result pages are cached.
	PageExpiresAt string `json:"page_expires_at"`
	// NextPageToken is set by search. Pass it to Search.GetPage.
	NextPageToken string `json:"next_page_token"`
}

// Page is a single page of results.
type Page[T any] struct {
	Data     []T       `json:"data"`
	PageInfo *PageInfo `json:"page_info"`
}

// WaitOptions controls the polling helpers.
type WaitOptions struct {
	// PollInterval between status checks. Defaults to 5 seconds.
	PollInterval time.Duration
	// Timeout limits the total wait. Zero means no limit.
	Timeout time.Duration
	// OnPoll, when set, is called after every status check.
	OnPoll func(status string) error
}

// FileUpload is a local file to send as multipart/form-data.
type FileUpload struct {
	// Filename is sent to the platform. Defaults to the path's base name.
	Filename string
	// Content is read until EOF.
	Content io.Reader
	// ContentType overrides the MIME type. When empty it is inferred from the
	// filename extension, falling back to application/octet-stream.
	ContentType string

	closer io.Closer
}

// NewFileUpload opens path for upload. Close it when the request completes.
func NewFileUpload(path string) (*FileUpload, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &FileUpload{
		Filename: filepath.Base(path),
		Content:  file,
		closer:   file,
	}, nil
}

// Close releases the underlying file, if the upload owns one.
func (f *FileUpload) Close() error {
	if f == nil || f.closer == nil {
		return nil
	}
	return f.closer.Close()
}

func (f *FileUpload) contentType() string {
	if f.ContentType != "" {
		return f.ContentType
	}
	if ext := filepath.Ext(f.Filename); ext != "" {
		if ct := mime.TypeByExtension(ext); ct != "" {
			return ct
		}
	}
	return "application/octet-stream"
}

func (f *FileUpload) validate(field string) error {
	if f == nil {
		return validationError(field, "a file is required")
	}
	if f.Filename == "" {
		return validationError(field, "a filename is required")
	}
	if f.Content == nil {
		return validationError(field, "file content is required")
	}
	return nil
}

// MediaSource identifies media by URL, asset ID, or inline base64 data.
// Provide exactly one field.
type MediaSource struct {
	URL          string `json:"url,omitempty"`
	AssetID      string `json:"asset_id,omitempty"`
	Base64String string `json:"base64_string,omitempty"`
}

// MediaSourceFromURL references media by its public URL.
func MediaSourceFromURL(mediaURL string) MediaSource {
	return MediaSource{URL: mediaURL}
}

// MediaSourceFromAssetID references media by the ID of a ready asset.
func MediaSourceFromAssetID(assetID string) MediaSource {
	return MediaSource{AssetID: assetID}
}

// MediaSourceFromBase64 references inline base64-encoded media data.
func MediaSourceFromBase64(encoded string) MediaSource {
	return MediaSource{Base64String: encoded}
}

func (m MediaSource) validate(field string) error {
	provided := 0
	for _, value := range []string{m.URL, m.AssetID, m.Base64String} {
		if value != "" {
			provided++
		}
	}
	if provided != 1 {
		return validationError(field, "set exactly one of URL, AssetID, or Base64String")
	}
	return nil
}

// HLS describes HLS streaming artefacts for an asset or indexed asset.
type HLS struct {
	VideoURL      string   `json:"video_url,omitempty"`
	ThumbnailURLs []string `json:"thumbnail_urls,omitempty"`
	Status        string   `json:"status,omitempty"`
	UpdatedAt     string   `json:"updated_at,omitempty"`
}

// ErrorBody is a simple code/message error attached to a resource.
type ErrorBody struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// String implements fmt.Stringer.
func (e *ErrorBody) String() string {
	if e == nil {
		return ""
	}
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

const (
	// StatusProcessing means the platform is still working on the resource.
	StatusProcessing = "processing"
	// StatusReady means the resource is usable.
	StatusReady = "ready"
	// StatusFailed means processing failed.
	StatusFailed = "failed"
)

// ensureContext guards against a nil context being passed by callers.
func ensureContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
