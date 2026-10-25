package twelvelabs

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
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
	EmbeddingOptionVisualText    = "visual-text"
	EmbeddingOptionText          = "text"
	EmbeddingOptionFused         = "fused"
)

// Embedding scope values returned by the API. EmbeddingScopeLocal is a request
// value; responses use clip, page, chunk, or asset.
const (
	EmbeddingScopeClip  = "clip"
	EmbeddingScopeAsset = "asset"
	// EmbeddingScopeLocal requests one embedding per segment, page, or chunk.
	EmbeddingScopeLocal = "local"
	// EmbeddingScopePage is returned for PDF pages.
	EmbeddingScopePage = "page"
	// EmbeddingScopeChunk is returned for plain text and Markdown chunks.
	EmbeddingScopeChunk = "chunk"
)

// Page quadrant values returned for PDFs segmented with the quadrants
// strategy.
const (
	QuadrantTopLeft     = "top_left"
	QuadrantTopRight    = "top_right"
	QuadrantBottomLeft  = "bottom_left"
	QuadrantBottomRight = "bottom_right"
)

// Document segmentation strategies.
const (
	DocumentSpatialStandard    = "standard"
	DocumentSpatialQuadrants   = "quadrants"
	DocumentSequentialSentence = "sentence"
)

// Media types accepted by multi-input embeddings.
const (
	EmbedMediaImage    = "image"
	EmbedMediaVideo    = "video"
	EmbedMediaAudio    = "audio"
	EmbedMediaDocument = "document"
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
	// TimeBasedMetadata folds time-aligned text into fused embeddings. It is
	// asynchronous-only, requires Marengo 3.5, and requires
	// EmbeddingType to contain fused_embedding.
	TimeBasedMetadata []TimeBasedMetadataEntry `json:"time_based_metadata,omitempty"`
}

// EmbedImageTaskInput embeds an image asynchronously.
type EmbedImageTaskInput struct {
	MediaSource     MediaSource `json:"media_source"`
	EmbeddingOption []string    `json:"embedding_option,omitempty"`
	EmbeddingScope  []string    `json:"embedding_scope,omitempty"`
	EmbeddingType   []string    `json:"embedding_type,omitempty"`
}

// DocumentSpatialSegmentation divides each page of a PDF. It requires the
// visual embedding option and the local embedding scope.
type DocumentSpatialSegmentation struct {
	// Strategy is DocumentSpatialStandard (the default) or
	// DocumentSpatialQuadrants. Quadrants returns five embeddings per page.
	Strategy string `json:"strategy,omitempty"`
}

// DocumentSequentialSegmentation divides plain text or Markdown into chunks. It
// requires the text embedding option and the local embedding scope.
type DocumentSequentialSegmentation struct {
	// Strategy is always DocumentSequentialSentence.
	Strategy string `json:"strategy"`
	// MaxSentences caps the sentences per chunk. Required by the API.
	MaxSentences int `json:"max_sentences"`
	// OverlapSentences repeats this many sentences between chunks. It must be
	// non-negative and smaller than MaxSentences.
	OverlapSentences int `json:"overlap_sentences,omitempty"`
}

// DocumentSegmentation divides a document before embedding.
type DocumentSegmentation struct {
	Spatial    *DocumentSpatialSegmentation    `json:"spatial,omitempty"`
	Sequential *DocumentSequentialSegmentation `json:"sequential,omitempty"`
}

// EmbedDocumentInput embeds a PDF, plain text, or Markdown file. Marengo 3.5
// only, asynchronous endpoint only.
//
// PDF files support the visual and text embedding options; plain text and
// Markdown files support text only. The spatial segmentation strategy is for
// PDFs, and the sequential strategy is for plain text and Markdown.
type EmbedDocumentInput struct {
	MediaSource     MediaSource           `json:"media_source"`
	Segmentation    *DocumentSegmentation `json:"segmentation,omitempty"`
	EmbeddingOption []string              `json:"embedding_option,omitempty"`
	EmbeddingScope  []string              `json:"embedding_scope,omitempty"`
	EmbeddingType   []string              `json:"embedding_type,omitempty"`
}

// EmbedMultiInput combines text and up to 10 media sources into one embedding.
// With Marengo 3.5 the text can be up to 2,000 tokens; AutoTruncate controls
// whether longer text is truncated or rejected.
type EmbedMultiInput struct {
	// InputText may reference a media source with <@name>. Every reference
	// must match a source Name.
	InputText string `json:"input_text,omitempty"`
	// MediaSources contains at most 10 sources. Marengo 3.0 accepts images
	// only; Marengo 3.5 also accepts audio, video, and documents.
	MediaSources []MultiInputMediaSource `json:"media_sources,omitempty"`
}

