package twelvelabs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Embedding input types for the synchronous endpoint.
const (
	EmbedInputText      = "text"
	EmbedInputImage     = "image"
	EmbedInputTextImage = "text_image"
	EmbedInputAudio     = "audio"
	EmbedInputVideo     = "video"
	EmbedInputMulti     = "multi_input"
)

// Embedding input types for the asynchronous endpoint.
const (
	EmbedTaskInputAudio    = "audio"
	EmbedTaskInputVideo    = "video"
	EmbedTaskInputDocument = "document"
	EmbedTaskInputImage    = "image"
)

// Embedding option values.
const (
	EmbeddingOptionVisual        = "visual"
	EmbeddingOptionAudio         = "audio"
	EmbeddingOptionTranscription = "transcription"
	EmbeddingOptionText          = "text"
)

// Embedding scope values.
const (
	EmbeddingScopeClip  = "clip"
	EmbeddingScopeAsset = "asset"
	// EmbeddingScopeLocal is used by document embeddings.
	EmbeddingScopeLocal = "local"
)

// Embedding type values.
const (
	EmbeddingTypeSeparate = "separate_embedding"
	EmbeddingTypeFused    = "fused_embedding"
)

// Segmentation strategies.
const (
	SegmentationStrategyFixed   = "fixed"
	SegmentationStrategyDynamic = "dynamic"
)

// Embedding dimensions accepted by Marengo 3.5.
const (
	EmbeddingDimensions128 = "128"
	EmbeddingDimensions256 = "256"
	EmbeddingDimensions512 = "512"
)

// Embedding task status values.
const (
	EmbedTaskStatusProcessing = "processing"
	EmbedTaskStatusReady      = "ready"
	EmbedTaskStatusFailed     = "failed"
)

// EmbedService creates multimodal embeddings with the Embed API v2.
type EmbedService struct {
	client *Client
}

// FixedSegmentation splits media into equal-length segments.
type FixedSegmentation struct {
	// DurationSec must be between 2 and 10 seconds.
	DurationSec int `json:"duration_sec"`
}

// DynamicSegmentation splits media at scene boundaries.
type DynamicSegmentation struct {
	// MinDurationSec must be between 2 and 5 seconds.
	MinDurationSec int `json:"min_duration_sec"`
}

// Segmentation describes how the platform divides media into segments.
//
// Marengo 3.0 reads Strategy together with Fixed or Dynamic. Marengo 3.5 task
// requests nest the settings under Temporal instead. Use FixedSegments,
// DynamicSegments, or TemporalSegments to build one.
type Segmentation struct {
	Strategy string               `json:"strategy,omitempty"`
	Fixed    *FixedSegmentation   `json:"fixed,omitempty"`
	Dynamic  *DynamicSegmentation `json:"dynamic,omitempty"`
	Temporal *Segmentation        `json:"temporal,omitempty"`
}

// FixedSegments returns fixed-length segmentation.
func FixedSegments(durationSec int) *Segmentation {
	return &Segmentation{Strategy: SegmentationStrategyFixed, Fixed: &FixedSegmentation{DurationSec: durationSec}}
}

// DynamicSegments returns scene-aware segmentation.
func DynamicSegments(minDurationSec int) *Segmentation {
	return &Segmentation{Strategy: SegmentationStrategyDynamic, Dynamic: &DynamicSegmentation{MinDurationSec: minDurationSec}}
}

// TemporalSegments wraps segmentation settings for Marengo 3.5 task requests.
func TemporalSegments(inner *Segmentation) *Segmentation {
	return &Segmentation{Temporal: inner}
}

// EmbedTextInput embeds text.
type EmbedTextInput struct {
	InputText string `json:"input_text"`
}

// EmbedImageInput embeds an image.
type EmbedImageInput struct {
	MediaSource MediaSource `json:"media_source"`
}

// EmbedTextImageInput embeds text and an image together.
type EmbedTextImageInput struct {
	MediaSource MediaSource `json:"media_source"`
	InputText   string      `json:"input_text"`
}

