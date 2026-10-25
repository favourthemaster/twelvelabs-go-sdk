package twelvelabs

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
)

// Search grouping values.
const (
	GroupByVideo = "video"
	GroupByClip  = "clip"
)

// Search operators.
const (
	OperatorOr  = "or"
	OperatorAnd = "and"
)

// Transcription matching modes.
const (
	TranscriptionOptionLexical  = "lexical"
	TranscriptionOptionSemantic = "semantic"
)

// SearchMediaTypeImage is the only media type accepted by Search.
const SearchMediaTypeImage = "image"

// SearchService finds moments in indexed content.
type SearchService struct {
	client *Client
}

// SearchRequest describes a search. It is always sent as multipart/form-data.
//
// Provide QueryText, one or more QueryMediaURLs, or one or more
// QueryMediaFiles. To combine an image with text in a single query, set both
// QueryText and the image fields.
type SearchRequest struct {
	// IndexID is the index to search. Required.
	IndexID string
	// QueryText is a natural-language query, or an additional prompt alongside
	// an image query.
	QueryText string
	// SearchOptions selects the modalities to match: ModalityVisual,
	// ModalityAudio, and/or ModalityTranscription. Required.
	SearchOptions []string
	// QueryMediaType must be "image" when QueryMediaURLs or QueryMediaFiles is
	// set. It is required for media queries.
	QueryMediaType string
	// QueryMediaURLs are public image URLs to use as a query. Up to 10.
	QueryMediaURLs []string
	// QueryMediaFiles are local images to use as a query. Up to 10.
	QueryMediaFiles []*FileUpload
	// TranscriptionOptions refines transcription matching to "lexical",
	// "semantic", or both.
	TranscriptionOptions []string
	// GroupBy is "video" or "clip". The API defaults to "clip".
	GroupBy string
	// Operator combines multiple search options with "or" or "and". The API
	// defaults to "or".
	Operator string
	// PageLimit caps the results per page. The API defaults to 10.
	PageLimit int
	// Filter is a JSON-marshalable object that filters results by system or
	// user metadata. It is encoded to a JSON string.
	Filter any
	// IncludeUserMetadata asks the platform to include user metadata in the
	// results.
	IncludeUserMetadata *bool
}

// SearchResult is one matching moment.
type SearchResult struct {
	Start         float64 `json:"start"`
	End           float64 `json:"end"`
	VideoID       string  `json:"video_id"`
	Rank          int     `json:"rank"`
	ThumbnailURL  string  `json:"thumbnail_url,omitempty"`
	Transcription string  `json:"transcription,omitempty"`
	// ID is the indexed asset ID. It is only set when grouping by video.
	ID           string         `json:"id,omitempty"`
	UserMetadata map[string]any `json:"user_metadata,omitempty"`
	// Clips holds the per-clip matches when grouping by video.
	Clips []SearchClip `json:"clips,omitempty"`
}

// SearchClip is one matching moment inside a grouped search result.
type SearchClip struct {
	Start         float64        `json:"start"`
	End           float64        `json:"end"`
	Rank          int            `json:"rank"`
	ThumbnailURL  string         `json:"thumbnail_url,omitempty"`
	Transcription string         `json:"transcription,omitempty"`
	VideoID       string         `json:"video_id,omitempty"`
	UserMetadata  map[string]any `json:"user_metadata,omitempty"`
}

// SearchPageInfo describes a page of search results. Search pages are cached
// by the platform and expire.
type SearchPageInfo struct {
	LimitPerPage      int    `json:"limit_per_page"`
	PageExpiresAt     string `json:"page_expires_at"`
	TotalResults      int    `json:"total_results"`
	TotalInnerMatches int    `json:"total_inner_matches"`
	NextPageToken     string `json:"next_page_token"`
	PrevPageToken     string `json:"prev_page_token"`
}

// SearchPool describes the index that was searched.
type SearchPool struct {
	TotalCount    int     `json:"total_count"`
	TotalDuration float64 `json:"total_duration"`
	IndexID       string  `json:"index_id"`
}

// SearchResponse is one page of search results.
type SearchResponse struct {
	Data       []SearchResult  `json:"data"`
	PageInfo   *SearchPageInfo `json:"page_info,omitempty"`
	SearchPool *SearchPool     `json:"search_pool,omitempty"`
}