// MultiInputMediaSource is one media source of a multi-input embedding.
type MultiInputMediaSource struct {
	// Name is unique within the request. It is required when InputText
	// references this source.
	Name string `json:"name,omitempty"`
	// MediaType is EmbedMediaImage, EmbedMediaVideo, EmbedMediaAudio, or
	// EmbedMediaDocument.
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
	// AutoTruncate truncates text longer than 2,000 tokens instead of failing.
	// Marengo 3.5 only.
	AutoTruncate bool `json:"auto_truncate,omitempty"`
	// EmbeddingUncertainty returns a per-dimension uncertainty vector. Marengo
	// 3.5 only, and only for a text-only or media-only request that does not
	// contain a document.
	EmbeddingUncertainty bool `json:"embedding_uncertainty,omitempty"`
	// EmbeddingDimension is one of the EmbeddingDimensions* constants. Marengo
	// 3.5 only. The API defaults to 512.
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
	// ModelName is ModelMarengo30 (default) or ModelMarengo35. Audio and video
	// support both; document and image inputs require ModelMarengo35.
	ModelName string `json:"model_name,omitempty"`
	// EmbeddingUncertainty returns a per-dimension uncertainty vector. Marengo
	// 3.5 only. Audio and video must exclude the asset scope; PDFs are
	// rejected; plain text and Markdown require the local scope.
	EmbeddingUncertainty bool `json:"embedding_uncertainty,omitempty"`
	// EmbeddingDimension is one of the EmbeddingDimensions* constants. Marengo
	// 3.5 only. The API defaults to 512.
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
	// InputType is audio, video, document, or image.
	InputType          string   `json:"input_type,omitempty"`
	InputURL           string   `json:"input_url,omitempty"`
	InputFilename      string   `json:"input_filename,omitempty"`
	EmbeddingOptions   []string `json:"embedding_options,omitempty"`
	EmbeddingScopes    []string `json:"embedding_scopes,omitempty"`
	Duration           float64  `json:"duration,omitempty"`
	StartOffsetSec     float64  `json:"start_offset_sec,omitempty"`
	EndOffsetSec       float64  `json:"end_offset_sec,omitempty"`
	EmbeddingDimension int      `json:"embedding_dimension,omitempty"`
	// ClipLength is the fixed-segment length in seconds for video input.
	ClipLength int `json:"clip_length,omitempty"`
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

// EmbedTask is an asynchronous embedding job. GetTask returns Data, Usage, and
// Metadata when the task is ready. ListTasks returns the same type, populated
// with the per-modality VideoEmbedding, AudioEmbedding, DocumentEmbedding, or
// ImageEmbedding metadata instead.
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
	// StartedAt and EndedAt are RFC 3339 timestamps.
	StartedAt string
	EndedAt   string
	// Status is processing, ready, or failed.
	Status string
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
		switch options.Status {
		case "", EmbedTaskStatusProcessing, EmbedTaskStatusReady, EmbedTaskStatusFailed:
		default:
			return nil, validationError("Status", "must be %q, %q, or %q", EmbedTaskStatusProcessing, EmbedTaskStatusReady, EmbedTaskStatusFailed)
		}
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
			message := "the platform did not report a reason"
			if task.Error != nil && task.Error.Message != "" {
				message = task.Error.Message
			}
			return task, fmt.Errorf("twelvelabs: embedding task %s failed: %s", taskID, message)
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
	if err := validateEmbedModel(request.ModelName); err != nil {
		return err
	}
	if err := validateEmbeddingDimension(request.EmbeddingDimension); err != nil {
		return err
	}
	if request.EmbeddingDimension != "" && request.ModelName != ModelMarengo35 {
		return validationError("EmbeddingDimension", "requires model %q", ModelMarengo35)
	}
	if request.AutoTruncate && request.ModelName != ModelMarengo35 {
		return validationError("AutoTruncate", "requires model %q", ModelMarengo35)
	}
	if request.EmbeddingUncertainty && request.ModelName != ModelMarengo35 {
		return validationError("EmbeddingUncertainty", "requires model %q", ModelMarengo35)
	}

	set := 0
	for _, isSet := range []bool{
		request.Text != nil,
		request.Image != nil,
		request.TextImage != nil,
		request.Audio != nil,
		request.Video != nil,
		request.MultiInput != nil,
	} {
		if isSet {
			set++
		}
	}
	if set != 1 {
		return validationError("InputType", "set exactly the field that matches InputType and leave the others nil")
	}

	switch request.InputType {
	case EmbedInputText:
		if request.ModelName != ModelMarengo30 {
			return unsupportedSyncEmbedInput(request.InputType)
		}
		if request.Text == nil || strings.TrimSpace(request.Text.InputText) == "" {
			return validationError("Text", "InputText is required for input type %q", EmbedInputText)
		}
	case EmbedInputImage:
		if request.ModelName != ModelMarengo30 {
			return unsupportedSyncEmbedInput(request.InputType)
		}
		if request.Image == nil {
			return validationError("Image", "is required for input type %q", EmbedInputImage)
		}
		if err := request.Image.MediaSource.validate("Image.MediaSource"); err != nil {
			return err
		}
	case EmbedInputTextImage:
		if request.ModelName != ModelMarengo30 {
			return unsupportedSyncEmbedInput(request.InputType)
		}
		if request.TextImage == nil {
			return validationError("TextImage", "is required for input type %q", EmbedInputTextImage)
		}
		if strings.TrimSpace(request.TextImage.InputText) == "" {
			return validationError("TextImage", "InputText is required")
		}
		if err := request.TextImage.MediaSource.validate("TextImage.MediaSource"); err != nil {
			return err
		}
	case EmbedInputAudio:
		if request.ModelName != ModelMarengo30 {
			return unsupportedSyncEmbedInput(request.InputType)
		}
		if err := validateEmbedMediaInput("Audio", request.Audio, EmbedInputAudio, request.ModelName, false); err != nil {
			return err
		}
	case EmbedInputVideo:
		if request.ModelName != ModelMarengo30 {
			return unsupportedSyncEmbedInput(request.InputType)
		}
		if err := validateEmbedMediaInput("Video", request.Video, EmbedInputVideo, request.ModelName, false); err != nil {
			return err
		}
	case EmbedInputMulti:
		if err := validateMultiInput("MultiInput", request.MultiInput, request.ModelName); err != nil {
			return err
		}
	default:
		return validationError("InputType", "must be one of text, image, text_image, audio, video, or multi_input")
	}

	if request.EmbeddingUncertainty {
		return validateSyncEmbeddingUncertainty(request)
	}
	return nil
}

