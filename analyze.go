package twelvelabs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Analysis modes.
const (
	AnalysisModeGeneral           = "general"
	AnalysisModeTimeBasedMetadata = "time_based_metadata"
)

// Analyze task status values. Ready, failed, and canceled are final.
const (
	AnalyzeTaskStatusQueued     = "queued"
	AnalyzeTaskStatusPending    = "pending"
	AnalyzeTaskStatusProcessing = "processing"
	AnalyzeTaskStatusReady      = "ready"
	AnalyzeTaskStatusFailed     = "failed"
	AnalyzeTaskStatusCanceled   = "canceled"
)

// Stream event types emitted by Analyze.Stream.
const (
	StreamEventStart = "stream_start"
	StreamEventText  = "text_generation"
	StreamEventEnd   = "stream_end"
)

// Response format types.
const (
	ResponseFormatJSONSchema         = "json_schema"
	ResponseFormatSegmentDefinitions = "segment_definitions"
)

// Segment field types.
const (
	SegmentFieldString    = "string"
	SegmentFieldBoolean   = "boolean"
	SegmentFieldNumber    = "number"
	SegmentFieldInteger   = "integer"
	SegmentFieldArray     = "array"
	SegmentFieldTimestamp = "timestamp"
	SegmentFieldTimeArray = "time_array"
)

// Segment timestamp formats.
const (
	TimeFormatSeconds      = "seconds"
	TimeFormatHHMMSS       = "hh:mm:ss"
	TimeFormatHHMMSSMillis = "hh:mm:ss.fff"
)

var customIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// AnalyzeService generates text and structured metadata from video and images.
type AnalyzeService struct {
	client *Client
}

// VideoContext identifies the video to analyze. Use VideoFromURL,
// VideoFromAssetID, or VideoFromBase64 to build one.
type VideoContext struct {
	Type         string `json:"type"`
	URL          string `json:"url,omitempty"`
	AssetID      string `json:"asset_id,omitempty"`
	Base64String string `json:"base64_string,omitempty"`
}

// VideoFromURL analyzes a publicly accessible video file or HLS manifest.
func VideoFromURL(mediaURL string) *VideoContext {
	return &VideoContext{Type: "url", URL: mediaURL}
}

// VideoFromAssetID analyzes a ready asset.
func VideoFromAssetID(assetID string) *VideoContext {
	return &VideoContext{Type: "asset_id", AssetID: assetID}
}

// VideoFromBase64 analyzes inline base64-encoded video data.
func VideoFromBase64(encoded string) *VideoContext {
	return &VideoContext{Type: "base64_string", Base64String: encoded}
}

func (v *VideoContext) validate(field string) error {
	if v == nil {
		return validationError(field, "is required")
	}
	switch v.Type {
	case "url":
		if v.URL == "" {
			return validationError(field, "URL is required when Type is %q", v.Type)
		}
	case "asset_id":
		if v.AssetID == "" {
			return validationError(field, "AssetID is required when Type is %q", v.Type)
		}
	case "base64_string":
		if v.Base64String == "" {
			return validationError(field, "Base64String is required when Type is %q", v.Type)
		}
	default:
		return validationError(field, `Type must be "url", "asset_id", or "base64_string"`)
	}
	return nil
}

// AnalyzeImage is a reference image supplied alongside a prompt.
type AnalyzeImage struct {
	// Name is referenced from the prompt as <@name>. It must be unique.
	Name string `json:"name"`
	// Provide exactly one of URL, AssetID, or Base64String.
	URL          string `json:"url,omitempty"`
	AssetID      string `json:"asset_id,omitempty"`
	Base64String string `json:"base64_string,omitempty"`
}

// MediaSourceRef is a named reference image used by prompt_v2, segment
// definitions, and multi-input embeddings.
type MediaSourceRef struct {
	Name      string `json:"name"`
	MediaType string `json:"media_type"`
	// Provide exactly one of URL, AssetID, or Base64String.
	URL          string `json:"url,omitempty"`
	AssetID      string `json:"asset_id,omitempty"`
	Base64String string `json:"base64_string,omitempty"`
}

// AnalyzePromptV2 is a prompt that references images by name.
type AnalyzePromptV2 struct {
	InputText    string           `json:"input_text"`
	MediaSources []MediaSourceRef `json:"media_sources,omitempty"`
}

