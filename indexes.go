package twelvelabs

import (
	"context"
	"net/http"
	"net/url"
)

// Model names accepted by the platform.
const (
	// ModelMarengo30 is the video understanding model used by indexes and
	// embeddings. It is the only model an index accepts.
	ModelMarengo30 = "marengo3.0"
	// ModelMarengo35 is the embedding model that adds Matryoshka dimensions,
	// multimodal inputs, and documents. Embed API v2 only.
	ModelMarengo35 = "marengo3.5"
	// ModelPegasus15 is the default analysis model.
	ModelPegasus15 = "pegasus1.5"
	// ModelPegasus16 is the latest analysis model and the only one that
	// supports image inputs and time_array segment fields.
	ModelPegasus16 = "pegasus1.6"
)

// Modalities an index can be configured to analyze.
const (
	ModalityVisual        = "visual"
	ModalityAudio         = "audio"
	ModalityTranscription = "transcription"
)

// Index status values derived from the index lifecycle.
const (
	IndexStatusProcessing = "processing"
	IndexStatusReady      = "ready"
	IndexStatusFailed     = "failed"
)

// IndexesService creates and manages indexes.
type IndexesService struct {
	client *Client
}

// Index is a container that makes indexed assets searchable.
type Index struct {
	ID            string       `json:"_id"`
	CreatedAt     string       `json:"created_at,omitempty"`
	UpdatedAt     string       `json:"updated_at,omitempty"`
	ExpiresAt     *string      `json:"expires_at,omitempty"`
	IndexName     string       `json:"index_name,omitempty"`
	TotalDuration float64      `json:"total_duration,omitempty"`
	VideoCount    float64      `json:"video_count,omitempty"`
	Models        []IndexModel `json:"models,omitempty"`
	Addons        []string     `json:"addons,omitempty"`
}

// IndexModel is one model configuration attached to an index.
type IndexModel struct {
	ModelName    string   `json:"model_name"`
	ModelOptions []string `json:"model_options"`
}

// IndexCreateRequest creates an index. Model configuration cannot be changed
// after creation.
type IndexCreateRequest struct {
	IndexName string       `json:"index_name"`
	Models    []IndexModel `json:"models"`
	// Addons enable optional platform capabilities, for example "thumbnail".
	Addons []string `json:"addons,omitempty"`
}

// IndexUpdateRequest renames an index.
type IndexUpdateRequest struct {
	IndexName string `json:"index_name"`
}

// IndexListOptions filters List.
type IndexListOptions struct {
	Pagination
	SortBy      string
	SortOption  string
	IndexName   string
	ModelFamily string
	// ModelOptions filters by modality, for example ModalityVisual.
	ModelOptions []string
	CreatedAt    string
	UpdatedAt    string
}

// Create creates an index. Only ModelMarengo30 is accepted.
func (s *IndexesService) Create(ctx context.Context, request *IndexCreateRequest) (*Index, error) {
	ctx = ensureContext(ctx)
	if request == nil {
		return nil, validationError("request", "is required")
	}
	if request.IndexName == "" {
		return nil, validationError("IndexName", "is required")
	}
	if len(request.Models) == 0 {
		request.Models = []IndexModel{{ModelName: ModelMarengo30, ModelOptions: []string{ModalityVisual, ModalityAudio}}}
	}
	for i, model := range request.Models {
		if model.ModelName == "" {
			return nil, validationError("Models", "model %d has no ModelName", i)
		}
		if len(model.ModelOptions) == 0 {
			return nil, validationError("Models", "model %d has no ModelOptions", i)
		}
	}

	var created struct {
		ID string `json:"_id"`
	}
	if err := s.client.transport.doJSON(ctx, http.MethodPost, "/indexes", nil, request, &created); err != nil {
		return nil, err
	}
	if created.ID == "" {
		return nil, errEmptyID("index")
	}
	return &Index{ID: created.ID, IndexName: request.IndexName, Models: request.Models, Addons: request.Addons}, nil
}