// TimeBasedMetadataEntry folds text into the fused embedding of the segments it
// overlaps.
type TimeBasedMetadataEntry struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Text  string  `json:"text"`
}

// EmbedAudioVideoInput embeds audio or video. Which fields are valid depends on
// the input type and model version.
type EmbedAudioVideoInput struct {
	MediaSource     MediaSource   `json:"media_source"`
	StartSec        *float64      `json:"start_sec,omitempty"`
	EndSec          *float64      `json:"end_sec,omitempty"`
	Segmentation    *Segmentation `json:"segmentation,omitempty"`
	EmbeddingOption []string      `json:"embedding_option,omitempty"`
	EmbeddingScope  []string      `json:"embedding_scope,omitempty"`
	EmbeddingType   []string      `json:"embedding_type,omitempty"`
	// TimeBasedMetadata is only accepted by the asynchronous endpoint.
	TimeBasedMetadata []TimeBasedMetadataEntry `json:"time_based_metadata,omitempty"`
}

// EmbedImageTaskInput embeds an image asynchronously.
type EmbedImageTaskInput struct {
	MediaSource     MediaSource `json:"media_source"`
	EmbeddingOption []string    `json:"embedding_option,omitempty"`
	EmbeddingScope  []string    `json:"embedding_scope,omitempty"`
	EmbeddingType   []string    `json:"embedding_type,omitempty"`
}

// DocumentSpatialSegmentation divides each page of a PDF.
type DocumentSpatialSegmentation struct {
	// Strategy is "standard" (default) or "quadrants".
	Strategy string `json:"strategy,omitempty"`
}

// DocumentSequentialSegmentation divides plain text or Markdown into chunks.
type DocumentSequentialSegmentation struct {
	// Strategy is always "sentence".
	Strategy string `json:"strategy"`
	// MaxSentences caps the sentences per chunk. Required by the API.
	MaxSentences     int `json:"max_sentences"`
	OverlapSentences int `json:"overlap_sentences,omitempty"`
}

// DocumentSegmentation divides a document before embedding.
type DocumentSegmentation struct {
	Spatial    *DocumentSpatialSegmentation    `json:"spatial,omitempty"`
	Sequential *DocumentSequentialSegmentation `json:"sequential,omitempty"`
}

// EmbedDocumentInput embeds a PDF, plain text, or Markdown file. Marengo 3.5
// only, asynchronous endpoint only.
type EmbedDocumentInput struct {
	MediaSource     MediaSource           `json:"media_source"`
	Segmentation    *DocumentSegmentation `json:"segmentation,omitempty"`
	EmbeddingOption []string              `json:"embedding_option,omitempty"`
	EmbeddingScope  []string              `json:"embedding_scope,omitempty"`
	EmbeddingType   []string              `json:"embedding_type,omitempty"`
}

// EmbedMultiInput combines text and up to 10 media sources into one embedding.
type EmbedMultiInput struct {
	InputText    string                  `json:"input_text,omitempty"`
	MediaSources []MultiInputMediaSource `json:"media_sources,omitempty"`
}

// MultiInputMediaSource is one media source of a multi-input embedding.
type MultiInputMediaSource struct {
	Name string `json:"name,omitempty"`
	// MediaType is "image", "video", "audio", or "document".
	MediaType    string `json:"media_type"`
	URL          string `json:"url,omitempty"`
	AssetID      string `json:"asset_id,omitempty"`
	Base64String string `json:"base64_string,omitempty"`
}

// EmbedRequest creates a synchronous embedding.
type EmbedRequest struct {
	// InputType is one of the EmbedInput* constants.
	InputType string `json:"input_type"`
	// ModelName is ModelMarengo30 (default) or ModelMarengo35.
	ModelName string `json:"model_name,omitempty"`
	// AutoTruncate truncates input that exceeds the context window instead of
	// failing.
	AutoTruncate bool `json:"auto_truncate,omitempty"`
	// EmbeddingUncertainty returns a per-dimension uncertainty vector. Marengo
	// 3.5 only.
	EmbeddingUncertainty bool `json:"embedding_uncertainty,omitempty"`
	// EmbeddingDimension is one of the EmbeddingDimensions* constants. Marengo
	// 3.5 only.
	EmbeddingDimension string `json:"embedding_dimension,omitempty"`

	Text       *EmbedTextInput       `json:"text,omitempty"`
	Image      *EmbedImageInput      `json:"image,omitempty"`
	TextImage  *EmbedTextImageInput  `json:"text_image,omitempty"`
	Audio      *EmbedAudioVideoInput `json:"audio,omitempty"`
	Video      *EmbedAudioVideoInput `json:"video,omitempty"`
	MultiInput *EmbedMultiInput      `json:"multi_input,omitempty"`
}

