package services

import (
	"fmt"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
)

type IndexesService struct {
	Client ClientInterface
}

func (s *IndexesService) List(filters map[string]string) ([]models.Index, error) {
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

	req, err := s.Client.NewRequest("GET", url, nil)
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

func (s *IndexesService) Create(reqBody *models.IndexCreateRequest) (*models.Index, error) {
	req, err := s.Client.NewRequest("POST", "/indexes", reqBody)
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

func (s *IndexesService) Retrieve(id string) (*models.Index, error) {
	path := fmt.Sprintf("/indexes/%s", id)
	req, err := s.Client.NewRequest("GET", path, nil)
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

func (s *IndexesService) Update(indexID string, reqBody *models.IndexUpdateRequest) (*models.Index, error) {
	path := fmt.Sprintf("/indexes/%s", indexID)
	req, err := s.Client.NewRequest("PUT", path, reqBody)
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

func (s *IndexesService) Delete(indexID string) error {
	path := fmt.Sprintf("/indexes/%s", indexID)
	req, err := s.Client.NewRequest("DELETE", path, nil)
	if err != nil {
		return err
	}

	_, err = s.Client.Do(req, nil)
	return err
}

// Video management within indexes
func (s *IndexesService) ListVideos(indexID string, filters map[string]string) ([]models.Video, error) {
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

	req, err := s.Client.NewRequest("GET", url, nil)
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

func (s *IndexesService) RetrieveVideo(indexID, videoID string) (*models.Video, error) {
	path := fmt.Sprintf("/indexes/%s/videos/%s", indexID, videoID)
	req, err := s.Client.NewRequest("GET", path, nil)
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

func (s *IndexesService) UpdateVideo(indexID, videoID string, reqBody *models.VideoUpdateRequest) (*models.Video, error) {
	path := fmt.Sprintf("/indexes/%s/videos/%s", indexID, videoID)
	req, err := s.Client.NewRequest("PUT", path, reqBody)
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

func (s *IndexesService) DeleteVideo(indexID, videoID string) error {
	path := fmt.Sprintf("/indexes/%s/videos/%s", indexID, videoID)
	req, err := s.Client.NewRequest("DELETE", path, nil)
	if err != nil {
		return err
	}

	_, err = s.Client.Do(req, nil)
	return err
}