// List returns indexes, most recent first.
func (s *IndexesService) List(ctx context.Context, options *IndexListOptions) (*Page[Index], error) {
	ctx = ensureContext(ctx)
	query := url.Values{}
	if options != nil {
		query = encodeQuery(&options.Pagination)
		setIfNotEmpty(query, "sort_by", options.SortBy)
		setIfNotEmpty(query, "sort_option", options.SortOption)
		setIfNotEmpty(query, "index_name", options.IndexName)
		setIfNotEmpty(query, "model_family", options.ModelFamily)
		setIfNotEmpty(query, "created_at", options.CreatedAt)
		setIfNotEmpty(query, "updated_at", options.UpdatedAt)
		addStrings(query, "model_options", options.ModelOptions)
	}

	var page Page[Index]
	if err := s.client.transport.doJSON(ctx, http.MethodGet, "/indexes", query, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// Get retrieves an index.
func (s *IndexesService) Get(ctx context.Context, indexID string) (*Index, error) {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return nil, validationError("indexID", "is required")
	}
	var index Index
	path := "/indexes/" + escapePath(indexID)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, nil, nil, &index); err != nil {
		return nil, err
	}
	return &index, nil
}

// Update renames an index. The API returns an empty body; the index is
// re-fetched so the caller gets the current representation.
func (s *IndexesService) Update(ctx context.Context, indexID string, request *IndexUpdateRequest) (*Index, error) {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return nil, validationError("indexID", "is required")
	}
	if request == nil || request.IndexName == "" {
		return nil, validationError("IndexName", "is required")
	}
	path := "/indexes/" + escapePath(indexID)
	if err := s.client.transport.doJSON(ctx, http.MethodPut, path, nil, request, nil); err != nil {
		return nil, err
	}
	return s.Get(ctx, indexID)
}

// Delete removes an index and everything in it.
func (s *IndexesService) Delete(ctx context.Context, indexID string) error {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return validationError("indexID", "is required")
	}
	return s.client.transport.doJSON(ctx, http.MethodDelete, "/indexes/"+escapePath(indexID), nil, nil, nil)
}

// --- Deprecated video endpoints ---------------------------------------------
//
// TwelveLabs kept the v1.2 /indexes/{index-id}/videos routes as aliases for
// indexed assets. Use IndexedAssets instead.

// ListVideos returns the indexed assets of an index.
//
// Deprecated: use Client.IndexedAssets.List.
func (s *IndexesService) ListVideos(ctx context.Context, indexID string, options *IndexedAssetListOptions) (*Page[Video], error) {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return nil, validationError("indexID", "is required")
	}
	var page Page[Video]
	path := "/indexes/" + escapePath(indexID) + "/videos"
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, indexedAssetQuery(options), nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// GetVideo returns one indexed asset of an index.
//
// Deprecated: use Client.IndexedAssets.Get.
func (s *IndexesService) GetVideo(ctx context.Context, indexID, videoID string, options *VideoGetOptions) (*VideoDetail, error) {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return nil, validationError("indexID", "is required")
	}
	if videoID == "" {
		return nil, validationError("videoID", "is required")
	}
	query := url.Values{}
	if options != nil {
		addStrings(query, "embedding_option", options.EmbeddingOptions)
		if options.Transcription != nil {
			query.Set("transcription", boolString(*options.Transcription))
		}
	}
	var video VideoDetail
	path := "/indexes/" + escapePath(indexID) + "/videos/" + escapePath(videoID)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, query, nil, &video); err != nil {
		return nil, err
	}
	return &video, nil
}

// UpdateVideoMetadata replaces the user metadata of an indexed asset.
//
// Deprecated: use Client.IndexedAssets.Update.
func (s *IndexesService) UpdateVideoMetadata(ctx context.Context, indexID, videoID string, metadata map[string]any) error {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return validationError("indexID", "is required")
	}
	if videoID == "" {
		return validationError("videoID", "is required")
	}
	body := UpdateIndexedAssetRequest{UserMetadata: metadata}
	path := "/indexes/" + escapePath(indexID) + "/videos/" + escapePath(videoID)
	return s.client.transport.doJSON(ctx, http.MethodPatch, path, nil, &body, nil)
}

// DeleteVideo removes an indexed asset from an index.
//
// Deprecated: use Client.IndexedAssets.Delete.
func (s *IndexesService) DeleteVideo(ctx context.Context, indexID, videoID string) error {
	ctx = ensureContext(ctx)
	if indexID == "" {
		return validationError("indexID", "is required")
	}
	if videoID == "" {
		return validationError("videoID", "is required")
	}
	path := "/indexes/" + escapePath(indexID) + "/videos/" + escapePath(videoID)
	return s.client.transport.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func setIfNotEmpty(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}
