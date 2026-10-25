package twelvelabs

import (
	"context"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Legacy video indexing task status values.
//
// Deprecated: the /tasks endpoints are superseded by assets and indexed assets.
const (
	TaskStatusValidating = "validating"
	TaskStatusPending    = "pending"
	TaskStatusQueued     = "queued"
	TaskStatusIndexing   = "indexing"
	TaskStatusReady      = "ready"
	TaskStatusFailed     = "failed"
)

// TasksService manages legacy video indexing tasks.
//
// Deprecated: TwelveLabs replaced /tasks with /assets plus
// /indexes/{index-id}/indexed-assets. Use Assets and IndexedAssets for new
// code; this service exists for compatibility with indexes created before the
// change.
type TasksService struct {
	client *Client
}

// VideoIndexingTask is a legacy indexing job.
//
// Deprecated: use IndexedAsset.
type VideoIndexingTask struct {
	ID             string                           `json:"_id"`
	VideoID        string                           `json:"video_id,omitempty"`
	CreatedAt      string                           `json:"created_at,omitempty"`
	UpdatedAt      string                           `json:"updated_at,omitempty"`
	Status         string                           `json:"status,omitempty"`
	IndexID        string                           `json:"index_id,omitempty"`
	SystemMetadata *VideoIndexingTaskSystemMetadata `json:"system_metadata,omitempty"`
	HLS            *HLS                             `json:"hls,omitempty"`
	Error          *ErrorBody                       `json:"error,omitempty"`
}

// VideoIndexingTaskSystemMetadata describes the indexed video.
type VideoIndexingTaskSystemMetadata struct {
	Filename string  `json:"filename,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	Width    int     `json:"width,omitempty"`
	Height   int     `json:"height,omitempty"`
}

// TaskCreateRequest creates a legacy indexing task from a file or a URL.
// Set exactly one of File or VideoURL.
type TaskCreateRequest struct {
	IndexID  string
	File     *FileUpload
	VideoURL string
	// EnableVideoStream generates an HLS playlist. The API defaults to true.
	EnableVideoStream *bool
	UserMetadata      map[string]any
}

// TaskListOptions filters List.
type TaskListOptions struct {
	Pagination
	SortBy     string
	SortOption string
	IndexID    string
	Status     []string
	Filename   string
	// Duration filters on the duration in seconds. The API matches this exact
	// value; it does not accept a range.
	Duration *float64
	// Width filters on the width in pixels.
	Width *int
	// Height filters on the height in pixels.
	Height *int
	// CreatedAt filters on the RFC 3339 creation time.
	CreatedAt string
	// UpdatedAt filters on the RFC 3339 update time.
	UpdatedAt string
}

// Create starts a legacy indexing task. The task is asynchronous; use
// WaitForReady before searching the video.
//
// Deprecated: use Assets.Create followed by IndexedAssets.Create.
func (s *TasksService) Create(ctx context.Context, request *TaskCreateRequest) (*VideoIndexingTask, error) {
	ctx = ensureContext(ctx)
	if request == nil {
		return nil, validationError("request", "is required")
	}
	if request.IndexID == "" {
		return nil, validationError("IndexID", "is required")
	}
	if request.File != nil && request.VideoURL != "" {
		return nil, validationError("request", "set either File or VideoURL, not both")
	}
	if request.File == nil && request.VideoURL == "" {
		return nil, validationError("request", "one of File or VideoURL is required")
	}
	if request.File != nil {
		if err := request.File.validate("File"); err != nil {
			return nil, err
		}
	}

	body := &multipartBody{write: func(w *multipart.Writer) error {
		if err := w.WriteField("index_id", request.IndexID); err != nil {
			return err
		}
		if request.File != nil {
			if err := writeFilePart(w, "video_file", request.File); err != nil {
				return err
			}
		} else if err := w.WriteField("video_url", request.VideoURL); err != nil {
			return err
		}
		if request.EnableVideoStream != nil {
			if err := w.WriteField("enable_video_stream", boolString(*request.EnableVideoStream)); err != nil {
				return err
			}
		}
		return writeUserMetadataField(w, request.UserMetadata)
	}}

	var created struct {
		ID      string `json:"_id"`
		VideoID string `json:"video_id"`
	}
	if err := s.client.transport.doMultipart(ctx, http.MethodPost, "/tasks", nil, body, &created); err != nil {
		return nil, err
	}
	if created.ID == "" {
		return nil, errEmptyID("task")
	}
	return &VideoIndexingTask{ID: created.ID, VideoID: created.VideoID, IndexID: request.IndexID}, nil
}

// List returns legacy indexing tasks.
//
// Deprecated: use IndexedAssets.List.
func (s *TasksService) List(ctx context.Context, options *TaskListOptions) (*Page[VideoIndexingTask], error) {
	ctx = ensureContext(ctx)
	query := url.Values{}
	if options != nil {
		query = encodeQuery(&options.Pagination)
		setIfNotEmpty(query, "sort_by", options.SortBy)
		setIfNotEmpty(query, "sort_option", options.SortOption)
		setIfNotEmpty(query, "index_id", options.IndexID)
		setIfNotEmpty(query, "filename", options.Filename)
		setIfNotEmpty(query, "created_at", options.CreatedAt)
		setIfNotEmpty(query, "updated_at", options.UpdatedAt)
		for i, status := range options.Status {
			switch status {
			case TaskStatusValidating, TaskStatusPending, TaskStatusQueued, TaskStatusIndexing, TaskStatusReady, TaskStatusFailed:
				query.Add("status", status)
			default:
				return nil, validationError("Status", "[%d] must be a valid task status", i)
			}
		}
		if options.Duration != nil {
			if *options.Duration < 0 {
				return nil, validationError("Duration", "must not be negative")
			}
			query.Set("duration", strconv.FormatFloat(*options.Duration, 'f', -1, 64))
		}
		if options.Width != nil {
			if *options.Width < 0 {
				return nil, validationError("Width", "must not be negative")
			}
			query.Set("width", strconv.Itoa(*options.Width))
		}
		if options.Height != nil {
			if *options.Height < 0 {
				return nil, validationError("Height", "must not be negative")
			}
			query.Set("height", strconv.Itoa(*options.Height))
		}
	}

	var page Page[VideoIndexingTask]
	if err := s.client.transport.doJSON(ctx, http.MethodGet, "/tasks", query, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// Get retrieves a legacy indexing task.
//
// Deprecated: use IndexedAssets.Get.
func (s *TasksService) Get(ctx context.Context, taskID string) (*VideoIndexingTask, error) {
	ctx = ensureContext(ctx)
	if taskID == "" {
		return nil, validationError("taskID", "is required")
	}
	var task VideoIndexingTask
	path := "/tasks/" + escapePath(taskID)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, nil, nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// Delete removes a legacy indexing task and its indexed video.
//
// Deprecated: use IndexedAssets.Delete.
func (s *TasksService) Delete(ctx context.Context, taskID string) error {
	ctx = ensureContext(ctx)
	if taskID == "" {
		return validationError("taskID", "is required")
	}
	return s.client.transport.doJSON(ctx, http.MethodDelete, "/tasks/"+escapePath(taskID), nil, nil, nil)
}

// WaitForReady polls a legacy indexing task until it is ready or failed.
//
// Deprecated: use IndexedAssets.WaitForReady.
func (s *TasksService) WaitForReady(ctx context.Context, taskID string, options *WaitOptions) (*VideoIndexingTask, error) {
	ctx = ensureContext(ctx)
	if taskID == "" {
		return nil, validationError("taskID", "is required")
	}

	var deadline time.Time
	if options != nil && options.Timeout > 0 {
		deadline = time.Now().Add(options.Timeout)
	}

	for {
		task, err := s.Get(ctx, taskID)
		if err != nil {
			return nil, err
		}

		switch task.Status {
		case TaskStatusReady:
			return task, nil
		case TaskStatusFailed:
			return task, errTaskFailed("task", taskID, task.Error)
		}

		if options != nil && options.OnPoll != nil {
			if err := options.OnPoll(task.Status); err != nil {
				return nil, err
			}
		}
		if !deadline.IsZero() && time.Now().After(deadline) {
			return nil, ErrTimeout
		}
		if err := sleepContext(ctx, options.waitInterval()); err != nil {
			return nil, err
		}
	}
}
