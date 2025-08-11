package services

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
)

type ClientInterface interface {
	NewRequest(method, path string, body interface{}) (*http.Request, error)
	Do(req *http.Request, v interface{}) (*http.Response, error)
	DoRaw(req *http.Request) (*http.Response, error)
}

type EmbedService struct {
	Client ClientInterface
}

func (s *EmbedService) Create(reqBody *models.EmbedRequest) (*models.EmbedResponse, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add model_name field
	if err := w.WriteField("model_name", reqBody.ModelName); err != nil {
		return nil, fmt.Errorf("failed to write model_name field: %w", err)
	}

	// Add text field if provided
	if reqBody.Text != "" {
		if err := w.WriteField("text", reqBody.Text); err != nil {
			return nil, fmt.Errorf("failed to write text field: %w", err)
		}
	}

	// Add image_url field if provided
	if reqBody.ImageURL != "" {
		if err := w.WriteField("image_url", reqBody.ImageURL); err != nil {
			return nil, fmt.Errorf("failed to write image_url field: %w", err)
		}
	}

	// Add image_file field if provided
	if reqBody.ImageFile != "" {
		file, err := os.Open(reqBody.ImageFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open image file: %w", err)
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				fmt.Printf("failed to close image file: %v\n", err)
			}
		}(file)

		part, err := w.CreateFormFile("image_file", reqBody.ImageFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}

		if _, err = io.Copy(part, file); err != nil {
			return nil, fmt.Errorf("failed to copy file content: %w", err)
		}
	}

	// Add video fields if provided
	if reqBody.VideoURL != "" {
		if err := w.WriteField("video_url", reqBody.VideoURL); err != nil {
			return nil, fmt.Errorf("failed to write video_url field: %w", err)
		}
	}

	if reqBody.VideoFile != "" {
		file, err := os.Open(reqBody.VideoFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open video file: %w", err)
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				fmt.Printf("failed to close video file: %v\n", err)
			}
		}(file)

		part, err := w.CreateFormFile("video_file", reqBody.VideoFile)
		if err != nil {
			return nil, fmt.Errorf("failed to create form file: %w", err)
		}

		if _, err = io.Copy(part, file); err != nil {
			return nil, fmt.Errorf("failed to copy file content: %w", err)
		}
	}

	err := w.Close()
	if err != nil {
		return nil, err
	}

	req, err := s.Client.NewRequest("POST", "/embed", &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	var embedResponse models.EmbedResponse
	_, err = s.Client.Do(req, &embedResponse)
	if err != nil {
		return nil, err
	}

	return &embedResponse, nil
}