// Query runs a search. Pass PageInfo.NextPageToken to GetPage for more
// results; the token is valid until PageInfo.PageExpiresAt.
func (s *SearchService) Query(ctx context.Context, request *SearchRequest) (*SearchResponse, error) {
	ctx = ensureContext(ctx)
	if request == nil {
		return nil, validationError("request", "is required")
	}
	if request.IndexID == "" {
		return nil, validationError("IndexID", "is required")
	}
	if len(request.SearchOptions) == 0 {
		return nil, validationError("SearchOptions", "at least one of visual, audio, or transcription is required")
	}
	seenOptions := make(map[string]bool, len(request.SearchOptions))
	for i, option := range request.SearchOptions {
		switch option {
		case ModalityVisual, ModalityAudio, ModalityTranscription:
		default:
			return nil, validationError("SearchOptions", "[%d] must be %q, %q, or %q", i, ModalityVisual, ModalityAudio, ModalityTranscription)
		}
		if seenOptions[option] {
			return nil, validationError("SearchOptions", "[%d] duplicates %q", i, option)
		}
		seenOptions[option] = true
	}
	for i, option := range request.TranscriptionOptions {
		switch option {
		case TranscriptionOptionLexical, TranscriptionOptionSemantic:
		default:
			return nil, validationError("TranscriptionOptions", "[%d] must be %q or %q", i, TranscriptionOptionLexical, TranscriptionOptionSemantic)
		}
		if !seenOptions[ModalityTranscription] {
			return nil, validationError("TranscriptionOptions", "requires %q in SearchOptions", ModalityTranscription)
		}
	}
	if request.GroupBy != "" && request.GroupBy != GroupByVideo && request.GroupBy != GroupByClip {
		return nil, validationError("GroupBy", "must be %q or %q", GroupByVideo, GroupByClip)
	}
	if request.Operator != "" && request.Operator != OperatorOr && request.Operator != OperatorAnd {
		return nil, validationError("Operator", "must be %q or %q", OperatorOr, OperatorAnd)
	}
	if request.PageLimit < 0 || request.PageLimit > 50 {
		return nil, validationError("PageLimit", "must be between 1 and 50 when set")
	}

	hasMedia := len(request.QueryMediaURLs) > 0 || len(request.QueryMediaFiles) > 0
	if !hasMedia && request.QueryText == "" {
		return nil, validationError("QueryText", "a text query or a media query is required")
	}
	if hasMedia {
		if request.QueryMediaType == "" {
			request.QueryMediaType = SearchMediaTypeImage
		}
		if request.QueryMediaType != SearchMediaTypeImage {
			return nil, validationError("QueryMediaType", "must be %q", SearchMediaTypeImage)
		}
		if len(request.QueryMediaURLs)+len(request.QueryMediaFiles) > 10 {
			return nil, validationError("QueryMediaFiles", "at most 10 media files or URLs are allowed")
		}
	}
	for i, file := range request.QueryMediaFiles {
		if err := file.validate(fmt.Sprintf("QueryMediaFiles[%d]", i)); err != nil {
			return nil, err
		}
	}

	var filter string
	if request.Filter != nil {
		encoded, err := json.Marshal(request.Filter)
		if err != nil {
			return nil, fmt.Errorf("twelvelabs: encode search filter: %w", err)
		}
		filter = string(encoded)
	}

	body := &multipartBody{write: func(w *multipart.Writer) error {
		for _, option := range request.SearchOptions {
			if option == "" {
				continue
			}
			if err := w.WriteField("search_options", option); err != nil {
				return err
			}
		}
		if err := w.WriteField("index_id", request.IndexID); err != nil {
			return err
		}
		if request.QueryText != "" {
			if err := w.WriteField("query_text", request.QueryText); err != nil {
				return err
			}
		}
		if request.QueryMediaType != "" {
			if err := w.WriteField("query_media_type", request.QueryMediaType); err != nil {
				return err
			}
		}
		for _, mediaURL := range request.QueryMediaURLs {
			if err := w.WriteField("query_media_url", mediaURL); err != nil {
				return err
			}
		}
		for _, file := range request.QueryMediaFiles {
			if err := writeFilePart(w, "query_media_file", file); err != nil {
				return err
			}
		}
		for _, option := range request.TranscriptionOptions {
			if err := w.WriteField("transcription_options", option); err != nil {
				return err
			}
		}
		if request.GroupBy != "" {
			if err := w.WriteField("group_by", request.GroupBy); err != nil {
				return err
			}
		}
		if request.Operator != "" {
			if err := w.WriteField("operator", request.Operator); err != nil {
				return err
			}
		}
		if request.PageLimit > 0 {
			if err := w.WriteField("page_limit", fmt.Sprint(request.PageLimit)); err != nil {
				return err
			}
		}
		if filter != "" {
			if err := w.WriteField("filter", filter); err != nil {
				return err
			}
		}
		if request.IncludeUserMetadata != nil {
			return w.WriteField("include_user_metadata", boolString(*request.IncludeUserMetadata))
		}
		return nil
	}}

	var response SearchResponse
	if err := s.client.transport.doMultipart(ctx, http.MethodPost, "/search", nil, body, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetPage retrieves the next page of a search using the token from
// PageInfo.NextPageToken.
func (s *SearchService) GetPage(ctx context.Context, pageToken string, includeUserMetadata *bool) (*SearchResponse, error) {
	ctx = ensureContext(ctx)
	if pageToken == "" {
		return nil, validationError("pageToken", "is required")
	}
	query := url.Values{}
	if includeUserMetadata != nil {
		query.Set("include_user_metadata", boolString(*includeUserMetadata))
	}
	var response SearchResponse
	path := "/search/" + escapePath(pageToken)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, query, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