// Embedding is one vector in an embedding response.
type Embedding struct {
	Embedding            []float64 `json:"embedding"`
	EmbeddingUncertainty []float64 `json:"embedding_uncertainty,omitempty"`
	EmbeddingOption      *string   `json:"embedding_option,omitempty"`
	EmbeddingScope       *string   `json:"embedding_scope,omitempty"`
	StartSec             *float64  `json:"start_sec,omitempty"`
	EndSec               *float64  `json:"end_sec,omitempty"`
	StartPageNumber      *int      `json:"start_page_number,omitempty"`
	EndPageNumber        *int      `json:"end_page_number,omitempty"`
	Quadrant             *string   `json:"quadrant,omitempty"`
	ChunkIndex           *int      `json:"chunk_index,omitempty"`
}

// EmbeddingUsage counts the tokens an embedding request used.
type EmbeddingUsage struct {
	// InputTokens maps a content type (video, audio, image, document, text) to
	// its token count.
	InputTokens      map[string]int `json:"input_tokens"`
	Truncated        bool           `json:"truncated"`
	TruncationReason string         `json:"truncation_reason,omitempty"`
}

// EmbeddingMediaMetadata describes the media an embedding was created from.
type EmbeddingMediaMetadata struct {
	InputType          string   `json:"input_type,omitempty"`
	InputURL           string   `json:"input_url,omitempty"`
	InputFilename      string   `json:"input_filename,omitempty"`
	EmbeddingOptions   []string `json:"embedding_options,omitempty"`
	EmbeddingScopes    []string `json:"embedding_scopes,omitempty"`
	Duration           float64  `json:"duration,omitempty"`
	StartOffsetSec     float64  `json:"start_offset_sec,omitempty"`
	EndOffsetSec       float64  `json:"end_offset_sec,omitempty"`
	EmbeddingDimension int      `json:"embedding_dimension,omitempty"`
	ClipLength         int      `json:"clip_length,omitempty"`
}

// EmbedResponse is a synchronous embedding response.
type EmbedResponse struct {
	Data     []Embedding             `json:"data"`
	Usage    *EmbeddingUsage         `json:"usage,omitempty"`
	Metadata *EmbeddingMediaMetadata `json:"metadata,omitempty"`
}

// EmbedTaskCreateRequest creates an asynchronous embedding task.
type EmbedTaskCreateRequest struct {
	// InputType is one of the EmbedTaskInput* constants.
	InputType string `json:"input_type"`
	// ModelName is ModelMarengo30 (default) or ModelMarengo35.
	ModelName string `json:"model_name,omitempty"`
	// EmbeddingUncertainty returns a per-dimension uncertainty vector. Marengo
	// 3.5 only.
	EmbeddingUncertainty bool `json:"embedding_uncertainty,omitempty"`
	// EmbeddingDimension is one of the EmbeddingDimensions* constants. Marengo
	// 3.5 only.
	EmbeddingDimension string `json:"embedding_dimension,omitempty"`

	Audio    *EmbedAudioVideoInput `json:"audio,omitempty"`
	Video    *EmbedAudioVideoInput `json:"video,omitempty"`
	Document *EmbedDocumentInput   `json:"document,omitempty"`
	Image    *EmbedImageTaskInput  `json:"image,omitempty"`
}

// EmbedTaskCreateResponse is returned when an embedding task is queued.
type EmbedTaskCreateResponse struct {
	ID       string      `json:"_id"`
	Status   string      `json:"status"`
	Data     []Embedding `json:"data,omitempty"`
	Metadata *struct {
		EmbeddingDimension int `json:"embedding_dimension,omitempty"`
	} `json:"metadata,omitempty"`
}

