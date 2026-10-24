package twelvelabs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
// Deprecated: use IndexedAssets.Get.
type VideoGetOptions struct {
	// EmbeddingOptions requests embeddings for the given modalities, for
	// example ModalityVisual or ModalityAudio.
	EmbeddingOptions []string
	// Transcription requests the transcription of the asset.
	Transcription *bool
}

// IndexedAssetCreateRequest adds an asset to an index.
type IndexedAssetCreateRequest struct {
	// AssetID identifies a ready asset.
	AssetID string `json:"asset_id"`
	// EnableVideoStream generates an HLS playlist for the indexed asset. The
	// API defaults to true.
	EnableVideoStream *bool `json:"enable_video_stream,omitempty"`
	// UserMetadata is arbitrary key/value metadata scoped to this index entry.
	UserMetadata map[string]any `json:"user_metadata,omitempty"`
}

// UpdateIndexedAssetRequest replaces the user metadata of an indexed asset.
type UpdateIndexedAssetRequest struct {
	UserMetadata map[string]any `json:"user_metadata"`
}

// IndexedAssetListOptions filters List.
type IndexedAssetListOptions struct {
	Pagination
	SortBy     string
	SortOption string
	Status     []string
	Filename   string
	CreatedAt  string
	UpdatedAt  string
	// UserMetadata filters on user metadata. The map is JSON-encoded into the
	// query string.
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
	var page Page[IndexedAsset]
	path := "/indexes/" + escapePath(indexID) + "/indexed-assets"
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, indexedAssetQuery(options), nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// Get retrieves an indexed asset, including its embeddings and transcription
// when the platform has generated them.
func (s *IndexedAssetsService) Get(ctx context.Context, indexID, indexedAssetID string) (*IndexedAsset, error) {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return nil, validationError("indexID", "is required")
	}
	if indexedAssetID == "" {
		return nil, validationError("indexedAssetID", "is required")
	}
	var asset IndexedAsset
	path := "/indexes/" + escapePath(indexID) + "/indexed-assets/" + escapePath(indexedAssetID)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, nil, nil, &asset); err != nil {
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
		asset, err := s.Get(ctx, indexID, indexedAssetID)
		if err != nil {
			return nil, err
		}

		switch asset.Status {
		case IndexedAssetStatusReady:
			return asset, nil
		case IndexedAssetStatusFailed:
			return asset, fmt.Errorf("twelvelabs: indexed asset %s failed", indexedAssetID)
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

func indexedAssetQuery(options *IndexedAssetListOptions) url.Values {
	query := url.Values{}
	if options == nil {
		return query
	}
	query = encodeQuery(&options.Pagination)
	setIfNotEmpty(query, "sort_by", options.SortBy)
	setIfNotEmpty(query, "sort_option", options.SortOption)
	setIfNotEmpty(query, "filename", options.Filename)
	setIfNotEmpty(query, "created_at", options.CreatedAt)
	setIfNotEmpty(query, "updated_at", options.UpdatedAt)
	addStrings(query, "status", options.Status)
	if len(options.UserMetadata) > 0 {
		if encoded, err := json.Marshal(options.UserMetadata); err == nil {
			query.Set("user_metadata", string(encoded))
		}
	}
	return query
}
