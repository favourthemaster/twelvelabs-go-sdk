package services

import (
	"context"
	"fmt"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
)

type IndexesService struct {
	Client ClientInterface
}

func (s *IndexesService) List(ctx context.Context, filters map[string]string) ([]models.Index, error) {
	queryParams := ""
	for key, value := range filters {
		if queryParams != "" {
			queryParams += "&"
		}
		queryParams += fmt.Sprintf("%s=%s", key, value)
	}

	url := "/indexes"
	if queryParams != "" {
		url += "?" + queryParams
	}

	req, err := s.Client.NewRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []models.Index `json:"data"`
	}
	_, err = s.Client.Do(req, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (s *IndexesService) Create(ctx context.Context, reqBody *models.IndexCreateRequest) (*models.Index, error) {
	req, err := s.Client.NewRequest(ctx, "POST", "/indexes", reqBody)
	if err != nil {
		return nil, err
	}

	var index models.Index
	_, err = s.Client.Do(req, &index)
	if err != nil {
		return nil, err
	}

	return &index, nil
}

func (s *IndexesService) Retrieve(ctx context.Context, id string) (*models.Index, error) {
	path := fmt.Sprintf("/indexes/%s", id)
	req, err := s.Client.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var index models.Index
	_, err = s.Client.Do(req, &index)
	if err != nil {
		return nil, err
	}

	return &index, nil
}

func (s *IndexesService) Update(ctx context.Context, indexID string, reqBody *models.IndexUpdateRequest) (*models.Index, error) {
	path := fmt.Sprintf("/indexes/%s", indexID)
	req, err := s.Client.NewRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return nil, err
	}

	var index models.Index
	_, err = s.Client.Do(req, &index)
	if err != nil {
		return nil, err
	}

	return &index, nil
}

func (s *IndexesService) Delete(ctx context.Context, indexID string) error {
	path := fmt.Sprintf("/indexes/%s", indexID)
	req, err := s.Client.NewRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	_, err = s.Client.Do(req, nil)
	return err
}

// Video management within indexes
func (s *IndexesService) ListVideos(ctx context.Context, indexID string, filters map[string]string) ([]models.Video, error) {
	queryParams := ""
	for key, value := range filters {
		if queryParams != "" {
			queryParams += "&"
		}
		queryParams += fmt.Sprintf("%s=%s", key, value)
	}

	url := fmt.Sprintf("/indexes/%s/videos", indexID)
	if queryParams != "" {
		url += "?" + queryParams
	}

	req, err := s.Client.NewRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []models.Video `json:"data"`
	}
	_, err = s.Client.Do(req, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (s *IndexesService) RetrieveVideo(ctx context.Context, indexID, videoID string) (*models.Video, error) {
	path := fmt.Sprintf("/indexes/%s/videos/%s", indexID, videoID)
	req, err := s.Client.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var video models.Video
	_, err = s.Client.Do(req, &video)
	if err != nil {
		return nil, err
	}

	return &video, nil
}

func (s *IndexesService) UpdateVideo(ctx context.Context, indexID, videoID string, reqBody *models.VideoUpdateRequest) (*models.Video, error) {
	path := fmt.Sprintf("/indexes/%s/videos/%s", indexID, videoID)
	req, err := s.Client.NewRequest(ctx, "PUT", path, reqBody)
	if err != nil {
		return nil, err
	}

	var video models.Video
	_, err = s.Client.Do(req, &video)
	if err != nil {
		return nil, err
	}

	return &video, nil
}

func (s *IndexesService) DeleteVideo(ctx context.Context, indexID, videoID string) error {
	path := fmt.Sprintf("/indexes/%s/videos/%s", indexID, videoID)
	req, err := s.Client.NewRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	_, err = s.Client.Do(req, nil)
	return err
}
