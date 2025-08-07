package twelvelabs

import (
	"fmt"
)

type IndexesService struct {
	client *Client
}

func (s *IndexesService) List() ([]Index, error) {
	req, err := s.client.newRequest("GET", "/indexes", nil)
	if err != nil {
		return nil, err
	}

	var indexes []Index
	_, err = s.client.do(req, &indexes)
	if err != nil {
		return nil, err
	}

	return indexes, nil
}

func (s *IndexesService) Create(reqBody *IndexesCreateRequest) (*Index, error) {
	req, err := s.client.newRequest("POST", "/indexes", reqBody)
	if err != nil {
		return nil, err
	}

	var index Index
	_, err = s.client.do(req, &index)
	if err != nil {
		return nil, err
	}

	return &index, nil
}

func (s *IndexesService) Retrieve(id string) (*Index, error) {
	path := fmt.Sprintf("/indexes/%s", id)
	req, err := s.client.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var index Index
	_, err = s.client.do(req, &index)
	if err != nil {
		return nil, err
	}

	return &index, nil
}

func (s *IndexesService) Delete(id string) error {
	path := fmt.Sprintf("/indexes/%s", id)
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