func unsupportedSyncEmbedInput(inputType string) error {
	return validationError("InputType", "input type %q requires model %q; the synchronous endpoint accepts only %q with model %q", inputType, ModelMarengo30, EmbedInputMulti, ModelMarengo35)
}

func prepareEmbedTaskRequest(request *EmbedTaskCreateRequest) error {
	if request == nil {
		return validationError("request", "is required")
	}
	if request.ModelName == "" {
		request.ModelName = ModelMarengo30
	}
	if err := validateEmbedModel(request.ModelName); err != nil {
		return err
	}
	if err := validateEmbeddingDimension(request.EmbeddingDimension); err != nil {
		return err
	}
	if request.EmbeddingDimension != "" && request.ModelName != ModelMarengo35 {
		return validationError("EmbeddingDimension", "requires model %q", ModelMarengo35)
	}
	if request.EmbeddingUncertainty && request.ModelName != ModelMarengo35 {
		return validationError("EmbeddingUncertainty", "requires model %q", ModelMarengo35)
	}

	set := 0
	for _, isSet := range []bool{
		request.Audio != nil,
		request.Video != nil,
		request.Document != nil,
		request.Image != nil,
	} {
		if isSet {
			set++
		}
	}
	if set != 1 {
		return validationError("InputType", "set exactly the field that matches InputType and leave the others nil")
	}

	switch request.InputType {
	case EmbedTaskInputAudio:
		if err := validateEmbedMediaInput("Audio", request.Audio, EmbedTaskInputAudio, request.ModelName, true); err != nil {
			return err
		}
	case EmbedTaskInputVideo:
		if err := validateEmbedMediaInput("Video", request.Video, EmbedTaskInputVideo, request.ModelName, true); err != nil {
			return err
		}
	case EmbedTaskInputDocument:
		if request.ModelName != ModelMarengo35 {
			return validationError("InputType", "document inputs require model %q", ModelMarengo35)
		}
		if err := validateEmbedDocumentInput("Document", request.Document, request.ModelName); err != nil {
			return err
		}
	case EmbedTaskInputImage:
		if request.ModelName != ModelMarengo35 {
			return validationError("InputType", "image task inputs require model %q", ModelMarengo35)
		}
		if err := validateEmbedImageTaskInput("Image", request.Image, request.ModelName); err != nil {
			return err
		}
	default:
		return validationError("InputType", "must be one of audio, video, document, or image")
	}

	if request.EmbeddingUncertainty {
		return validateTaskEmbeddingUncertainty(request)
	}
	return nil
}

