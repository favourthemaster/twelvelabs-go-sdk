package twelvelabs

type SearchService struct {
	client *Client
}

func (s *SearchService) Query(reqBody *SearchQueryRequest) ([]SearchResult, error) {
	req, err := s.client.newRequest("POST", "/search", reqBody)
	if err != nil {
		return nil, err
	}

	var searchResults []SearchResult
	_, err = s.client.do(req, &searchResults)
	if err != nil {
		return nil, err
	}

	return searchResults, nil
}