// TokenUsage counts the tokens consumed by a generation.
type TokenUsage struct {
	OutputTokens int `json:"output_tokens"`
	InputTokens  int `json:"input_tokens,omitempty"`
}

// SegmentField describes one custom field extracted for a segment.
type SegmentField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	// Format is required when Type is SegmentFieldTimestamp and must be empty
	// otherwise.
	Format string             `json:"format,omitempty"`
	Enum   []string           `json:"enum,omitempty"`
	Items  *SegmentFieldItems `json:"items,omitempty"`
}

// SegmentFieldItems declares the element type of an array field.
type SegmentFieldItems struct {
	Type   string         `json:"type"`
	Fields []SegmentField `json:"fields,omitempty"`
}

// TimeRange is a window of the video, in seconds.
type TimeRange struct {
	StartTime float64 `json:"start_time"`
	EndTime   float64 `json:"end_time"`
}

// SegmentDefinition describes a kind of segment to extract from a video.
type SegmentDefinition struct {
	ID           string           `json:"id"`
	Description  string           `json:"description"`
	Fields       []SegmentField   `json:"fields,omitempty"`
	MediaSources []MediaSourceRef `json:"media_sources,omitempty"`
	TimeRanges   []TimeRange      `json:"time_ranges,omitempty"`
}

// ResponseFormat requests structured output. Type is either
// ResponseFormatJSONSchema or ResponseFormatSegmentDefinitions.
type ResponseFormat struct {
	Type string `json:"type"`
	// JSONSchema is required when Type is ResponseFormatJSONSchema.
	JSONSchema map[string]any `json:"json_schema,omitempty"`
	// SegmentDefinitions is required when Type is
	// ResponseFormatSegmentDefinitions.
	SegmentDefinitions []SegmentDefinition `json:"segment_definitions,omitempty"`
	// SegmentTimeFormat controls the format of automatic segment boundaries.
	SegmentTimeFormat string `json:"segment_time_format,omitempty"`
}

func (f *ResponseFormat) validate(field string) error {
	if f == nil {
		return nil
	}
	switch f.Type {
	case ResponseFormatJSONSchema:
		if len(f.JSONSchema) == 0 {
			return validationError(field, "JSONSchema is required when Type is %q", ResponseFormatJSONSchema)
		}
	case ResponseFormatSegmentDefinitions:
		if len(f.SegmentDefinitions) == 0 {
			return validationError(field, "at least one SegmentDefinition is required when Type is %q", ResponseFormatSegmentDefinitions)
		}
		for i, definition := range f.SegmentDefinitions {
			if definition.ID == "" {
				return validationError(field, "SegmentDefinitions[%d].ID is required", i)
			}
			if definition.Description == "" {
				return validationError(field, "SegmentDefinitions[%d].Description is required", i)
			}
			for j, segmentField := range definition.Fields {
				if segmentField.Name == "" {
					return validationError(field, "SegmentDefinitions[%d].Fields[%d].Name is required", i, j)
				}
				if segmentField.Type == "" {
					return validationError(field, "SegmentDefinitions[%d].Fields[%d].Type is required", i, j)
				}
				if segmentField.Type == SegmentFieldTimestamp && segmentField.Format == "" {
					return validationError(field, "SegmentDefinitions[%d].Fields[%d].Format is required for timestamp fields", i, j)
				}
				if segmentField.Type != SegmentFieldTimestamp && segmentField.Format != "" {
					return validationError(field, "SegmentDefinitions[%d].Fields[%d].Format is only valid for timestamp fields", i, j)
				}
			}
			for j, span := range definition.TimeRanges {
				if span.EndTime-span.StartTime < 2 {
					return validationError(field, "SegmentDefinitions[%d].TimeRanges[%d] must be at least 2 seconds long", i, j)
				}
			}
		}
	default:
		return validationError(field, "Type must be %q or %q", ResponseFormatJSONSchema, ResponseFormatSegmentDefinitions)
	}
	return nil
}

