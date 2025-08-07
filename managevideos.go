package twelvelabs

import (
	"fmt"
)

type ManageVideosService struct {
	client *Client
}

func (s *ManageVideosService) Retrieve(id string) (*Video, error) {
	path := fmt.Sprintf("/videos/%s", id)
	req, err := s.client.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var video Video
	_, err = s.client.do(req, &video)
	if err != nil {
		return nil, err
	}

	return &video, nil
}

func (s *ManageVideosService) Delete(id string) error {
	path := fmt.Sprintf("/videos/%s", id)
	req, err := s.client.newRequest("DELETE", path, nil)
	if err != nil {
		return err
	}

	_, err = s.client.do(req, nil)
	if err != nil {
		return err
	}

	return nil
}