// EmbedTaskMediaMetadata describes the media of a task's embedding.
type EmbedTaskMediaMetadata struct {
	InputURL           string   `json:"input_url,omitempty"`
	InputFilename      string   `json:"input_filename,omitempty"`
	EmbeddingOptions   []string `json:"embedding_options,omitempty"`
	EmbeddingScopes    []string `json:"embedding_scopes,omitempty"`
	Duration           float64  `json:"duration,omitempty"`
	StartOffsetSec     float64  `json:"start_offset_sec,omitempty"`
	EndOffsetSec       float64  `json:"end_offset_sec,omitempty"`
	EmbeddingDimension int      `json:"embedding_dimension,omitempty"`
}

// VideoEmbeddingMetadata describes a task's video input.
type VideoEmbeddingMetadata struct {
	InputURL             string   `json:"input_url,omitempty"`
	InputFilename        string   `json:"input_filename,omitempty"`
	VideoClipLength      float64  `json:"video_clip_length,omitempty"`
	VideoEmbeddingScope  []string `json:"video_embedding_scope,omitempty"`
	VideoEmbeddingOption []string `json:"video_embedding_option,omitempty"`
	Duration             float64  `json:"duration,omitempty"`
}

// AudioEmbeddingMetadata describes a task's audio input.
type AudioEmbeddingMetadata struct {
	InputURL             string   `json:"input_url,omitempty"`
	InputFilename        string   `json:"input_filename,omitempty"`
	AudioEmbeddingOption []string `json:"audio_embedding_option,omitempty"`
	AudioEmbeddingScope  []string `json:"audio_embedding_scope,omitempty"`
	Duration             float64  `json:"duration,omitempty"`
	StartOffsetSec       float64  `json:"start_offset_sec,omitempty"`
	EndOffsetSec         float64  `json:"end_offset_sec,omitempty"`
}

// DocumentEmbeddingMetadata describes a task's document input.
type DocumentEmbeddingMetadata struct {
	InputURL                string   `json:"input_url,omitempty"`
	InputFilename           string   `json:"input_filename,omitempty"`
	DocumentEmbeddingOption []string `json:"document_embedding_option,omitempty"`
	DocumentEmbeddingScope  []string `json:"document_embedding_scope,omitempty"`
}

// ImageEmbeddingMetadata describes a task's image input.
type ImageEmbeddingMetadata struct {
	InputURL             string   `json:"input_url,omitempty"`
	InputFilename        string   `json:"input_filename,omitempty"`
	ImageEmbeddingOption []string `json:"image_embedding_option,omitempty"`
	ImageEmbeddingScope  []string `json:"image_embedding_scope,omitempty"`
}

// EmbedTaskEmbedding groups the metadata of one media type.
type EmbedTaskEmbedding[T any] struct {
	Metadata *T `json:"metadata,omitempty"`
}

// EmbedTaskError explains why an embedding task failed.
type EmbedTaskError struct {
	Message string `json:"message"`
}

// EmbedTask is an asynchronous embedding job.
type EmbedTask struct {
	ID                string                                         `json:"_id"`
	ModelName         string                                         `json:"model_name,omitempty"`
	Status            string                                         `json:"status"`
	CreatedAt         string                                         `json:"created_at,omitempty"`
	UpdatedAt         string                                         `json:"updated_at,omitempty"`
	Data              []Embedding                                    `json:"data"`
	Usage             *EmbeddingUsage                                `json:"usage,omitempty"`
	Metadata          *EmbedTaskMediaMetadata                        `json:"metadata,omitempty"`
	VideoEmbedding    *EmbedTaskEmbedding[VideoEmbeddingMetadata]    `json:"video_embedding,omitempty"`
	AudioEmbedding    *EmbedTaskEmbedding[AudioEmbeddingMetadata]    `json:"audio_embedding,omitempty"`
	DocumentEmbedding *EmbedTaskEmbedding[DocumentEmbeddingMetadata] `json:"document_embedding,omitempty"`
	ImageEmbedding    *EmbedTaskEmbedding[ImageEmbeddingMetadata]    `json:"image_embedding,omitempty"`
	Error             *EmbedTaskError                                `json:"error,omitempty"`
}