// AnalyzeRequest is a synchronous analysis request. The response is returned
// in one piece by Analyze or streamed incrementally by Stream.
type AnalyzeRequest struct {
	// ModelName defaults to ModelPegasus15. Set ModelPegasus16 for image
	// inputs.
	ModelName string `json:"model_name,omitempty"`
	// Video and Image are mutually exclusive as the primary media. At least one
	// is required.
	Video *VideoContext  `json:"video,omitempty"`
	Image []AnalyzeImage `json:"image,omitempty"`
	// Prompt is a free-form prompt. Mutually exclusive with PromptV2.
	Prompt string `json:"prompt,omitempty"`
	// PromptV2 is a prompt that references named images. Mutually exclusive
	// with Prompt.
	PromptV2 *AnalyzePromptV2 `json:"prompt_v2,omitempty"`
	// Temperature controls randomness between 0 and 1. The API defaults to 0.2.
	Temperature *float64 `json:"temperature,omitempty"`
	// ResponseFormat requests structured output.
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	// MaxTokens bounds the response. The API defaults to 4096.
	MaxTokens int `json:"max_tokens,omitempty"`
	// StartTime and EndTime narrow the analysis to a time window, in seconds.
	StartTime *float64 `json:"start_time,omitempty"`
	EndTime   *float64 `json:"end_time,omitempty"`
}

// AnalyzeResponse is the non-streaming analysis result.
type AnalyzeResponse struct {
	ID           string        `json:"id,omitempty"`
	Data         string        `json:"data"`
	FinishReason string        `json:"finish_reason,omitempty"`
	Usage        *TokenUsage   `json:"usage,omitempty"`
	Error        *AnalyzeError `json:"error,omitempty"`
}

// AnalyzeError reports a failed or partial generation.
type AnalyzeError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *AnalyzeError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

// StreamMetadata accompanies the start and end events of a stream.
type StreamMetadata struct {
	GenerationID string      `json:"generation_id,omitempty"`
	Usage        *TokenUsage `json:"usage,omitempty"`
}

// AnalyzeStreamEvent is one NDJSON event from Analyze.Stream.
type AnalyzeStreamEvent struct {
	EventType    string          `json:"event_type"`
	Metadata     *StreamMetadata `json:"metadata,omitempty"`
	Text         string          `json:"text,omitempty"`
	FinishReason string          `json:"finish_reason,omitempty"`
	Error        *AnalyzeError   `json:"error,omitempty"`
}

// AnalyzeStreamError is returned by Stream when generation fails after the
// stream has started.
type AnalyzeStreamError struct {
	Code    string
	Message string
}

func (e *AnalyzeStreamError) Error() string {
	if e.Code != "" {
		return "twelvelabs: stream failed: " + e.Code + ": " + e.Message
	}
	return "twelvelabs: stream failed: " + e.Message
}

// Analyze runs a synchronous analysis and returns the generated text.
//
// Use Stream when the prompt produces long output and you want it as it is
// generated.
func (s *AnalyzeService) Analyze(ctx context.Context, request *AnalyzeRequest) (*AnalyzeResponse, error) {
	ctx = ensureContext(ctx)
	body, err := s.encodeRequest(request, false)
	if err != nil {
		return nil, err
	}

	var response AnalyzeResponse
	if err := s.client.transport.doJSON(ctx, http.MethodPost, "/analyze", nil, body, &response); err != nil {
		return nil, err
	}
	if response.Error != nil {
		return &response, response.Error
	}
	return &response, nil
}

// Stream runs a synchronous analysis and calls handler for every event as it
// arrives. Events are NDJSON, not server-sent events.
//
// The handler is called for stream_start, every stream_text chunk, and
// stream_end. If the stream ends with an error event, Stream returns an
// *AnalyzeStreamError after the handler has been called for that event.
func (s *AnalyzeService) Stream(ctx context.Context, req *AnalyzeRequest, handler func(AnalyzeStreamEvent) error) error {
	ctx = ensureContext(ctx)
	body, err := s.encodeRequest(req, true)
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("twelvelabs: encode request body: %w", err)
	}

	reader, err := s.client.transport.stream(ctx, &request{
		method:   http.MethodPost,
		path:     "/analyze",
		jsonBody: encoded,
		accept:   "application/x-ndjson",
	})
	if err != nil {
		return err
	}
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	var finalErr error
	for {
		var event AnalyzeStreamEvent
		if err := decoder.Decode(&event); err != nil {
			if errors.Is(err, io.EOF) {
				return finalErr
			}
			return fmt.Errorf("twelvelabs: decode stream event: %w", err)
		}
		if handler != nil {
			if err := handler(event); err != nil {
				return err
			}
		}
		if event.EventType == StreamEventEnd && event.Error != nil {
			finalErr = &AnalyzeStreamError{Code: event.Error.Code, Message: event.Error.Message}
		}
	}
}

