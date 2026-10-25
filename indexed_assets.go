package twelvelabs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Indexed asset status values. Only ready and failed are final.
const (
	IndexedAssetStatusPending    = "pending"
	IndexedAssetStatusQueued     = "queued"
	IndexedAssetStatusIndexing   = "indexing"
	IndexedAssetStatusValidating = "validating"
	IndexedAssetStatusReady      = "ready"
	IndexedAssetStatusFailed     = "failed"
)

// IndexedAssetsService adds assets to indexes and manages their lifecycle.
type IndexedAssetsService struct {
	client *Client
}

// IndexedAsset is an asset made searchable within an index. The platform calls
// this object a video in older documentation.
type IndexedAsset struct {
	ID             string                      `json:"_id"`
	AssetID        string                      `json:"asset_id"`
	CreatedAt      string                      `json:"created_at,omitempty"`
	UpdatedAt      string                      `json:"updated_at,omitempty"`
	IndexedAt      string                      `json:"indexed_at,omitempty"`
	Status         string                      `json:"status"`
	SystemMetadata *IndexedAssetSystemMetadata `json:"system_metadata,omitempty"`
	UserMetadata   map[string]any              `json:"user_metadata,omitempty"`
	HLS            *HLS                        `json:"hls,omitempty"`
	Embedding      *IndexedAssetEmbedding      `json:"embedding,omitempty"`
	Transcription  []TranscriptionEntry        `json:"transcription,omitempty"`
}

// IndexedAssetSystemMetadata describes the media the platform indexed.
type IndexedAssetSystemMetadata struct {
	Filename string  `json:"filename,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	FPS      float64 `json:"fps,omitempty"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
	Size     float64 `json:"size,omitempty"`
}

// IndexedAssetEmbedding contains the embeddings generated for an indexed asset.
type IndexedAssetEmbedding struct {
	ModelName      string          `json:"model_name,omitempty"`
	VideoEmbedding *VideoEmbedding `json:"video_embedding,omitempty"`
}

// VideoEmbedding is a list of per-segment embeddings.
type VideoEmbedding struct {
	Segments []VideoSegment `json:"segments,omitempty"`
}

// VideoSegment is one embedding covering a time range of an indexed asset.
type VideoSegment struct {
	Float           []float64 `json:"float"`
	StartOffsetSec  float64   `json:"start_offset_sec"`
	EndOffsetSec    float64   `json:"end_offset_sec"`
	EmbeddingOption string    `json:"embedding_option,omitempty"`
	EmbeddingScope  string    `json:"embedding_scope,omitempty"`
}

// Video is the deprecated alias returned by Indexes.ListVideos.
//
// Deprecated: use IndexedAsset.
type Video = IndexedAsset

// VideoDetail is the deprecated alias returned by Indexes.GetVideo.
//
// Deprecated: use IndexedAsset.
type VideoDetail = IndexedAsset

// VideoGetOptions configures Indexes.GetVideo.
//
// Deprecated: use IndexedAssetGetOptions.
type VideoGetOptions = IndexedAssetGetOptions

// IndexedAssetCreateRequest adds an asset to an index.
type IndexedAssetCreateRequest struct {
	// AssetID identifies a ready asset.
	AssetID string `json:"asset_id"`
	// EnableVideoStream generates an HLS playlist for the indexed asset. The
	// API defaults to false.
	EnableVideoStream *bool `json:"enable_video_stream,omitempty"`
	// UserMetadata is arbitrary key/value metadata scoped to this index entry.
	UserMetadata map[string]any `json:"user_metadata,omitempty"`
}

// UpdateIndexedAssetRequest replaces the user metadata of an indexed asset.
type UpdateIndexedAssetRequest struct {
	UserMetadata map[string]any `json:"user_metadata"`
}

// IndexedAssetGetOptions selects which optional data to retrieve.
type IndexedAssetGetOptions struct {
	// EmbeddingOptions requests embeddings for the given modalities. Valid
	// values are EmbeddingOptionVisual, EmbeddingOptionAudio,
	// EmbeddingOptionTranscription, and EmbeddingOptionVisualText.
	EmbeddingOptions []string
	// Transcription requests the transcription of the asset.
	Transcription *bool
}

