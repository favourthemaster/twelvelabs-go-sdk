package twelvelabs

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"time"
)

type TasksService struct {
	client *Client
}

func (s *TasksService) List() ([]Task, error) {
	req, err := s.client.newRequest("GET", "/tasks", nil)
	if err != nil {
		return nil, err
	}

	var tasks []Task
	_, err = s.client.do(req, &tasks)
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func (s *TasksService) Create(reqBody *TasksCreateRequest) (*Task, error) {
	// Handle multipart/form-data for file upload
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// Add index_id field
	if err := w.WriteField("index_id", reqBody.IndexID); err != nil {
		return nil, fmt.Errorf("failed to write index_id field: %w", err)
	}

	// Add video_file field
	file, err := os.Open(reqBody.VideoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open video file: %w", err)
	}
	defer file.Close()

	part, err := w.CreateFormFile("video_file", reqBody.VideoFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err = io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}
	w.Close()

	req, err := s.client.newRequest("POST", "/tasks", &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	var task Task
	_, err = s.client.do(req, &task)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (s *TasksService) Retrieve(id string) (*Task, error) {
	path := fmt.Sprintf("/tasks/%s", id)
	req, err := s.client.newRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var task Task
	_, err = s.client.do(req, &task)
	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (s *TasksService) Delete(id string) error {
	path := fmt.Sprintf("/tasks/%s", id)
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

func (s *TasksService) WaitForDone(id string, interval time.Duration, callback func(*Task)) (*Task, error) {
	for {
		task, err := s.Retrieve(id)
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