type analyzeWire struct {
	*AnalyzeRequest
	Stream bool `json:"stream"`
}

func (s *AnalyzeService) encodeRequest(request *AnalyzeRequest, stream bool) (*analyzeWire, error) {
	if request == nil {
		return nil, validationError("request", "is required")
	}

	modelName := request.ModelName
	if modelName == "" {
		modelName = ModelPegasus15
	}
	request.ModelName = modelName

	if request.Video == nil && len(request.Image) == 0 {
		return nil, validationError("Video", "a video or at least one image is required")
	}
	if request.Video != nil {
		if err := request.Video.validate("Video"); err != nil {
			return nil, err
		}
	}
	if request.Prompt != "" && request.PromptV2 != nil {
		return nil, validationError("PromptV2", "set either Prompt or PromptV2, not both")
	}
	for i, image := range request.Image {
		if image.Name == "" {
			return nil, validationError("Image", "Image[%d].Name is required", i)
		}
		if err := validateImageSource("Image", i, image.URL, image.AssetID, image.Base64String); err != nil {
			return nil, err
		}
	}
	if request.PromptV2 != nil {
		if request.PromptV2.InputText == "" {
			return nil, validationError("PromptV2", "InputText is required")
		}
		for i, source := range request.PromptV2.MediaSources {
			if source.Name == "" {
				return nil, validationError("PromptV2", "MediaSources[%d].Name is required", i)
			}
			if err := validateImageSource("PromptV2", i, source.URL, source.AssetID, source.Base64String); err != nil {
				return nil, err
			}
		}
	}
	if err := request.ResponseFormat.validate("ResponseFormat"); err != nil {
		return nil, err
	}
	if request.Temperature != nil && (*request.Temperature < 0 || *request.Temperature > 1) {
		return nil, validationError("Temperature", "must be between 0 and 1")
	}
	if request.StartTime != nil && request.EndTime != nil && *request.EndTime <= *request.StartTime {
		return nil, validationError("EndTime", "must be greater than StartTime")
	}

	return &analyzeWire{AnalyzeRequest: request, Stream: stream}, nil
}

// AnalyzeTaskCreateRequest creates an asynchronous analysis task.
type AnalyzeTaskCreateRequest struct {
	// ModelName defaults to ModelPegasus15.
	ModelName string `json:"model_name,omitempty"`
	// CustomID is echoed back on the task so results can be correlated. It must
	// match ^[a-zA-Z0-9_-]{1,64}$.
	CustomID string `json:"custom_id,omitempty"`
	// Video is the video to analyze. At least one of Video or Image is
	// required.
	Video *VideoContext `json:"video,omitempty"`
	// Image supplies reference images. Requires ModelPegasus16.
	Image []AnalyzeImage `json:"image,omitempty"`
	// Prompt is a free-form prompt. Mutually exclusive with PromptV2.
	Prompt string `json:"prompt,omitempty"`
	// PromptV2 is a prompt that references named images. Mutually exclusive
	// with Prompt.
	PromptV2 *AnalyzePromptV2 `json:"prompt_v2,omitempty"`
	// AnalysisMode is AnalysisModeGeneral (default) or
	// AnalysisModeTimeBasedMetadata.
	AnalysisMode string `json:"analysis_mode,omitempty"`
	// Temperature controls randomness between 0 and 1.
	Temperature *float64 `json:"temperature,omitempty"`
	// MaxTokens bounds the response. Zero uses the platform default.
	MaxTokens int `json:"max_tokens,omitempty"`
	// UnlimitedTokens removes the token limit. It requires
	// AnalysisModeTimeBasedMetadata and ModelPegasus16, and cannot be combined
	// with time ranges in the segment definitions.
	UnlimitedTokens bool `json:"-"`
	// ResponseFormat requests structured output or segment extraction.
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	// MinSegmentDuration and MaxSegmentDuration constrain extracted segments,
	// in seconds. They require AnalysisModeTimeBasedMetadata.
	MinSegmentDuration float64 `json:"min_segment_duration,omitempty"`
	MaxSegmentDuration float64 `json:"max_segment_duration,omitempty"`
	// StartTime and EndTime narrow the analysis to a time window, in seconds.
	StartTime *float64 `json:"start_time,omitempty"`
	EndTime   *float64 `json:"end_time,omitempty"`
}

