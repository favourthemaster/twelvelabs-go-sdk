package wrappers

import (
	"context"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/services"
)

// IndexesWrapper wraps the basic IndexesService with additional functionality
type IndexesWrapper struct {
	service *services.IndexesService
	Videos  *IndexesVideosWrapper
}

// NewIndexesWrapper creates a new IndexesWrapper with the IndexesVideosWrapper
func NewIndexesWrapper(service *services.IndexesService) *IndexesWrapper {
	return &IndexesWrapper{
		service: service,
		Videos:  NewIndexesVideosWrapper(service),
	}
}

// Create creates a new index
func (iw *IndexesWrapper) Create(ctx context.Context, request *models.IndexCreateRequest) (*models.Index, error) {
	return iw.service.Create(ctx, request)
}

// List retrieves all indexes with optional filters
func (iw *IndexesWrapper) List(ctx context.Context, filters map[string]string) ([]models.Index, error) {
	return iw.service.List(ctx, filters)
}

// Retrieve gets a specific index by ID
func (iw *IndexesWrapper) Retrieve(ctx context.Context, indexID string) (*models.Index, error) {
	return iw.service.Retrieve(ctx, indexID)
}

// Update updates an existing index
func (iw *IndexesWrapper) Update(ctx context.Context, indexID string, request *models.IndexUpdateRequest) (*models.Index, error) {
	return iw.service.Update(ctx, indexID, request)
}

// Delete deletes an index
func (iw *IndexesWrapper) Delete(ctx context.Context, indexID string) error {
	return iw.service.Delete(ctx, indexID)
}

// IndexesVideosWrapper wraps video operations within an index context
type IndexesVideosWrapper struct {
	service *services.IndexesService
}

// NewIndexesVideosWrapper creates a new IndexesVideosWrapper
func NewIndexesVideosWrapper(service *services.IndexesService) *IndexesVideosWrapper {
	return &IndexesVideosWrapper{
		service: service,
	}
}

// List retrieves videos in an index with optional filters
func (ivw *IndexesVideosWrapper) List(ctx context.Context, indexID string, filters map[string]string) ([]models.Video, error) {
	return ivw.service.ListVideos(ctx, indexID, filters)
}

// Retrieve gets a specific video in an index
func (ivw *IndexesVideosWrapper) Retrieve(ctx context.Context, indexID, videoID string) (*models.Video, error) {
	return ivw.service.RetrieveVideo(ctx, indexID, videoID)
}

// Update updates a video in an index
func (ivw *IndexesVideosWrapper) Update(ctx context.Context, indexID, videoID string, request *models.VideoUpdateRequest) (*models.Video, error) {
	return ivw.service.UpdateVideo(ctx, indexID, videoID, request)
}

// Delete deletes a video from an index
func (ivw *IndexesVideosWrapper) Delete(ctx context.Context, indexID, videoID string) error {
	return ivw.service.DeleteVideo(ctx, indexID, videoID)
}
