package wrappers

import (
	"fmt"
	"time"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/errors"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/services"
)

// TasksWrapper provides enhanced task management capabilities for video upload and processing,
// including bulk operations, progress tracking, and completion waiting with callbacks.
type TasksWrapper struct {
	service *services.TasksService
}

// NewTasksWrapper creates a new TasksWrapper instance.
func NewTasksWrapper(service *services.TasksService) *TasksWrapper {
	return &TasksWrapper{service: service}
}

// Create creates a single video indexing task for uploading and processing a video.
// Supports both local files and publicly accessible URLs.
//
// Parameters:
//   - request: TasksCreateRequest with IndexID and either VideoFile or VideoURL
//
// Returns:
//   - Task containing the task ID and initial status
//   - error if task creation fails
//
// Example:
//
//	// Upload local file
//	task, err := client.Tasks.Create(&models.TasksCreateRequest{
//	    IndexID:   "your_index_id",
//	    VideoFile: "./videos/sample.mp4",
//	})
//
//	// Upload from URL
//	task, err := client.Tasks.Create(&models.TasksCreateRequest{
//	    IndexID:  "your_index_id",
//	    VideoURL: "https://example.com/video.mp4",
//	})
func (tw *TasksWrapper) Create(request *models.TasksCreateRequest) (*models.Task, error) {
	return tw.service.Create(request)
}

// List retrieves tasks with optional filtering by status, index ID, or other criteria.
//
// Parameters:
//   - filters: Map of filter criteria (e.g., {"status": "completed", "index_id": "your_index"})
//
// Returns:
//   - Array of Task objects matching the filter criteria
//   - error if retrieval fails
//
// Example:
//
//	// Get all completed tasks
//	tasks, err := client.Tasks.List(map[string]string{
//	    "status": "completed",
//	})
//
//	// Get tasks for specific index
//	tasks, err := client.Tasks.List(map[string]string{
//	    "index_id": "your_index_id",
//	})
func (tw *TasksWrapper) List(filters map[string]string) ([]models.Task, error) {
	return tw.service.List(filters)
}

// Retrieve gets detailed information about a specific task by its ID.
//
// Parameters:
//   - taskID: The unique identifier of the task
//
// Returns:
//   - Task object with current status and metadata
//   - error if task not found or retrieval fails
//
// Example:
//
//	task, err := client.Tasks.Retrieve("task_id_here")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Task status: %s\n", task.Status)
func (tw *TasksWrapper) Retrieve(taskID string) (*models.Task, error) {
	return tw.service.Retrieve(taskID)
}

// CreateBulkRequest represents a request for creating multiple video indexing tasks simultaneously.
// This enables efficient batch processing of multiple videos.
type CreateBulkRequest struct {
	// IndexID is the target index for all videos
	IndexID string `json:"index_id"`
	// VideoFiles contains paths to local video files (optional)
	VideoFiles []string `json:"video_files,omitempty"`
	// VideoURLs contains publicly accessible video URLs (optional)
	VideoURLs []string `json:"video_urls,omitempty"`
	// EnableVideoStream enables video streaming for processed content (optional)
	EnableVideoStream bool `json:"enable_video_stream,omitempty"`
}