// MarshalJSON encodes max_tokens as either a number or the "unlimited" string.
func (r AnalyzeTaskCreateRequest) MarshalJSON() ([]byte, error) {
	type alias AnalyzeTaskCreateRequest
	body := struct {
		alias
		MaxTokens any `json:"max_tokens,omitempty"`
	}{alias: alias(r)}

	switch {
	case r.UnlimitedTokens:
		body.MaxTokens = "unlimited"
	case r.MaxTokens > 0:
		body.MaxTokens = r.MaxTokens
	}
	return json.Marshal(body)
}

// AnalyzeVideoSource describes the video a task was created with.
type AnalyzeVideoSource struct {
	Type    string  `json:"type"`
	URL     *string `json:"url,omitempty"`
	AssetID *string `json:"asset_id,omitempty"`
	// VideoID and IndexID are set for tasks created from indexed videos.
	VideoID string `json:"video_id,omitempty"`
	IndexID string `json:"index_id,omitempty"`
	// Duration is the video length in seconds, when the platform could
	// determine it.
	SystemMetadata *struct {
		Duration float64 `json:"duration,omitempty"`
	} `json:"system_metadata,omitempty"`
}

// AnalyzeTaskParams echoes the parameters a task was created with.
type AnalyzeTaskParams struct {
	AnalysisMode       string           `json:"analysis_mode,omitempty"`
	Prompt             *string          `json:"prompt,omitempty"`
	PromptV2           *AnalyzePromptV2 `json:"prompt_v2,omitempty"`
	Image              []AnalyzeImage   `json:"image,omitempty"`
	ResponseFormat     *ResponseFormat  `json:"response_format,omitempty"`
	Temperature        *float64         `json:"temperature,omitempty"`
	MaxTokens          int              `json:"max_tokens,omitempty"`
	MinSegmentDuration float64          `json:"min_segment_duration,omitempty"`
	MaxSegmentDuration float64          `json:"max_segment_duration,omitempty"`
	StartTime          *float64         `json:"start_time,omitempty"`
	EndTime            *float64         `json:"end_time,omitempty"`
}

// AnalyzeTaskResult is the output of a completed task. For
// AnalysisModeTimeBasedMetadata, Data is a JSON-encoded object keyed by segment
// definition.
type AnalyzeTaskResult struct {
	GenerationID string      `json:"generation_id,omitempty"`
	Data         string      `json:"data"`
	FinishReason string      `json:"finish_reason,omitempty"`
	Usage        *TokenUsage `json:"usage,omitempty"`
}

// WebhookDelivery reports a webhook delivery attempt made by the platform.
type WebhookDelivery struct {
	URL       string `json:"url"`
	Delivered bool   `json:"delivered"`
	Attempts  int    `json:"attempts"`
	LastError string `json:"last_error,omitempty"`
}

// AnalyzeTask is an asynchronous analysis job.
type AnalyzeTask struct {
	TaskID        string              `json:"task_id"`
	CustomID      *string             `json:"custom_id"`
	BatchID       string              `json:"batch_id,omitempty"`
	VideoSource   *AnalyzeVideoSource `json:"video_source,omitempty"`
	RequestParams *AnalyzeTaskParams  `json:"request_params,omitempty"`
	Status        string              `json:"status"`
	CreatedAt     string              `json:"created_at,omitempty"`
	CompletedAt   string              `json:"completed_at,omitempty"`
	Result        *AnalyzeTaskResult  `json:"result,omitempty"`
	Error         *AnalyzeError       `json:"error,omitempty"`
	Webhooks      []WebhookDelivery   `json:"webhooks,omitempty"`
}

// AnalyzeTaskListOptions filters ListTasks.
type AnalyzeTaskListOptions struct {
	Pagination
	Status       []string
	VideoURL     string
	AssetID      string
	AnalysisMode string
}