// IndexedAssetListOptions filters List.
type IndexedAssetListOptions struct {
	Pagination
	// SortBy is "created_at" or "updated_at".
	SortBy string
	// SortOption is "asc" or "desc".
	SortOption string
	Status     []string
	Filename   string
	// Duration filters on the duration in seconds.
	Duration Range[float64]
	// FPS filters on frames per second.
	FPS Range[float64]
	// Width filters on width in pixels.
	Width Range[int]
	// Height filters on height in pixels.
	Height Range[int]
	// Size filters on size in bytes.
	Size Range[float64]
	// CreatedAt returns indexed assets created on or after this RFC 3339 time.
	CreatedAt string
	// UpdatedAt returns indexed assets last updated on or after this RFC 3339
	// time.
	UpdatedAt string
	// UserMetadata filters on user metadata. Each key becomes its own query
	// parameter, so the values must be strings, numbers, or booleans.
	UserMetadata map[string]any
}

// Create indexes an asset. Indexing is asynchronous: the returned indexed asset
// starts as pending or queued. Use WaitForReady to block until it is usable.
func (s *IndexedAssetsService) Create(ctx context.Context, indexID string, request *IndexedAssetCreateRequest) (*IndexedAsset, error) {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return nil, validationError("indexID", "is required")
	}
	if request == nil || request.AssetID == "" {
		return nil, validationError("AssetID", "is required")
	}

	var created IndexedAsset
	path := "/indexes/" + escapePath(indexID) + "/indexed-assets"
	if err := s.client.transport.doJSON(ctx, http.MethodPost, path, nil, request, &created); err != nil {
		return nil, err
	}
	if created.ID == "" {
		return nil, errEmptyID("indexed asset")
	}
	// The create response only carries _id and asset_id; fill in what the
	// caller asked for so the value is usable before the first poll.
	created.AssetID = request.AssetID
	return &created, nil
}

// List returns the indexed assets of an index.
func (s *IndexedAssetsService) List(ctx context.Context, indexID string, options *IndexedAssetListOptions) (*Page[IndexedAsset], error) {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return nil, validationError("indexID", "is required")
	}
	query, err := indexedAssetQuery(options)
	if err != nil {
		return nil, err
	}
	var page Page[IndexedAsset]
	path := "/indexes/" + escapePath(indexID) + "/indexed-assets"
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, query, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// Get retrieves an indexed asset. Pass options to request embeddings or the
// transcription; both are omitted unless explicitly asked for.
func (s *IndexedAssetsService) Get(ctx context.Context, indexID, indexedAssetID string, options *IndexedAssetGetOptions) (*IndexedAsset, error) {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return nil, validationError("indexID", "is required")
	}
	if indexedAssetID == "" {
		return nil, validationError("indexedAssetID", "is required")
	}
	query := url.Values{}
	if options != nil {
		if err := validateEmbeddingOptions(options.EmbeddingOptions); err != nil {
			return nil, err
		}
		addStrings(query, "embedding_option", options.EmbeddingOptions)
		if options.Transcription != nil {
			setIfNotEmpty(query, "transcription", boolString(*options.Transcription))
		}
	}
	var asset IndexedAsset
	path := "/indexes/" + escapePath(indexID) + "/indexed-assets/" + escapePath(indexedAssetID)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, query, nil, &asset); err != nil {
		return nil, err
	}
	return &asset, nil
}

// Update replaces the user metadata of an indexed asset.
func (s *IndexedAssetsService) Update(ctx context.Context, indexID, indexedAssetID string, metadata map[string]any) error {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return validationError("indexID", "is required")
	}
	if indexedAssetID == "" {
		return validationError("indexedAssetID", "is required")
	}
	body := UpdateIndexedAssetRequest{UserMetadata: metadata}
	path := "/indexes/" + escapePath(indexID) + "/indexed-assets/" + escapePath(indexedAssetID)
	return s.client.transport.doJSON(ctx, http.MethodPatch, path, nil, &body, nil)
}

// Delete removes an indexed asset from an index. The underlying asset is kept.
func (s *IndexedAssetsService) Delete(ctx context.Context, indexID, indexedAssetID string) error {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return validationError("indexID", "is required")
	}
	if indexedAssetID == "" {
		return validationError("indexedAssetID", "is required")
	}
	path := "/indexes/" + escapePath(indexID) + "/indexed-assets/" + escapePath(indexedAssetID)
	return s.client.transport.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// WaitForReady polls an indexed asset until it is ready or failed.