func validateEmbedModel(model string) error {
	switch model {
	case ModelMarengo30, ModelMarengo35:
		return nil
	default:
		return validationError("ModelName", "must be %q or %q", ModelMarengo30, ModelMarengo35)
	}
}

func validateEmbeddingDimension(dimension string) error {
	switch dimension {
	case "", EmbeddingDimensions128, EmbeddingDimensions256, EmbeddingDimensions512:
		return nil
	default:
		return validationError("EmbeddingDimension", "must be %q, %q, or %q", EmbeddingDimensions128, EmbeddingDimensions256, EmbeddingDimensions512)
	}
}

func validateEmbedMediaInput(field string, input *EmbedAudioVideoInput, inputType, model string, async bool) error {
	if input == nil {
		return validationError(field, "is required for input type %q", inputType)
	}
	if err := input.MediaSource.validate(field + ".MediaSource"); err != nil {
		return err
	}
	if err := validateEmbedTimeRange(field, input.StartSec, input.EndSec); err != nil {
		return err
	}

	var options, scopes []string
	switch inputType {
	case EmbedInputAudio:
		options = []string{EmbeddingOptionAudio, EmbeddingOptionTranscription}
		if model == ModelMarengo35 {
			options = []string{EmbeddingOptionAudio}
		}
		scopes = []string{EmbeddingScopeClip, EmbeddingScopeAsset}
	case EmbedInputVideo:
		options = []string{EmbeddingOptionVisual, EmbeddingOptionAudio, EmbeddingOptionTranscription}
		if model == ModelMarengo35 {
			options = []string{EmbeddingOptionVisual, EmbeddingOptionAudio}
		}
		scopes = []string{EmbeddingScopeClip, EmbeddingScopeAsset}
	default:
		return validationError(field, "has an unsupported input type %q", inputType)
	}
	if async {
		scopes = []string{EmbeddingScopeClip, EmbeddingScopeLocal, EmbeddingScopeAsset}
	}

	if err := validateEmbedValues(field+".EmbeddingOption", input.EmbeddingOption, options); err != nil {
		return err
	}
	if err := validateEmbedValues(field+".EmbeddingScope", input.EmbeddingScope, scopes); err != nil {
		return err
	}
	if err := validateEmbedTypes(field+".EmbeddingType", input.EmbeddingType, input.EmbeddingOption, inputType, model, async, input.TimeBasedMetadata); err != nil {
		return err
	}
	if err := validateEmbedSegmentation(field+".Segmentation", input.Segmentation, model, inputType, async); err != nil {
		return err
	}

	if !async {
		if len(input.TimeBasedMetadata) > 0 {
			return validationError(field+".TimeBasedMetadata", "is only supported by the asynchronous endpoint")
		}
		return nil
	}
	if len(input.TimeBasedMetadata) > 0 {
		if model != ModelMarengo35 {
			return validationError(field+".TimeBasedMetadata", "requires model %q", ModelMarengo35)
		}
		if !containsEmbedValue(input.EmbeddingType, EmbeddingTypeFused) {
			return validationError(field+".TimeBasedMetadata", "requires %q in %s.EmbeddingType", EmbeddingTypeFused, field)
		}
		for i, entry := range input.TimeBasedMetadata {
			prefix := fmt.Sprintf("%s.TimeBasedMetadata[%d]", field, i)
			if math.IsNaN(entry.Start) || math.IsInf(entry.Start, 0) || math.IsNaN(entry.End) || math.IsInf(entry.End, 0) {
				return validationError(prefix, "times must be finite numbers")
			}
			if entry.Start < 0 {
				return validationError(prefix, "Start must not be negative")
			}
			if entry.End < entry.Start {
				return validationError(prefix, "End must be greater than or equal to Start")
			}
			if strings.TrimSpace(entry.Text) == "" {
				return validationError(prefix, "Text is required")
			}
		}
	}
	if model == ModelMarengo35 && containsEmbedValue(input.EmbeddingType, EmbeddingTypeFused) && len(input.TimeBasedMetadata) == 0 {
		return validationError(field+".TimeBasedMetadata", "is required when EmbeddingType contains %q with model %q", EmbeddingTypeFused, ModelMarengo35)
	}
	return nil
}