// CreateTask queues an asynchronous analysis task.
func (s *AnalyzeService) CreateTask(ctx context.Context, request *AnalyzeTaskCreateRequest) (*AnalyzeTask, error) {
	ctx = ensureContext(ctx)
	if err := prepareTaskRequest(request); err != nil {
		return nil, err
	}

	var response struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	if err := s.client.transport.doJSON(ctx, http.MethodPost, "/analyze/tasks", nil, request, &response); err != nil {
		return nil, err
	}
	if response.TaskID == "" {
		return nil, errEmptyID("analyze task")
	}
	return &AnalyzeTask{TaskID: response.TaskID, Status: response.Status, CustomID: optionalString(request.CustomID)}, nil
}

// ListTasks returns asynchronous analysis tasks.
func (s *AnalyzeService) ListTasks(ctx context.Context, options *AnalyzeTaskListOptions) (*Page[AnalyzeTask], error) {
	ctx = ensureContext(ctx)
	query := url.Values{}
	if options != nil {
		query = encodeQuery(&options.Pagination)
		addStrings(query, "status", options.Status)
		setIfNotEmpty(query, "video_url", options.VideoURL)
		setIfNotEmpty(query, "asset_id", options.AssetID)
		setIfNotEmpty(query, "analysis_mode", options.AnalysisMode)
	}

	var page Page[AnalyzeTask]
	if err := s.client.transport.doJSON(ctx, http.MethodGet, "/analyze/tasks", query, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// GetTask retrieves an analysis task, including its result once it is ready.
func (s *AnalyzeService) GetTask(ctx context.Context, taskID string) (*AnalyzeTask, error) {
	ctx = ensureContext(ctx)
	if taskID == "" {
		return nil, validationError("taskID", "is required")
	}
	var task AnalyzeTask
	path := "/analyze/tasks/" + escapePath(taskID)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, nil, nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// CancelTask cancels a queued or processing task and returns its new status.
func (s *AnalyzeService) CancelTask(ctx context.Context, taskID string) (*AnalyzeTask, error) {
	ctx = ensureContext(ctx)
	if taskID == "" {
		return nil, validationError("taskID", "is required")
	}
	var response struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	path := "/analyze/tasks/" + escapePath(taskID) + "/cancel"
	if err := s.client.transport.doJSON(ctx, http.MethodPost, path, nil, nil, &response); err != nil {
		return nil, err
	}
	return &AnalyzeTask{TaskID: response.TaskID, Status: response.Status}, nil
}

// DeleteTask removes a task and its results.
func (s *AnalyzeService) DeleteTask(ctx context.Context, taskID string) error {
	ctx = ensureContext(ctx)
	if taskID == "" {
		return validationError("taskID", "is required")
	}
	return s.client.transport.doJSON(ctx, http.MethodDelete, "/analyze/tasks/"+escapePath(taskID), nil, nil, nil)
}

// WaitForTask polls a task until it reaches ready, failed, or canceled.
func (s *AnalyzeService) WaitForTask(ctx context.Context, taskID string, options *WaitOptions) (*AnalyzeTask, error) {
	ctx = ensureContext(ctx)
	if taskID == "" {
		return nil, validationError("taskID", "is required")
	}

	var deadline time.Time
	if options != nil && options.Timeout > 0 {
		deadline = time.Now().Add(options.Timeout)
	}

	for {
		task, err := s.GetTask(ctx, taskID)
		if err != nil {
			return nil, err
		}

		switch task.Status {
		case AnalyzeTaskStatusReady:
			return task, nil
		case AnalyzeTaskStatusFailed:
			return task, fmt.Errorf("twelvelabs: analyze task %s failed: %s", taskID, task.Error)
		case AnalyzeTaskStatusCanceled:
			return task, fmt.Errorf("twelvelabs: analyze task %s was canceled", taskID)
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

func prepareTaskRequest(request *AnalyzeTaskCreateRequest) error {
	if request == nil {
		return validationError("request", "is required")
	}
	if request.ModelName == "" {
		request.ModelName = ModelPegasus15
	}
	if request.CustomID != "" && !customIDPattern.MatchString(request.CustomID) {
		return validationError("CustomID", "must match %s", customIDPattern)
	}
	if request.Video == nil && len(request.Image) == 0 {
		return validationError("Video", "a video or at least one image is required")
	}
	if request.Video != nil {
		if err := request.Video.validate("Video"); err != nil {
			return err
		}
	}
	if request.Prompt != "" && request.PromptV2 != nil {
		return validationError("PromptV2", "set either Prompt or PromptV2, not both")
	}
	if request.AnalysisMode == "" {
		request.AnalysisMode = AnalysisModeGeneral
	}
	switch request.AnalysisMode {
	case AnalysisModeGeneral:
		if request.ResponseFormat != nil && request.ResponseFormat.Type == ResponseFormatSegmentDefinitions {
			return validationError("ResponseFormat", "segment definitions require AnalysisMode %q", AnalysisModeTimeBasedMetadata)
		}
	case AnalysisModeTimeBasedMetadata:
		if request.Prompt != "" || request.PromptV2 != nil {
			return validationError("Prompt", "prompts are not supported with AnalysisMode %q", AnalysisModeTimeBasedMetadata)
		}
		if request.ResponseFormat == nil || request.ResponseFormat.Type != ResponseFormatSegmentDefinitions {
			return validationError("ResponseFormat", "analysis mode %q requires segment definitions", AnalysisModeTimeBasedMetadata)
		}
		if request.UnlimitedTokens && request.ModelName != ModelPegasus16 {
			return validationError("UnlimitedTokens", "requires model %q", ModelPegasus16)
		}
		if request.UnlimitedTokens && request.MaxTokens > 0 {
			return validationError("MaxTokens", "set either MaxTokens or UnlimitedTokens, not both")
		}
		if request.UnlimitedTokens {
			for _, definition := range request.ResponseFormat.SegmentDefinitions {
				if len(definition.TimeRanges) > 0 {
					return validationError("UnlimitedTokens", "cannot be combined with segment time ranges")
				}
			}
		}
		if request.MaxTokens > 0 && request.MaxTokens < 2048 {
			return validationError("MaxTokens", "must be at least 2048 with AnalysisMode %q", AnalysisModeTimeBasedMetadata)
		}
	default:
		return validationError("AnalysisMode", "must be %q or %q", AnalysisModeGeneral, AnalysisModeTimeBasedMetadata)
	}
	if err := request.ResponseFormat.validate("ResponseFormat"); err != nil {
		return err
	}
	for i, image := range request.Image {
		if image.Name == "" {
			return validationError("Image", "Image[%d].Name is required", i)
		}
		if err := validateImageSource("Image", i, image.URL, image.AssetID, image.Base64String); err != nil {
			return err
		}
	}
	if request.PromptV2 != nil {
		if request.PromptV2.InputText == "" {
			return validationError("PromptV2", "InputText is required")
		}
		for i, source := range request.PromptV2.MediaSources {
			if source.Name == "" {
				return validationError("PromptV2", "MediaSources[%d].Name is required", i)
			}
			if err := validateImageSource("PromptV2", i, source.URL, source.AssetID, source.Base64String); err != nil {
				return err
			}
		}
	}
	if request.Temperature != nil && (*request.Temperature < 0 || *request.Temperature > 1) {
		return validationError("Temperature", "must be between 0 and 1")
	}
	if request.MinSegmentDuration > 0 && request.MaxSegmentDuration > 0 && request.MaxSegmentDuration < request.MinSegmentDuration {
		return validationError("MaxSegmentDuration", "must be greater than or equal to MinSegmentDuration")
	}
	if request.StartTime != nil && request.EndTime != nil && *request.EndTime <= *request.StartTime {
		return validationError("EndTime", "must be greater than StartTime")
	}
	return nil
}

func validateImageSource(field string, index int, url, assetID, base64String string) error {
	provided := 0
	for _, value := range []string{url, assetID, base64String} {
		if value != "" {
			provided++
		}
	}
	if provided != 1 {
		return validationError(field, "[%d] must set exactly one of URL, AssetID, or Base64String", index)
	}
	return nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// ParseAnalyzeSegments decodes the structured data returned by an analysis in
// AnalysisModeTimeBasedMetadata. The result maps each segment definition ID to
// its extracted segments.
func ParseAnalyzeSegments(data string) (map[string][]AnalyzeSegment, error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, nil
	}
	var segments map[string][]AnalyzeSegment
	if err := json.Unmarshal([]byte(data), &segments); err != nil {
		return nil, fmt.Errorf("twelvelabs: decode segment data: %w", err)
	}
	return segments, nil
}

// AnalyzeSegment is one extracted time range and its custom metadata.
type AnalyzeSegment struct {
	StartTime any            `json:"start_time"`
	EndTime   any            `json:"end_time"`
	Metadata  map[string]any `json:"metadata"`
}
