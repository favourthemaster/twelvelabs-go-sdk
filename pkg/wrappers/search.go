package wrappers

import (
	"fmt"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/services"
)

// SearchWrapper wraps the basic SearchService with additional functionality
type SearchWrapper struct {
	service *services.SearchService
}

// NewSearchWrapper creates a new SearchWrapper
func NewSearchWrapper(service *services.SearchService) *SearchWrapper {
	return &SearchWrapper{service: service}
}

// Query performs a search query with enhanced functionality similar to the Node.js SDK.
// This method searches for relevant matches in an index using text or various media queries.
//
// Text queries:
// - Use the QueryText parameter to specify your query.
//
// Media queries:
// - Set the QueryMediaType parameter to the corresponding media type (example: "image").
// - Specify either one of the following parameters:
//   - QueryMediaURL: Publicly accessible URL of your media file.
//   - QueryMediaFile: Local media file.
//     If both QueryMediaURL and QueryMediaFile are specified, QueryMediaURL takes precedence.
//
// Parameters:
//   - request: Search request containing query parameters
//
// Returns: A SearchResponse with paginated results
//
// Example:
//
//	response, err := client.Search.Query(&models.SearchQueryRequest{
//	    IndexID: "index_id",
//	    QueryText: "person running",
//	    SearchOptions: []string{"visual"},
//	})
func (sw *SearchWrapper) Query(request *models.SearchQueryRequest) (*models.SearchResponse, error) {
	// Use the existing SearchQueryRequest from search service
	results, err := sw.service.Query(request)
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}

	// Convert results to SearchResponse format
	return &models.SearchResponse{
		Data: results,
	}, nil
}

// Create performs a search and returns the search ID for further operations
func (sw *SearchWrapper) Create(request *models.SearchQueryRequest) (*models.SearchResponse, error) {
	return sw.Query(request)
}

// Retrieve gets search results by search ID with pagination support
func (sw *SearchWrapper) Retrieve(searchID string, pageToken string, includeUserMetadata bool) (*models.SearchResponse, error) {
	return sw.service.Retrieve(searchID, pageToken, includeUserMetadata)
}

// SearchByText is a convenience method for text-based searches
func (sw *SearchWrapper) SearchByText(indexID, queryText string, options []string) (*models.SearchResponse, error) {
	request := &models.SearchQueryRequest{
		IndexID:       indexID,
		QueryText:     queryText,
		SearchOptions: options,
	}
	return sw.Query(request)
}

// SearchByImage is a convenience method for image-based searches
func (sw *SearchWrapper) SearchByImage(indexID, imageURL string, options []string) (*models.SearchResponse, error) {
	request := &models.SearchQueryRequest{
		IndexID:        indexID,
		QueryMediaType: "image",
		QueryMediaURL:  imageURL,
		SearchOptions:  options,
	}
	return sw.Query(request)
}

// SearchByVideo is a convenience method for video-based searches
func (sw *SearchWrapper) SearchByVideo(indexID, videoURL string, options []string) (*models.SearchResponse, error) {
	request := &models.SearchQueryRequest{
		IndexID:        indexID,
		QueryMediaType: "video",
		QueryMediaURL:  videoURL,
		SearchOptions:  options,
	}
	return sw.Query(request)
}