// EmbedTaskListOptions filters ListTasks.
type EmbedTaskListOptions struct {
	Pagination
	StartedAt string
	EndedAt   string
	Status    string
}

// Embed creates embeddings synchronously.
func (s *EmbedService) Embed(ctx context.Context, request *EmbedRequest) (*EmbedResponse, error) {
	ctx = ensureContext(ctx)
	if err := prepareEmbedRequest(request); err != nil {
		return nil, err
	}

	var response EmbedResponse
	if err := s.client.transport.doJSON(ctx, http.MethodPost, "/embed-v2", nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// CreateTask queues an asynchronous embedding task for long media.
func (s *EmbedService) CreateTask(ctx context.Context, request *EmbedTaskCreateRequest) (*EmbedTaskCreateResponse, error) {
	ctx = ensureContext(ctx)
	if err := prepareEmbedTaskRequest(request); err != nil {
		return nil, err
	}

	var response EmbedTaskCreateResponse
	if err := s.client.transport.doJSON(ctx, http.MethodPost, "/embed-v2/tasks", nil, request, &response); err != nil {
		return nil, err
	}
	if response.ID == "" {
		return nil, errEmptyID("embedding task")
	}
	return &response, nil
}

// ListTasks returns embedding tasks.
func (s *EmbedService) ListTasks(ctx context.Context, options *EmbedTaskListOptions) (*Page[EmbedTask], error) {
	ctx = ensureContext(ctx)
	query := url.Values{}
	if options != nil {
		query = encodeQuery(&options.Pagination)
		setIfNotEmpty(query, "started_at", options.StartedAt)
		setIfNotEmpty(query, "ended_at", options.EndedAt)
		setIfNotEmpty(query, "status", options.Status)
	}

	var page Page[EmbedTask]
	if err := s.client.transport.doJSON(ctx, http.MethodGet, "/embed-v2/tasks", query, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// GetTask retrieves an embedding task. Data is present once it is ready.
func (s *EmbedService) GetTask(ctx context.Context, taskID string) (*EmbedTask, error) {
	ctx = ensureContext(ctx)
	if taskID == "" {
		return nil, validationError("taskID", "is required")
	}
	var task EmbedTask
	path := "/embed-v2/tasks/" + escapePath(taskID)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, nil, nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// WaitForTask polls an embedding task until it is ready or failed.
func (s *EmbedService) WaitForTask(ctx context.Context, taskID string, options *WaitOptions) (*EmbedTask, error) {
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
		case EmbedTaskStatusReady:
			return task, nil
		case EmbedTaskStatusFailed:
			return task, fmt.Errorf("twelvelabs: embedding task %s failed: %s", taskID, task.Error)
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

func prepareEmbedRequest(request *EmbedRequest) error {
	if request == nil {
		return validationError("request", "is required")
	}
	if request.ModelName == "" {
		request.ModelName = ModelMarengo30
	}
	switch request.InputType {
	case EmbedInputText:
		if request.Text == nil || request.Text.InputText == "" {
			return validationError("Text", "InputText is required for input type %q", EmbedInputText)
		}
	case EmbedInputImage:
		if request.Image == nil {
			return validationError("Image", "is required for input type %q", EmbedInputImage)
		}
		if err := request.Image.MediaSource.validate("Image.MediaSource"); err != nil {
			return err
		}
	case EmbedInputTextImage:
		if request.TextImage == nil {
			return validationError("TextImage", "is required for input type %q", EmbedInputTextImage)
		}
		if request.TextImage.InputText == "" {
			return validationError("TextImage", "InputText is required")
		}
		if err := request.TextImage.MediaSource.validate("TextImage.MediaSource"); err != nil {
			return err
		}
	case EmbedInputAudio:
		return validateAudioVideoInput("Audio", request.Audio, request.InputType)
	case EmbedInputVideo:
		return validateAudioVideoInput("Video", request.Video, request.InputType)
	case EmbedInputMulti:
		if request.MultiInput == nil {
			return validationError("MultiInput", "is required for input type %q", EmbedInputMulti)
		}
		if request.MultiInput.InputText == "" && len(request.MultiInput.MediaSources) == 0 {
			return validationError("MultiInput", "set InputText, MediaSources, or both")
		}
		if len(request.MultiInput.MediaSources) > 10 {
			return validationError("MultiInput", "at most 10 media sources are allowed")
		}
		for i, source := range request.MultiInput.MediaSources {
			if source.MediaType == "" {
				return validationError("MultiInput", "MediaSources[%d].MediaType is required", i)
			}
			if err := validateMediaSourceFields("MultiInput", i, source.URL, source.AssetID, source.Base64String); err != nil {
				return err
			}
		}
	default:
		return validationError("InputType", "must be one of text, image, text_image, audio, video, or multi_input")
	}
	if request.EmbeddingDimension != "" && request.ModelName != ModelMarengo35 {
		return validationError("EmbeddingDimension", "requires model %q", ModelMarengo35)
	}
	return nil
}

func prepareEmbedTaskRequest(request *EmbedTaskCreateRequest) error {
	if request == nil {
		return validationError("request", "is required")
	}
	if request.ModelName == "" {
		request.ModelName = ModelMarengo30
	}
	switch request.InputType {
	case EmbedTaskInputAudio:
		return validateAudioVideoInput("Audio", request.Audio, request.InputType)
	case EmbedTaskInputVideo:
		return validateAudioVideoInput("Video", request.Video, request.InputType)
	case EmbedTaskInputDocument:
		if request.Document == nil {
			return validationError("Document", "is required for input type %q", EmbedTaskInputDocument)
		}
		if err := request.Document.MediaSource.validate("Document.MediaSource"); err != nil {
			return err
		}
		if request.ModelName != ModelMarengo35 {
			return validationError("InputType", "document inputs require model %q", ModelMarengo35)
		}
		if request.Document.Segmentation == nil {
			return validationError("Document", "Segmentation is required")
		}
		segmentation := request.Document.Segmentation
		if segmentation.Spatial == nil && segmentation.Sequential == nil {
			return validationError("Document", "Segmentation must set Spatial or Sequential")
		}
		if sequential := segmentation.Sequential; sequential != nil {
			if sequential.Strategy == "" {
				sequential.Strategy = "sentence"
			}
			if sequential.MaxSentences <= 0 {
				return validationError("Document", "Sequential.MaxSentences must be greater than zero")
			}
		}
	case EmbedTaskInputImage:
		if request.Image == nil {
			return validationError("Image", "is required for input type %q", EmbedTaskInputImage)
		}
		if err := request.Image.MediaSource.validate("Image.MediaSource"); err != nil {
			return err
		}
		if request.ModelName != ModelMarengo35 {
			return validationError("InputType", "image task inputs require model %q", ModelMarengo35)
		}
	default:
		return validationError("InputType", "must be one of audio, video, document, or image")
	}
	if request.EmbeddingDimension != "" && request.ModelName != ModelMarengo35 {
		return validationError("EmbeddingDimension", "requires model %q", ModelMarengo35)
	}
	return nil
}

func validateAudioVideoInput(field string, input *EmbedAudioVideoInput, inputType string) error {
	if input == nil {
		return validationError(field, "is required for input type %q", inputType)
	}
	if err := input.MediaSource.validate(field + ".MediaSource"); err != nil {
		return err
	}
	if input.StartSec != nil && input.EndSec != nil && *input.EndSec <= *input.StartSec {
		return validationError(field, "EndSec must be greater than StartSec")
	}
	return nil
}

func validateMediaSourceFields(field string, index int, url, assetID, base64String string) error {
	provided := 0
	for _, value := range []string{url, assetID, base64String} {
		if value != "" {
			provided++
		}
	}
	if provided != 1 {
		return validationError(field, "source %d must set exactly one of URL, AssetID, or Base64String", index)
	}
	return nil
}
