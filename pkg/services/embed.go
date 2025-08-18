package services

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

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

	// Add audio_url field if provided
	if reqBody.AudioURL != "" {
		if err := w.WriteField("audio_url", reqBody.AudioURL); err != nil {
			return nil, fmt.Errorf("failed to write audio_url field: %w", err)
		}
	}

	if reqBody.AudioFile != "" {
		file, err := os.Open(reqBody.AudioFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open audio file: %w", err)
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				fmt.Printf("failed to close audio file: %v\n", err)
			}
		}(file)

		part, err := w.CreateFormFile("audio_file", reqBody.AudioFile)
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

	path := "/embed"
	if reqBody.VideoFile != "" || reqBody.VideoURL != "" {
		path = "/embed/tasks"
	}

	req, err := s.Client.NewRequest("POST", path, &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	if reqBody.VideoFile != "" || reqBody.VideoURL != "" {
		var data map[string]interface{}
		_, err = s.Client.Do(req, &data)
		if err != nil {
			return nil, err
		}
		if embedID, ok := data["_id"].(string); ok {
			embedResponse, err := s.WaitForEmbedTask(embedID, 10*time.Second, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to wait for embed task: %w", err)
			}
			return embedResponse, nil
		}
	}

	var embedResponse models.EmbedResponse
	_, err = s.Client.Do(req, &embedResponse)
	if err != nil {
		return nil, err
	}

	return &embedResponse, nil
}

func (s *EmbedService) WaitForEmbedTask(taskID string, interval time.Duration, callback func(status models.EmbedTaskStatus)) (*models.EmbedResponse, error) {
	for {
		req, err := s.Client.NewRequest("GET", fmt.Sprintf("/embed/tasks/%s/status", taskID), nil)
		if err != nil {
			return nil, err
		}

		var status models.EmbedTaskStatus
		resp, err := s.Client.Do(req, &status)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get task status: %s", resp.Status)
		}

		if status.Status == "ready" {
			embedReq, err := s.Client.NewRequest("GET", fmt.Sprintf("/embed/tasks/%s", taskID), nil)
			if err != nil {
				return nil, err
			}

			var embedResponse models.EmbedResponse
			_, err = s.Client.Do(embedReq, &embedResponse)
			if err != nil {
				return nil, err
			}

			return &embedResponse, nil
		} else if status.Status == "failed" {
			return nil, fmt.Errorf("embed task failed with status: %s", status.Status)
		}

		if callback != nil {
			callback(status)
		}

		func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				fmt.Printf("failed to close response body: %v\n", err)
			}
		}(resp.Body)

		time.Sleep(interval)
	}
}
