package wrappers

import (
	"fmt"
	"time"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/services"
)

// TasksWrapper wraps the basic TasksService with enhanced functionality
type TasksWrapper struct {
	service *services.TasksService
}

// NewTasksWrapper creates a new TasksWrapper
func NewTasksWrapper(service *services.TasksService) *TasksWrapper {
	return &TasksWrapper{service: service}
}

// Create creates a single video indexing task
func (tw *TasksWrapper) Create(request *models.TasksCreateRequest) (*models.Task, error) {
	return tw.service.Create(request)
}

// List retrieves tasks with optional filters
func (tw *TasksWrapper) List(filters map[string]string) ([]models.Task, error) {
	return tw.service.List(filters)
}

// Retrieve gets a specific task by ID
func (tw *TasksWrapper) Retrieve(taskID string) (*models.Task, error) {
	return tw.service.Retrieve(taskID)
}

// CreateBulkRequest represents the request for bulk task creation
type CreateBulkRequest struct {
	IndexID           string   `json:"index_id"`
	VideoFiles        []string `json:"video_files,omitempty"`
	VideoURLs         []string `json:"video_urls,omitempty"`
	EnableVideoStream bool     `json:"enable_video_stream,omitempty"`
}

// CreateBulk creates multiple video indexing tasks that upload and index videos in bulk.
// This method creates multiple video indexing tasks that upload and index videos in bulk.
// Ensure your videos meet the requirements in the Prerequisites section.
//
// Upload options:
// - Local files: Use the VideoFiles parameter to provide an array of file paths.
// - Publicly accessible URLs: Use the VideoURLs parameter to provide an array of URLs.
//
// Parameters:
//   - request: Request parameters containing indexId and either videoFiles or videoUrls
//
// Returns: A slice of video indexing tasks that were successfully created
//
// Example:
//
//	tasks, err := client.Tasks.CreateBulk(&CreateBulkRequest{
//	    IndexID: "index_id",
//	    VideoURLs: []string{"https://example.com/video1.mp4", "https://example.com/video2.mp4"},
//	})
func (tw *TasksWrapper) CreateBulk(request *CreateBulkRequest) ([]*models.Task, error) {
	if len(request.VideoFiles) == 0 && len(request.VideoURLs) == 0 {
		return nil, fmt.Errorf("either VideoFiles or VideoURLs must be provided")
	}

	var tasks []*models.Task

	// Process video files
	if len(request.VideoFiles) > 0 {
		for _, videoFile := range request.VideoFiles {
			taskRequest := &models.TasksCreateRequest{
				IndexID:           request.IndexID,
				VideoFile:         videoFile,
				EnableVideoStream: request.EnableVideoStream,
			}

			task, err := tw.service.Create(taskRequest)
			if err != nil {
				fmt.Printf("Error processing file %s: %v\n", videoFile, err)
				continue
			}
			tasks = append(tasks, task)
		}
	}

	// Process video URLs
	if len(request.VideoURLs) > 0 {
		for _, videoURL := range request.VideoURLs {
			taskRequest := &models.TasksCreateRequest{
				IndexID:           request.IndexID,
				VideoURL:          videoURL,
				EnableVideoStream: request.EnableVideoStream,
			}

			task, err := tw.service.Create(taskRequest)
			if err != nil {
				fmt.Printf("Error processing URL %s: %v\n", videoURL, err)
				continue
			}
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// WaitForDoneOptions represents options for the WaitForDone method
type WaitForDoneOptions struct {
	SleepInterval time.Duration
	Callback      func(*models.Task) error
}

// WaitForDone waits for a task to complete by periodically checking its status.
//
// Parameters:
//   - taskID: The unique identifier of the task to wait for
//   - options: Options for the wait operation including sleep interval and callback
//
// Returns: The completed task response
//
// Example:
//
//	task, err := client.Tasks.Create(&models.TasksCreateRequest{
//	    IndexID: "index_id",
//	    VideoURL: "https://example.com/video.mp4",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	completedTask, err := client.Tasks.WaitForDone(task.ID, &WaitForDoneOptions{
//	    SleepInterval: 10 * time.Second,
//	    Callback: func(task *models.Task) error {
//	        fmt.Printf("Current status: %s\n", task.Status)
//	        return nil
//	    },
//	})
func (tw *TasksWrapper) WaitForDone(taskID string, options *WaitForDoneOptions) (*models.Task, error) {
	if options == nil {
		options = &WaitForDoneOptions{}
	}

	sleepInterval := options.SleepInterval
	if sleepInterval <= 0 {
		sleepInterval = 5 * time.Second
	}

	callback := options.Callback

	// Get initial task
	task, err := tw.service.Retrieve(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve initial task: %w", err)
	}

	// Define done statuses
	doneStatuses := map[string]bool{
		"ready":  true,
		"failed": true,
	}

	// Continue checking until it's done
	for !doneStatuses[task.Status] {
		time.Sleep(sleepInterval)

		task, err = tw.service.Retrieve(taskID)
		if err != nil {
			fmt.Printf("Retrieving task failed: %v. Retrying...\n", err)
			continue
		}

		// Call callback if provided
		if callback != nil {
			if err := callback(task); err != nil {
				return nil, fmt.Errorf("callback error: %w", err)
			}
		}
	}

	return task, nil
}
