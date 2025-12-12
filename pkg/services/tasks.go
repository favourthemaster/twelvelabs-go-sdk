package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"time"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
)

type TasksService struct {
	Client ClientInterface
}

func (s *TasksService) List(ctx context.Context, filters map[string]string) ([]models.Task, error) {
	queryParams := ""
	for key, value := range filters {
		if queryParams != "" {
			queryParams += "&"
		}
		queryParams += fmt.Sprintf("%s=%s", key, value)
	}

	url := "/tasks"
	if queryParams != "" {
		url += "?" + queryParams
	}

	req, err := s.Client.NewRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Data []models.Task `json:"data"`
	}
	_, err = s.Client.Do(req, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (s *TasksService) Create(ctx context.Context, reqBody *models.TasksCreateRequest) (*models.Task, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add index_id field
	if err := w.WriteField("index_id", reqBody.IndexID); err != nil {
		return nil, fmt.Errorf("failed to write index_id field: %w", err)
	}

	// Add video_file field if provided
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

	// Add video_url field if provided
	if reqBody.VideoURL != "" {
		if err := w.WriteField("video_url", reqBody.VideoURL); err != nil {
			return nil, fmt.Errorf("failed to write video_url field: %w", err)
		}
	}

	// Add additional optional fields...
	if reqBody.EnableVideoStream {
		if err := w.WriteField("enable_video_stream", "true"); err != nil {
			return nil, fmt.Errorf("failed to write enable_video_stream field: %w", err)
		}
	}

	err := w.Close()
	if err != nil {
		return nil, err
	}

	req, err := s.Client.NewRequest(ctx, "POST", "/tasks", &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	var task models.Task
	_, err = s.Client.Do(req, &task)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (s *TasksService) Retrieve(ctx context.Context, id string) (*models.Task, error) {
	path := fmt.Sprintf("/tasks/%s", id)
	req, err := s.Client.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var task models.Task
	_, err = s.Client.Do(req, &task)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (s *TasksService) Delete(ctx context.Context, id string) error {
	path := fmt.Sprintf("/tasks/%s", id)
	req, err := s.Client.NewRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}

	_, err = s.Client.Do(req, nil)
	return err
}

func (s *TasksService) WaitForDone(ctx context.Context, id string, interval time.Duration, callback func(*models.Task)) (*models.Task, error) {
	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		task, err := s.Retrieve(ctx, id)
		if err != nil {
			return nil, err
		}

		if callback != nil {
			callback(task)
		}

		if task.Status == "ready" || task.Status == "failed" || task.Status == "error" {
			return task, nil
		}

		time.Sleep(interval)
	}
}