func (s *IndexedAssetsService) WaitForReady(ctx context.Context, indexID, indexedAssetID string, options *WaitOptions) (*IndexedAsset, error) {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return nil, validationError("indexID", "is required")
	}
	if indexedAssetID == "" {
		return nil, validationError("indexedAssetID", "is required")
	}

	var deadline time.Time
	if options != nil && options.Timeout > 0 {
		deadline = time.Now().Add(options.Timeout)
	}

	for {
		asset, err := s.Get(ctx, indexID, indexedAssetID, nil)
		if err != nil {
			return nil, err
		}

		switch asset.Status {
		case IndexedAssetStatusReady:
			return asset, nil
		case IndexedAssetStatusFailed:
			return asset, errTaskFailed("indexed asset", indexedAssetID, nil)
		}

		if options != nil && options.OnPoll != nil {
			if err := options.OnPoll(asset.Status); err != nil {
				return nil, err
			}
		}
		if !deadline.IsZero() && time.Now().After(deadline) {
			return nil, ErrTimeout
		}
		if err := sleepContext(ctx, options.waitInterval()); err != nil {
			return nil, err
		}
	}
}

func indexedAssetQuery(options *IndexedAssetListOptions) (url.Values, error) {
	query := url.Values{}
	if options == nil {
		return query, nil
	}
	query = encodeQuery(&options.Pagination)
	if err := validateIndexedAssetStatus(options.Status); err != nil {
		return nil, err
	}
	addStrings(query, "status", options.Status)
	setIfNotEmpty(query, "sort_by", options.SortBy)
	setIfNotEmpty(query, "sort_option", options.SortOption)
	setIfNotEmpty(query, "filename", options.Filename)
	setIfNotEmpty(query, "created_at", options.CreatedAt)
	setIfNotEmpty(query, "updated_at", options.UpdatedAt)
	if err := encodeRange(query, "duration", options.Duration); err != nil {
		return nil, err
	}
	if err := encodeRange(query, "fps", options.FPS); err != nil {
		return nil, err
	}
	if err := encodeRange(query, "width", options.Width); err != nil {
		return nil, err
	}
	if err := encodeRange(query, "height", options.Height); err != nil {
		return nil, err
	}
	if err := encodeRange(query, "size", options.Size); err != nil {
		return nil, err
	}
	for key, value := range options.UserMetadata {
		if key == "" {
			return nil, validationError("UserMetadata", "keys must not be empty")
		}
		if query.Has(key) {
			return nil, validationError("UserMetadata", "key %q conflicts with a built-in query parameter", key)
		}
		switch typed := value.(type) {
		case string:
			query.Set(key, typed)
		case bool:
			query.Set(key, strconv.FormatBool(typed))
		case int:
			query.Set(key, strconv.Itoa(typed))
		case int64:
			query.Set(key, strconv.FormatInt(typed, 10))
		case float64:
			query.Set(key, strconv.FormatFloat(typed, 'f', -1, 64))
		default:
			return nil, validationError("UserMetadata", "value for %q must be a string, number, or boolean, got %T", key, value)
		}
	}
	return query, nil
}

func validateIndexedAssetStatus(statuses []string) error {
	allowed := map[string]bool{
		IndexedAssetStatusPending:    true,
		IndexedAssetStatusQueued:     true,
		IndexedAssetStatusIndexing:   true,
		IndexedAssetStatusValidating: true,
		IndexedAssetStatusReady:      true,
		IndexedAssetStatusFailed:     true,
	}
	for i, status := range statuses {
		if !allowed[status] {
			return validationError("Status", "[%d] must be a valid indexed asset status", i)
		}
	}
	return nil
}

func validateEmbeddingOptions(options []string) error {
	allowed := map[string]bool{
		EmbeddingOptionVisual:        true,
		EmbeddingOptionAudio:         true,
		EmbeddingOptionTranscription: true,
		EmbeddingOptionVisualText:    true,
	}
	for i, option := range options {
		if !allowed[option] {
			return validationError("EmbeddingOptions", "[%d] must be %q, %q, %q, or %q",
				i, EmbeddingOptionVisual, EmbeddingOptionAudio, EmbeddingOptionTranscription, EmbeddingOptionVisualText)
		}
	}
	return nil
}

// encodeRange writes a Range as the JSON query value the range filters expect,
// for example duration={"gte":10,"lte":10}.
func encodeRange[T int | float64](query url.Values, key string, value Range[T]) error {
	if value.IsZero() {
		return nil
	}
	if value.GTE != nil && value.LTE != nil && *value.GTE > *value.LTE {
		return validationError(key, "GTE must not be greater than LTE")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return validationError(key, "cannot be encoded: %v", err)
	}
	query.Set(key, string(encoded))
	return nil
}