// CreateBulk creates multiple video indexing tasks for batch processing of videos.
// This method efficiently handles bulk video uploads and processing.
//
// Upload options:
//   - Local files: Use VideoFiles to provide an array of file paths
//   - Publicly accessible URLs: Use VideoURLs to provide an array of URLs
//   - Mixed sources: Can combine both local files and URLs in a single request
//
// Parameters:
//   - request: CreateBulkRequest with IndexID and video sources
//
// Returns:
//   - Array of Task objects, one for each video
//   - error if bulk task creation fails
//
// Example:
//
//	tasks, err := client.Tasks.CreateBulk(&wrappers.CreateBulkRequest{
//	    IndexID: "your_index_id",
//	    VideoFiles: []string{
//	        "./videos/video1.mp4",
//	        "./videos/video2.mp4",
//	    },
//	    VideoURLs: []string{
//	        "https://example.com/video3.mp4",
//	        "https://example.com/video4.mp4",
//	    },
//	    EnableVideoStream: true,
//	})
//	fmt.Printf("Created %d tasks\n", len(tasks))
func (tw *TasksWrapper) CreateBulk(request *CreateBulkRequest) ([]models.Task, error) {
	if len(request.VideoFiles) == 0 && len(request.VideoURLs) == 0 {
		return nil, errors.NewValidationError("either VideoFiles or VideoURLs must be provided")
	}

	var tasks []models.Task

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
			tasks = append(tasks, *task)
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
			tasks = append(tasks, *task)
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
		return nil, errors.NewServiceError("Tasks", "failed to retrieve initial task: "+err.Error())
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
				return nil, errors.NewServiceError("Tasks", "callback error: "+err.Error())
			}
		}
	}

	return task, nil
}

// WaitForCompletion waits for a task to complete, calling the optional callback function
// with status updates. This method polls the task status at regular intervals.
//
// Parameters:
//   - taskID: The task ID to monitor
//   - callback: Optional function called with each status update (can be nil)
//
// Returns:
//   - error if the task fails or polling encounters an error
//
// Example:
//
//	err := client.Tasks.WaitForCompletion("task_id", func(status string) {
//	    fmt.Printf("Task status: %s\n", status)
//	})
//	if err != nil {
//	    log.Printf("Task failed: %v", err)
//	} else {
//	    fmt.Println("Task completed successfully!")
//	}
func (tw *TasksWrapper) WaitForCompletion(taskID string, callback func(string)) error {
	// Get initial task
	task, err := tw.service.Retrieve(taskID)
	if err != nil {
		return errors.NewServiceError("Tasks", "failed to retrieve initial task: "+err.Error())
	}

	// Define done statuses
	doneStatuses := map[string]bool{
		"ready":  true,
		"failed": true,
	}

	// Continue checking until it's done
	for !doneStatuses[task.Status] {
		time.Sleep(5 * time.Second)

		task, err = tw.service.Retrieve(taskID)
		if err != nil {
			return errors.NewServiceError("Tasks", "retrieving task failed: "+err.Error())
		}

		// Call callback if provided
		if callback != nil {
			callback(task.Status)
		}
	}

	return nil
}

// WaitForCompletionWithTimeout waits for task completion with a specified timeout.
// This prevents indefinite waiting for stuck or very long-running tasks.
//
// Parameters:
//   - taskID: The task ID to monitor
//   - timeout: Maximum time to wait for completion
//   - callback: Optional function called with status updates
//
// Returns:
//   - error if timeout exceeded, task fails, or polling encounters an error
//
// Example:
//
//	err := client.Tasks.WaitForCompletionWithTimeout(
//	    "task_id",
//	    10*time.Minute,
//	    func(status string) {
//	        fmt.Printf("Status: %s\n", status)
//	    },
//	)
func (tw *TasksWrapper) WaitForCompletionWithTimeout(taskID string, timeout time.Duration, callback func(string)) error {
	// Get initial task
	task, err := tw.service.Retrieve(taskID)
	if err != nil {
		return errors.NewServiceError("Tasks", "failed to retrieve initial task: "+err.Error())
	}

	// Define done statuses
	doneStatuses := map[string]bool{
		"ready":  true,
		"failed": true,
	}

	// Set a deadline for completion
	deadline := time.Now().Add(timeout)

	// Continue checking until it's done or timeout
	for !doneStatuses[task.Status] {
		if time.Now().After(deadline) {
			return errors.NewTimeoutError("timeout exceeded while waiting for task completion")
		}

		time.Sleep(5 * time.Second)

		task, err = tw.service.Retrieve(taskID)
		if err != nil {
			return errors.NewServiceError("Tasks", "retrieving task failed: "+err.Error())
		}

		// Call callback if provided
		if callback != nil {
			callback(task.Status)
		}
	}

	return nil
}