func validateEmbedTypes(field string, types, options []string, inputType, model string, async bool, metadata []TimeBasedMetadataEntry) error {
	allowed := []string{EmbeddingTypeSeparate, EmbeddingTypeFused}
	if err := validateEmbedValues(field, types, allowed); err != nil {
		return err
	}
	if len(types) == 0 {
		return nil
	}
	optionCount := len(options)
	if optionCount == 0 {
		// Apply the API defaults for the model, because EmbeddingType is only
		// valid when at least two embedding options are in effect.
		optionCount = 2
		if model == ModelMarengo35 && inputType == EmbedInputAudio {
			optionCount = 1
		}
	}
	if optionCount < 2 {
		return validationError(field, "is only valid when EmbeddingOption contains at least two values")
	}
	if !async && len(metadata) > 0 {
		return validationError(field, "TimeBasedMetadata is only supported by the asynchronous endpoint")
	}
	return nil
}

func validateEmbedSegmentation(field string, segmentation *Segmentation, model, mediaType string, async bool) error {
	if segmentation == nil {
		return nil
	}
	if segmentation.Temporal != nil {
		if model != ModelMarengo35 {
			return validationError(field+".Temporal", "requires model %q", ModelMarengo35)
		}
		if segmentation.Strategy != "" || segmentation.Fixed != nil || segmentation.Dynamic != nil {
			return validationError(field, "when Temporal is set, put the strategy and its settings inside Temporal")
		}
		return validateEmbedSegmentationLevel(field+".Temporal", segmentation.Temporal, model, mediaType)
	}
	if async && model == ModelMarengo35 {
		return validationError(field, "Marengo 3.5 requires the segmentation settings to be nested under Temporal")
	}
	return validateEmbedSegmentationLevel(field, segmentation, model, mediaType)
}

func validateEmbedSegmentationLevel(field string, segmentation *Segmentation, model, mediaType string) error {
	if segmentation == nil {
		return validationError(field, "is required")
	}
	if segmentation.Temporal != nil {
		return validationError(field+".Temporal", "cannot be nested more than once")
	}
	allowDynamic := mediaType == EmbedInputVideo || model == ModelMarengo35
	switch segmentation.Strategy {
	case SegmentationStrategyFixed:
		if segmentation.Fixed == nil {
			return validationError(field+".Fixed", "is required when Strategy is %q", SegmentationStrategyFixed)
		}
		if segmentation.Dynamic != nil {
			return validationError(field+".Dynamic", "must be nil when Strategy is %q", SegmentationStrategyFixed)
		}
		if d := segmentation.Fixed.DurationSec; d < 2 || d > 10 {
			return validationError(field+".Fixed.DurationSec", "must be between 2 and 10 seconds")
		}
	case SegmentationStrategyDynamic:
		if !allowDynamic {
			return validationError(field+".Strategy", "dynamic segmentation is not supported for audio with model %q", ModelMarengo30)
		}
		if segmentation.Dynamic == nil {
			return validationError(field+".Dynamic", "is required when Strategy is %q", SegmentationStrategyDynamic)
		}
		if segmentation.Fixed != nil {
			return validationError(field+".Fixed", "must be nil when Strategy is %q", SegmentationStrategyDynamic)
		}
		if d := segmentation.Dynamic.MinDurationSec; d < 2 || d > 5 {
			return validationError(field+".Dynamic.MinDurationSec", "must be between 2 and 5 seconds")
		}
	case "":
		return validationError(field+".Strategy", "is required when Segmentation is set")
	default:
		return validationError(field+".Strategy", "must be %q or %q", SegmentationStrategyFixed, SegmentationStrategyDynamic)
	}
	return nil
}

func validateEmbedTimeRange(field string, start, end *float64) error {
	for name, value := range map[string]*float64{"StartSec": start, "EndSec": end} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0)) {
			return validationError(field+"."+name, "must be a finite number")
		}
	}
	if start != nil && *start < 0 {
		return validationError(field+".StartSec", "must not be negative")
	}
	if start != nil && end != nil && *end <= *start {
		return validationError(field+".EndSec", "must be greater than StartSec")
	}
	return nil
}

