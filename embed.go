package twelvelabs

type EmbedService struct {
	client *Client
}

func (s *EmbedService) Create(reqBody *EmbedRequest) (*EmbedResponse, error) {
	req, err := s.client.newRequest("POST", "/embed", reqBody)
	if err != nil {
		return nil, err
	}

	var embedResponse EmbedResponse
	_, err = s.client.do(req, &embedResponse)
	if err != nil {
		return nil, err
	}

	return &embedResponse, nil
}
