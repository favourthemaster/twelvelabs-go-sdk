package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
)

type SearchService struct {
	Client ClientInterface
}

func (s *SearchService) Query(ctx context.Context, reqBody *models.SearchQueryRequest) (*models.SearchResponse, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add required fields
	if err := w.WriteField("index_id", reqBody.IndexID); err != nil {
		return nil, fmt.Errorf("failed to write index_id field: %w", err)
	}

	// Add optional query fields
	if reqBody.QueryText != "" {
		if err := w.WriteField("query_text", reqBody.QueryText); err != nil {
			return nil, fmt.Errorf("failed to write query_text field: %w", err)
		}
	}

	if reqBody.QueryMediaType != "" {
		if err := w.WriteField("query_media_type", reqBody.QueryMediaType); err != nil {
			return nil, fmt.Errorf("failed to write query_media_type field: %w", err)
		}
	}

	if reqBody.QueryMediaURL != "" {
		if err := w.WriteField("query_media_url", reqBody.QueryMediaURL); err != nil {
			return nil, fmt.Errorf("failed to write query_media_url field: %w", err)
		}
	}

	// Handle file upload if provided
	if reqBody.QueryMediaFile != "" {
		file, err := os.Open(reqBody.QueryMediaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open query media file: %w", err)
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				fmt.Printf("failed to close query media file: %v\n", err)
			}
		}(file)

		part, err := w.CreateFormFile("query_media_file", reqBody.QueryMediaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}

		if _, err = io.Copy(part, file); err != nil {
			return nil, fmt.Errorf("failed to copy file content: %w", err)
		}
	}

	// Add search options
	for _, option := range reqBody.SearchOptions {
		if err := w.WriteField("search_options", option); err != nil {
			return nil, fmt.Errorf("failed to write search_options field: %w", err)
		}
	}

	err := w.Close()
	if err != nil {
		return nil, err
	}

	req, err := s.Client.NewRequest(ctx, "POST", "/search", &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	var response models.SearchResponse
	_, err = s.Client.Do(req, &response)
	if err != nil {
		return nil, err
	}

	// Add debugging to see what we actually received
	fmt.Printf("DEBUG: Raw response: %+v\n", response)
	fmt.Printf("DEBUG: Data length: %d\n", len(response.Data))
	if response.SearchPool != nil {
		fmt.Printf("DEBUG: SearchPool: %+v\n", *response.SearchPool)
	} else {
		fmt.Printf("DEBUG: SearchPool is nil\n")
	}
	if response.PageInfo != nil {
		fmt.Printf("DEBUG: PageInfo: %+v\n", *response.PageInfo)
	} else {
		fmt.Printf("DEBUG: PageInfo is nil\n")
	}

	return &response, nil // Return the complete response, not just response.Data
}

func (s *SearchService) Search(ctx context.Context, request *models.SearchRequest) (*models.SearchResponse, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add required fields
	if err := w.WriteField("index_id", request.IndexID); err != nil {
		return nil, fmt.Errorf("failed to write index_id field: %w", err)
	}

	// Add optional query fields
	if request.QueryText != "" {
		if err := w.WriteField("query_text", request.QueryText); err != nil {
			return nil, fmt.Errorf("failed to write query_text field: %w", err)
		}
	}

	if request.QueryMediaType != "" {
		if err := w.WriteField("query_media_type", request.QueryMediaType); err != nil {
			return nil, fmt.Errorf("failed to write query_media_type field: %w", err)
		}
	}

	if request.QueryMediaURL != "" {
		if err := w.WriteField("query_media_url", request.QueryMediaURL); err != nil {
			return nil, fmt.Errorf("failed to write query_media_url field: %w", err)
		}
	}

	// Handle file upload if provided
	if request.QueryMediaFile != "" {
		file, err := os.Open(request.QueryMediaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open query media file: %w", err)
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				fmt.Printf("failed to close query media file: %v\n", err)
			}
		}(file)

		part, err := w.CreateFormFile("query_media_file", request.QueryMediaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}

		if _, err = io.Copy(part, file); err != nil {
			return nil, fmt.Errorf("failed to copy file content: %w", err)
		}
	}

	// Add search options
	for _, option := range request.SearchOptions {
		if err := w.WriteField("search_options", option); err != nil {
			return nil, fmt.Errorf("failed to write search_options field: %w", err)
		}
	}

	// Add other optional fields
	if request.PageLimit > 0 {
		if err := w.WriteField("page_limit", fmt.Sprintf("%d", request.PageLimit)); err != nil {
			return nil, fmt.Errorf("failed to write page_limit field: %w", err)
		}
	}

	err := w.Close()
	if err != nil {
		return nil, err
	}

	req, err := s.Client.NewRequest(ctx, "POST", "/search", &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	var response models.SearchResponse
	_, err = s.Client.Do(req, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (s *SearchService) Retrieve(ctx context.Context, pageToken string) (*models.SearchResponse, error) {
	queryParams := ""
	if pageToken != "" {
		queryParams += fmt.Sprintf("page_token=%s", pageToken)
	}

	url := fmt.Sprintf("/search/%s", pageToken)
	if queryParams != "" {
		url += "?" + queryParams
	}

	req, err := s.Client.NewRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response models.SearchResponse
	_, err = s.Client.Do(req, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