func validateEmbedValues(field string, values, allowed []string) error {
	allowedSet := make(map[string]bool, len(allowed))
	for _, value := range allowed {
		allowedSet[value] = true
	}
	seen := make(map[string]bool, len(values))
	for i, value := range values {
		if !allowedSet[value] {
			return validationError(field, "[%d] must be one of %s", i, strings.Join(allowed, ", "))
		}
		if seen[value] {
			return validationError(field, "[%d] duplicates %q", i, value)
		}
		seen[value] = true
	}
	return nil
}

func containsEmbedValue(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func validateSyncEmbeddingUncertainty(request *EmbedRequest) error {
	// EmbeddingUncertainty requires Marengo 3.5, whose synchronous endpoint
	// accepts only multi_input.
	if request.MultiInput == nil {
		return nil
	}
	hasText := strings.TrimSpace(request.MultiInput.InputText) != ""
	hasMedia := len(request.MultiInput.MediaSources) > 0
	if hasText && hasMedia {
		return validationError("EmbeddingUncertainty", "is only supported for text-only or media-only requests")
	}
	for i, source := range request.MultiInput.MediaSources {
		if source.MediaType == EmbedMediaDocument {
			return validationError("EmbeddingUncertainty", "is not supported when MediaSources[%d] is a document", i)
		}
	}
	return nil
}

func validateTaskEmbeddingUncertainty(request *EmbedTaskCreateRequest) error {
	switch request.InputType {
	case EmbedTaskInputAudio:
		return requireNoAssetScope("Audio", request.Audio)
	case EmbedTaskInputVideo:
		return requireNoAssetScope("Video", request.Video)
	case EmbedTaskInputDocument:
		if request.Document == nil {
			return nil
		}
		if classifyEmbedDocument(request.Document.MediaSource) == embedDocumentPDF {
			return validationError("EmbeddingUncertainty", "is not supported for PDF documents")
		}
		scopes := request.Document.EmbeddingScope
		if len(scopes) != 1 || scopes[0] != EmbeddingScopeLocal {
			return validationError("Document.EmbeddingScope", "must be [%q] when EmbeddingUncertainty is true", EmbeddingScopeLocal)
		}
	}
	return nil
}

func requireNoAssetScope(field string, input *EmbedAudioVideoInput) error {
	if input == nil {
		return nil
	}
	if len(input.EmbeddingScope) == 0 {
		return validationError("EmbeddingUncertainty", "requires %s.EmbeddingScope to exclude %q; the default includes it", field, EmbeddingScopeAsset)
	}
	if containsEmbedValue(input.EmbeddingScope, EmbeddingScopeAsset) {
		return validationError("EmbeddingUncertainty", "requires %s.EmbeddingScope to exclude %q", field, EmbeddingScopeAsset)
	}
	return nil
}

func validateMultiInput(field string, multi *EmbedMultiInput, model string) error {
	if multi == nil {
		return validationError(field, "is required for input type %q", EmbedInputMulti)
	}
	if strings.TrimSpace(multi.InputText) == "" && len(multi.MediaSources) == 0 {
		return validationError(field, "set InputText, MediaSources, or both")
	}
	if len(multi.MediaSources) > 10 {
		return validationError(field, "at most 10 media sources are allowed")
	}

	names := make(map[string]bool, len(multi.MediaSources))
	mediaTypes := make(map[string]bool, 4)
	nonDocumentSources := 0
	textDocuments := 0
	hasPDF := false
	for i := range multi.MediaSources {
		source := &multi.MediaSources[i]
		prefix := fmt.Sprintf("%s.MediaSources[%d]", field, i)
		switch source.MediaType {
		case EmbedMediaImage, EmbedMediaVideo, EmbedMediaAudio, EmbedMediaDocument:
		case "":
			return validationError(prefix, "MediaType is required")
		default:
			return validationError(prefix, "MediaType must be %q, %q, %q, or %q", EmbedMediaImage, EmbedMediaVideo, EmbedMediaAudio, EmbedMediaDocument)
		}
		if model == ModelMarengo30 && source.MediaType != EmbedMediaImage {
			return validationError(prefix, "media type %q requires model %q", source.MediaType, ModelMarengo35)
		}
		if err := validateMediaSourceFields(prefix, source.URL, source.AssetID, source.Base64String); err != nil {
			return err
		}
		name := strings.TrimSpace(source.Name)
		if name != "" {
			if names[name] {
				return validationError(prefix, "Name %q is duplicated", source.Name)
			}
			names[name] = true
		}
		mediaTypes[source.MediaType] = true
		if source.MediaType == EmbedMediaDocument {
			switch classifyEmbedDocument(MediaSource{URL: source.URL, AssetID: source.AssetID, Base64String: source.Base64String}) {
			case embedDocumentPDF:
				hasPDF = true
			case embedDocumentText:
				textDocuments++
			}
		} else {
			nonDocumentSources++
		}
	}

	if hasPDF && (len(multi.MediaSources) != 1 || strings.TrimSpace(multi.InputText) != "") {
		return validationError(field, "a PDF document must be the only source and cannot be combined with InputText")
	}
	if textDocuments > 0 && nonDocumentSources == 0 {
		if len(multi.MediaSources) > 1 {
			return validationError(field, "an all-text request must contain a single text document or InputText, not both")
		}
		if strings.TrimSpace(multi.InputText) != "" {
			return validationError(field, "a text document cannot be combined with InputText; add an image, audio, or video source instead")
		}
	}
	if len(mediaTypes) > 1 && strings.TrimSpace(multi.InputText) == "" {
		return validationError(field, "InputText is required when combining media sources of different types")
	}

	for _, match := range multiInputReferencePattern.FindAllStringSubmatch(multi.InputText, -1) {
		reference := strings.TrimSpace(match[1])
		if reference == "" || !names[reference] {
			return validationError(field+".InputText", "reference <@%s> has no matching media source Name", match[1])
		}
	}
	return nil
}

var multiInputReferencePattern = regexp.MustCompile(`<@([^<>]+)>`)

func validateEmbedImageTaskInput(field string, input *EmbedImageTaskInput, model string) error {
	if input == nil {
		return validationError(field, "is required for input type %q", EmbedTaskInputImage)
	}
	if err := input.MediaSource.validate(field + ".MediaSource"); err != nil {
		return err
	}
	if err := validateSingleEmbedValue(field+".EmbeddingOption", input.EmbeddingOption, EmbeddingOptionVisual); err != nil {
		return err
	}
	if err := validateSingleEmbedValue(field+".EmbeddingType", input.EmbeddingType, EmbeddingTypeSeparate); err != nil {
		return err
	}
	return validateSingleEmbedValue(field+".EmbeddingScope", input.EmbeddingScope, EmbeddingScopeAsset)
}

func validateSingleEmbedValue(field string, values []string, want string) error {
	if len(values) == 0 {
		return nil
	}
	if len(values) != 1 || values[0] != want {
		return validationError(field, "must be [%q] for image embeddings", want)
	}
	return nil
}

type embedDocumentKind int

const (
	embedDocumentUnknown embedDocumentKind = iota
	embedDocumentPDF
	embedDocumentText
)

func classifyEmbedDocument(source MediaSource) embedDocumentKind {
	if source.URL == "" {
		// Asset IDs and base64 data carry no filename.
		return embedDocumentUnknown
	}
	path := source.URL
	if parsed, err := url.Parse(path); err == nil && parsed.Path != "" {
		path = parsed.Path
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		return embedDocumentPDF
	case ".txt", ".md", ".markdown":
		return embedDocumentText
	default:
		return embedDocumentUnknown
	}
}

func validateEmbedDocumentInput(field string, input *EmbedDocumentInput, model string) error {
	if input == nil {
		return validationError(field, "is required for input type %q", EmbedTaskInputDocument)
	}
	if model != ModelMarengo35 {
		return validationError(field, "document inputs require model %q", ModelMarengo35)
	}
	if err := input.MediaSource.validate(field + ".MediaSource"); err != nil {
		return err
	}
	kind := classifyEmbedDocument(input.MediaSource)

	var spatial *DocumentSpatialSegmentation
	var sequential *DocumentSequentialSegmentation
	if input.Segmentation != nil {
		if input.Segmentation.Spatial == nil && input.Segmentation.Sequential == nil {
			return validationError(field+".Segmentation", "set Spatial for a PDF or Sequential for plain text or Markdown")
		}
		if input.Segmentation.Spatial != nil && input.Segmentation.Sequential != nil {
			return validationError(field+".Segmentation", "set either Spatial or Sequential, not both")
		}
		spatial = input.Segmentation.Spatial
		sequential = input.Segmentation.Sequential
	}
	if spatial != nil && kind == embedDocumentUnknown {
		kind = embedDocumentPDF
	}
	if sequential != nil && kind == embedDocumentUnknown {
		kind = embedDocumentText
	}
	if spatial != nil {
		if kind == embedDocumentText {
			return validationError(field+".Segmentation.Spatial", "is only valid for PDF documents")
		}
		switch spatial.Strategy {
		case "", DocumentSpatialStandard, DocumentSpatialQuadrants:
		default:
			return validationError(field+".Segmentation.Spatial.Strategy", "must be %q or %q", DocumentSpatialStandard, DocumentSpatialQuadrants)
		}
		if len(input.EmbeddingOption) > 0 && !containsEmbedValue(input.EmbeddingOption, EmbeddingOptionVisual) {
			return validationError(field+".Segmentation.Spatial", "requires %q in EmbeddingOption", EmbeddingOptionVisual)
		}
		if len(input.EmbeddingScope) > 0 && !containsEmbedValue(input.EmbeddingScope, EmbeddingScopeLocal) {
			return validationError(field+".Segmentation.Spatial", "requires %q in EmbeddingScope", EmbeddingScopeLocal)
		}
	}
	if sequential != nil {
		if kind == embedDocumentPDF {
			return validationError(field+".Segmentation.Sequential", "is only valid for plain text and Markdown documents")
		}
		if sequential.Strategy == "" {
			sequential.Strategy = DocumentSequentialSentence
		}
		if sequential.Strategy != DocumentSequentialSentence {
			return validationError(field+".Segmentation.Sequential.Strategy", "must be %q", DocumentSequentialSentence)
		}
		if sequential.MaxSentences <= 0 {
			return validationError(field+".Segmentation.Sequential.MaxSentences", "must be greater than zero")
		}
		if sequential.OverlapSentences < 0 || sequential.OverlapSentences >= sequential.MaxSentences {
			return validationError(field+".Segmentation.Sequential.OverlapSentences", "must be non-negative and smaller than MaxSentences")
		}
		if len(input.EmbeddingOption) > 0 && !containsEmbedValue(input.EmbeddingOption, EmbeddingOptionText) {
			return validationError(field+".Segmentation.Sequential", "requires %q in EmbeddingOption", EmbeddingOptionText)
		}
		if !containsEmbedValue(input.EmbeddingScope, EmbeddingScopeLocal) {
			return validationError(field+".Segmentation.Sequential", "requires %q in EmbeddingScope", EmbeddingScopeLocal)
		}
	}

	if err := validateEmbedValues(field+".EmbeddingOption", input.EmbeddingOption, []string{EmbeddingOptionVisual, EmbeddingOptionText}); err != nil {
		return err
	}
	if err := validateEmbedValues(field+".EmbeddingScope", input.EmbeddingScope, []string{EmbeddingScopeLocal, EmbeddingScopeAsset}); err != nil {
		return err
	}
	if err := validateEmbedValues(field+".EmbeddingType", input.EmbeddingType, []string{EmbeddingTypeSeparate}); err != nil {
		return err
	}

	if kind == embedDocumentText {
		if containsEmbedValue(input.EmbeddingOption, EmbeddingOptionVisual) {
			return validationError(field+".EmbeddingOption", "%q is only valid for PDF documents", EmbeddingOptionVisual)
		}
		if containsEmbedValue(input.EmbeddingScope, EmbeddingScopeLocal) && sequential == nil {
			return validationError(field+".Segmentation.Sequential", "is required for %q scope on plain text and Markdown documents", EmbeddingScopeLocal)
		}
	}
	if kind == embedDocumentPDF && containsEmbedValue(input.EmbeddingOption, EmbeddingOptionText) {
		if len(input.EmbeddingScope) == 0 {
			return validationError(field+".EmbeddingScope", "must include %q when a PDF uses %q", EmbeddingScopeAsset, EmbeddingOptionText)
		}
		if containsEmbedValue(input.EmbeddingScope, EmbeddingScopeLocal) {
			return validationError(field+".EmbeddingScope", "%q cannot be combined with %q for a PDF", EmbeddingScopeLocal, EmbeddingOptionText)
		}
	}
	return nil
}

func validateMediaSourceFields(field, url, assetID, base64String string) error {
	provided := 0
	for _, value := range []string{url, assetID, base64String} {
		if value != "" {
			provided++
		}
	}
	if provided != 1 {
		return validationError(field, "must set exactly one of URL, AssetID, or Base64String")
	}
	return nil
}
