package wrappers

import (
	"context"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/errors"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/services"
)

// SearchWrapper provides high-level multi-modal video search capabilities including
// text-based semantic search, image-based visual search, and advanced query options.
type SearchWrapper struct {
	service *services.SearchService
}

// NewSearchWrapper creates a new SearchWrapper instance.
func NewSearchWrapper(service *services.SearchService) *SearchWrapper {
	return &SearchWrapper{service: service}
}

// Query performs advanced multi-modal search with comprehensive options for filtering,
// pagination, and result customization. This is the most flexible search method.
//
// Supports both text and media-based queries:
//
// Text queries:
//   - Set QueryText parameter for semantic text search
//
// Media queries:
//   - Set QueryMediaType to "image", "video", or "audio"
//   - Provide either QueryMediaURL (for web URLs) or QueryMediaFile (for local files)
//   - QueryMediaURL takes precedence if both are specified
//
// Parameters:
//   - request: SearchQueryRequest containing query parameters, search options, and pagination settings
//
// Returns:
//   - SearchResponse with paginated results and metadata
//   - error if the search fails
//
// Example:
//
//	// Text-based search
//	response, err := client.Search.Query(&models.SearchQueryRequest{
//	    IndexID:       "your_index_id",
//	    QueryText:     "person running in park",
//	    SearchOptions: []string{"visual", "audio"},
//	    PageLimit:     10,
//	    SortOption:    "score",
//	})
//
//	// Image-based search
//	response, err := client.Search.Query(&models.SearchQueryRequest{
//	    IndexID:        "your_index_id",
//	    QueryMediaType: "image",
//	    QueryMediaURL:  "https://example.com/image.jpg",
//	    SearchOptions:  []string{"visual"},
//	})
func (sw *SearchWrapper) Query(ctx context.Context, request *models.SearchQueryRequest) (*models.SearchResponse, error) {
	// Use the existing SearchQueryRequest from search service
	results, err := sw.service.Query(ctx, request)
	if err != nil {
		return nil, errors.NewServiceError("Search", "search query failed: "+err.Error())
	}
	// Return the complete response directly (no need to wrap it again)
	return results, nil
}

// Create performs a search and returns the search ID for paginated result retrieval.
// This is an alias for Query() to maintain API compatibility.
func (sw *SearchWrapper) Create(ctx context.Context, request *models.SearchQueryRequest) (*models.SearchResponse, error) {
	return sw.Query(ctx, request)
}

// Retrieve gets paginated search results using a page token from a previous search response.
// Use this to navigate through large result sets efficiently.
//
// Parameters:
//   - pageToken: Page token from SearchResponse.PageInfo.NextPageToken
//
// Returns:
//   - SearchResponse with the next page of results
//   - error if retrieval fails
//
// Example:
//
//	// Initial search
//	response, err := client.Search.Query(&models.SearchQueryRequest{...})
//
//	// Get next page if available
//	if response.PageInfo.NextPageToken != "" {
//	    nextPage, err := client.Search.Retrieve(response.PageInfo.NextPageToken)
//	}
func (sw *SearchWrapper) Retrieve(ctx context.Context, pageToken string) (*models.SearchResponse, error) {
	return sw.service.Retrieve(ctx, pageToken)
}

// SearchByText is a convenience method for text-based semantic searches.
// This method simplifies common text search scenarios.
//
// Parameters:
//   - indexID: The ID of the index to search within
//   - queryText: The text query describing what to search for
//   - options: Search options like ["visual", "audio"] to specify search modalities
//
// Returns:
//   - SearchResponse with matching video segments
//   - error if the search fails
//
// Example:
//
//	results, err := client.Search.SearchByText(
//	    "your_index_id",
//	    "woman talking about cooking recipes",
//	    []string{"visual", "audio"},
//	)
//	for _, result := range results.Data {
//	    fmt.Printf("Found match in video %s at %fs\n", result.VideoID, result.Start)
//	}
func (sw *SearchWrapper) SearchByText(ctx context.Context, indexID, queryText string, options []string) (*models.SearchResponse, error) {
	request := &models.SearchRequest{
		IndexID:       indexID,
		QueryText:     queryText,
		SearchOptions: options,
	}
	return sw.Search(ctx, request)
}

// SearchByImage is a convenience method for image-based visual searches.
// Find video segments that are visually similar to the provided image.
//
// Parameters:
//   - indexID: The ID of the index to search within
//   - imageURL: Publicly accessible URL of the image to search for
//   - options: Search options, typically ["visual"] for image searches
//
// Returns:
//   - SearchResponse with visually similar video segments
//   - error if the search fails
//
// Example:
//
//	results, err := client.Search.SearchByImage(
//	    "your_index_id",
//	    "https://example.com/sample-image.jpg",
//	    []string{"visual"},
//	)
//	for _, result := range results.Data {
//	    fmt.Printf("Visual match: %s (confidence: %.2f)\n", result.VideoID, result.Score)
//	}
func (sw *SearchWrapper) SearchByImage(ctx context.Context, indexID, imageURL string, options []string) (*models.SearchResponse, error) {
	request := &models.SearchRequest{
		IndexID:        indexID,
		QueryMediaType: "image",
		QueryMediaURL:  imageURL,
		SearchOptions:  options,
	}
	return sw.Search(ctx, request)
}

// Search performs a search using the legacy SearchRequest format.
// For new applications, prefer using Query() or the convenience methods.
//
// Parameters:
//   - request: SearchRequest with query parameters
//
// Returns:
//   - SearchResponse with search results
//   - error if the search fails
func (sw *SearchWrapper) Search(ctx context.Context, request *models.SearchRequest) (*models.SearchResponse, error) {
	// Use the existing Search method from the base service
	results, err := sw.service.Search(ctx, request)
	if err != nil {
		return nil, errors.NewServiceError("Search", "search failed: "+err.Error())
	}
	return results, nil
}
