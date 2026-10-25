package twelvelabs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
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

var (
	customIDPattern  = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	imageNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{1,50}$`)
)

// TokenLimit is a response token limit that the API reports either as a number
// or as the literal "unlimited". The zero value means the field was absent.
type TokenLimit struct {
	Value     int
	Unlimited bool
}

// IsUnlimited reports whether the API removed the token limit.
func (t TokenLimit) IsUnlimited() bool { return t.Unlimited }

// MarshalJSON encodes a numeric limit or "unlimited".
func (t TokenLimit) MarshalJSON() ([]byte, error) {
	if t.Unlimited {
		return []byte(`"unlimited"`), nil
	}
	return json.Marshal(t.Value)
}

// UnmarshalJSON decodes a numeric limit or "unlimited".
func (t *TokenLimit) UnmarshalJSON(data []byte) error {
	if strings.TrimSpace(string(data)) == `"unlimited"` {
		t.Value = 0
		t.Unlimited = true
		return nil
	}
	var value int
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	t.Value = value
	t.Unlimited = false
	return nil
}

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

type responseFormatOptions struct {
	modelName               string
	allowSegmentDefinitions bool
	allowTimeArray          bool
}

func (f *ResponseFormat) validate(field string, options responseFormatOptions) error {
	if f == nil {
		return nil
	}
	switch f.Type {
	case ResponseFormatJSONSchema:
		if len(f.JSONSchema) == 0 {
			return validationError(field, "JSONSchema is required when Type is %q", ResponseFormatJSONSchema)
		}
		if len(f.SegmentDefinitions) > 0 || f.SegmentTimeFormat != "" {
			return validationError(field, "SegmentDefinitions and SegmentTimeFormat are only valid with Type %q", ResponseFormatSegmentDefinitions)
		}
	case ResponseFormatSegmentDefinitions:
		if !options.allowSegmentDefinitions {
			return validationError(field, "Type %q is only supported by asynchronous time-based analysis", ResponseFormatSegmentDefinitions)
		}
		if len(f.JSONSchema) > 0 {
			return validationError(field, "JSONSchema is only valid with Type %q", ResponseFormatJSONSchema)
		}
		if len(f.SegmentDefinitions) == 0 {
			return validationError(field, "at least one SegmentDefinition is required when Type is %q", ResponseFormatSegmentDefinitions)
		}
		switch f.SegmentTimeFormat {
		case "", TimeFormatSeconds, TimeFormatHHMMSS, TimeFormatHHMMSSMillis:
		default:
			return validationError(field, "SegmentTimeFormat must be %q, %q, or %q", TimeFormatSeconds, TimeFormatHHMMSS, TimeFormatHHMMSSMillis)
		}
		if err := validateSegmentDefinitions(field+".SegmentDefinitions", f.SegmentDefinitions, options); err != nil {
			return err
		}
	default:
		return validationError(field, "Type must be %q or %q", ResponseFormatJSONSchema, ResponseFormatSegmentDefinitions)
	}
	return nil
}

func validateSegmentDefinitions(field string, definitions []SegmentDefinition, options responseFormatOptions) error {
	ids := make(map[string]bool, len(definitions))
	for i := range definitions {
		definition := &definitions[i]
		prefix := fmt.Sprintf("%s[%d]", field, i)
		if definition.ID == "" {
			return validationError(prefix, "ID is required")
		}
		if len(definition.ID) > 100 {
			return validationError(prefix, "ID must be at most 100 characters")
		}
		if ids[definition.ID] {
			return validationError(prefix, "ID %q is duplicated", definition.ID)
		}
		ids[definition.ID] = true
		if definition.Description == "" {
			return validationError(prefix, "Description is required")
		}
		if len(definition.MediaSources) > 4 {
			return validationError(prefix, "at most 4 media sources are allowed")
		}
		if err := validateMediaSourceRefs(prefix+".MediaSources", definition.MediaSources); err != nil {
			return err
		}

		fieldNames := make(map[string]bool, len(definition.Fields))
		for j := range definition.Fields {
			segmentField := &definition.Fields[j]
			fieldPrefix := fmt.Sprintf("%s.Fields[%d]", prefix, j)
			if err := validateSegmentField(fieldPrefix, segmentField, options); err != nil {
				return err
			}
			if fieldNames[segmentField.Name] {
				return validationError(fieldPrefix, "name %q is duplicated", segmentField.Name)
			}
			fieldNames[segmentField.Name] = true
		}
		if err := validateTimeRanges(prefix+".TimeRanges", definition.TimeRanges); err != nil {
			return err
		}
	}
	return nil
}

func validateSegmentField(field string, segmentField *SegmentField, options responseFormatOptions) error {
	if segmentField.Name == "" {
		return validationError(field, "Name is required")
	}
	if len(segmentField.Name) > 50 {
		return validationError(field, "Name must be at most 50 characters")
	}
	if segmentField.Description == "" {
		return validationError(field, "Description is required")
	}
	if segmentField.Type == "" {
		return validationError(field, "Type is required")
	}
	if segmentField.Type == SegmentFieldTimestamp {
		if segmentField.Format == "" {
			return validationError(field, "Format is required for timestamp fields")
		}
		if !validTimeFormat(segmentField.Format) {
			return validationError(field, "Format must be %q, %q, or %q", TimeFormatSeconds, TimeFormatHHMMSS, TimeFormatHHMMSSMillis)
		}
		if segmentField.Items != nil {
			return validationError(field, "Items is not supported for timestamp fields")
		}
		if len(segmentField.Enum) > 0 {
			return validationError(field, "Enum is not supported for timestamp fields")
		}
	} else if segmentField.Format != "" {
		return validationError(field, "Format is only valid for timestamp fields")
	}

	switch segmentField.Type {
	case SegmentFieldString, SegmentFieldBoolean, SegmentFieldNumber, SegmentFieldInteger, SegmentFieldTimestamp:
		if segmentField.Items != nil {
			return validationError(field, "Items is only valid for array and time_array fields")
		}
	case SegmentFieldArray:
		if segmentField.Items == nil {
			return validationError(field, "Items is required for array fields")
		}
		switch segmentField.Items.Type {
		case SegmentFieldString, SegmentFieldBoolean, SegmentFieldNumber, SegmentFieldInteger:
		default:
			return validationError(field, "Items.Type must be a scalar type for array fields")
		}
		if len(segmentField.Items.Fields) > 0 {
			return validationError(field, "Items.Fields is only valid for time_array fields")
		}
	case SegmentFieldTimeArray:
		if !options.allowTimeArray || options.modelName != ModelPegasus16 {
			return validationError(field, "time_array fields require model %q", ModelPegasus16)
		}
		if segmentField.Items == nil || segmentField.Items.Type != "object" {
			return validationError(field, `time_array fields require Items.Type to be "object"`)
		}
		if len(segmentField.Items.Fields) == 0 {
			return validationError(field, "time_array fields require at least one field in Items.Fields")
		}
		if len(segmentField.Enum) > 0 {
			return validationError(field, "Enum is not supported for time_array fields")
		}
		nestedNames := make(map[string]bool, len(segmentField.Items.Fields))
		for k := range segmentField.Items.Fields {
			nested := &segmentField.Items.Fields[k]
			nestedPrefix := fmt.Sprintf("%s.Items.Fields[%d]", field, k)
			if err := validateTimeArrayItemField(nestedPrefix, nested); err != nil {
				return err
			}
			if nestedNames[nested.Name] {
				return validationError(nestedPrefix, "name %q is duplicated", nested.Name)
			}
			nestedNames[nested.Name] = true
		}
	default:
		return validationError(field, "Type must be one of string, boolean, number, integer, array, timestamp, or time_array")
	}

	if len(segmentField.Enum) > 100 {
		return validationError(field, "Enum supports at most 100 values")
	}
	for _, value := range segmentField.Enum {
		if value == "" || len(value) > 100 {
			return validationError(field, "Enum values must be non-empty and at most 100 characters")
		}
	}
	return nil
}

func validateTimeArrayItemField(field string, nested *SegmentField) error {
	switch nested.Name {
	case "":
		return validationError(field, "Name is required")
	case "start_time", "end_time", "metadata":
		return validationError(field, "Name %q is reserved", nested.Name)
	}
	if len(nested.Name) > 50 {
		return validationError(field, "Name must be at most 50 characters")
	}
	if nested.Description == "" {
		return validationError(field, "Description is required")
	}
	if nested.Format != "" {
		return validationError(field, "Format is not supported inside time_array fields")
	}
	switch nested.Type {
	case SegmentFieldString, SegmentFieldBoolean, SegmentFieldNumber, SegmentFieldInteger:
		if nested.Items != nil {
			return validationError(field, "Items is only valid for array fields")
		}
	case SegmentFieldArray:
		if nested.Items == nil {
			return validationError(field, "Items is required for array fields")
		}
		switch nested.Items.Type {
		case SegmentFieldString, SegmentFieldBoolean, SegmentFieldNumber, SegmentFieldInteger:
		default:
			return validationError(field, "Items.Type must be a scalar type")
		}
		if len(nested.Items.Fields) > 0 {
			return validationError(field, "Items.Fields is not supported inside time_array fields")
		}
	default:
		return validationError(field, "Type must be string, boolean, number, integer, or array")
	}
	if len(nested.Enum) > 0 {
		if nested.Type != SegmentFieldString {
			return validationError(field, "Enum is only supported for string fields")
		}
		if len(nested.Enum) > 100 {
			return validationError(field, "Enum supports at most 100 values")
		}
		for _, value := range nested.Enum {
			if value == "" || len(value) > 100 {
				return validationError(field, "Enum values must be non-empty and at most 100 characters")
			}
		}
	}
	return nil
}

func validateMediaSourceRefs(field string, sources []MediaSourceRef) error {
	names := make(map[string]bool, len(sources))
	for i := range sources {
		source := &sources[i]
		prefix := fmt.Sprintf("%s[%d]", field, i)
		if source.Name == "" {
			return validationError(prefix, "Name is required")
		}
		if len(source.Name) > 50 {
			return validationError(prefix, "Name must be at most 50 characters")
		}
		if names[source.Name] {
			return validationError(prefix, "name %q is duplicated", source.Name)
		}
		names[source.Name] = true
		if source.MediaType == "" {
			source.MediaType = "image"
		} else if source.MediaType != "image" {
			return validationError(prefix, `MediaType must be "image"`)
		}
		if err := validateImageSource(field, i, source.URL, source.AssetID, source.Base64String); err != nil {
			return err
		}
	}
	return nil
}

func validateTimeRanges(field string, ranges []TimeRange) error {
	for i, span := range ranges {
		prefix := fmt.Sprintf("%s[%d]", field, i)
		if math.IsNaN(span.StartTime) || math.IsInf(span.StartTime, 0) || math.IsNaN(span.EndTime) || math.IsInf(span.EndTime, 0) {
			return validationError(prefix, "times must be finite numbers")
		}
		if span.EndTime <= span.StartTime {
			return validationError(prefix, "EndTime must be greater than StartTime")
		}
		if span.EndTime-span.StartTime < 2 {
			return validationError(prefix, "must be at least 2 seconds long")
		}
	}
	for i := 0; i < len(ranges); i++ {
		for j := i + 1; j < len(ranges); j++ {
			if ranges[i].StartTime < ranges[j].EndTime && ranges[j].StartTime < ranges[i].EndTime {
				return validationError(fmt.Sprintf("%s[%d]", field, j), "time ranges must not overlap")
			}
		}
	}
	return nil
}

func validTimeFormat(format string) bool {
	switch format {
	case TimeFormatSeconds, TimeFormatHHMMSS, TimeFormatHHMMSSMillis:
		return true
	default:
		return false
	}
}

// AnalyzeRequest is a synchronous analysis request. The response is returned
// in one piece by Analyze or streamed incrementally by Stream.
type AnalyzeRequest struct {
	// ModelName defaults to ModelPegasus15. Image inputs require ModelPegasus16.
	ModelName string `json:"model_name,omitempty"`
	// Video and Image are mutually exclusive as the primary media. Exactly one
	// of them is required. Image may contain at most 20 entries and requires a
	// Prompt.
	Video *VideoContext  `json:"video,omitempty"`
	Image []AnalyzeImage `json:"image,omitempty"`
	// Prompt is a free-form prompt. Mutually exclusive with PromptV2. It is
	// required for image analysis.
	Prompt string `json:"prompt,omitempty"`
	// PromptV2 is a prompt that references named images. Mutually exclusive
	// with Prompt and Image.
	PromptV2 *AnalyzePromptV2 `json:"prompt_v2,omitempty"`
	// Temperature controls randomness between 0 and 1. The API defaults to 0.2.
	Temperature *float64 `json:"temperature,omitempty"`
	// ResponseFormat requests JSON schema output. Synchronous analysis does not
	// support segment definitions.
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	// MaxTokens bounds the response. It must be between 512 and 98,304 when
	// set; the API defaults to 4,096.
	MaxTokens int `json:"max_tokens,omitempty"`
	// StartTime and EndTime narrow the analysis to a time window, in seconds.
	// When both are set, EndTime must be greater by at least one second.
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

// Analyze runs a synchronous analysis and returns the generated text.
//
// Use Stream when the prompt produces long output and you want it as it is
// generated.
//
// When the model stops because it reached the response or context limit,
// FinishReason is "length" and Error describes the truncation. That is a
// warning, not a failure: the partial text is still returned and the method
// returns a nil error.
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
	return &response, nil
}

// Stream runs a synchronous analysis and calls handler for every event as it
// arrives. Events are NDJSON, not server-sent events.
//
// The handler is called for stream_start, every text_generation chunk, and
// stream_end. If generation was truncated, the stream_end event carries
// FinishReason "length" and an Error describing the truncation; that is a
// warning, not a failure, so Stream returns nil after the handler runs.
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
		accept:   "application/x-ndjson, application/json",
	})
	if err != nil {
		return err
	}
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	for {
		var event AnalyzeStreamEvent
		if err := decoder.Decode(&event); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("twelvelabs: decode stream event: %w", err)
		}
		if handler != nil {
			if err := handler(event); err != nil {
				return err
			}
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
	switch modelName {
	case ModelPegasus15, ModelPegasus16:
	default:
		return nil, validationError("ModelName", "must be %q or %q", ModelPegasus15, ModelPegasus16)
	}
	request.ModelName = modelName

	if request.Video == nil && len(request.Image) == 0 {
		return nil, validationError("Video", "a video or at least one image is required")
	}
	if request.Video != nil && len(request.Image) > 0 {
		return nil, validationError("Image", "Video and Image are mutually exclusive")
	}
	if request.Video != nil {
		if err := request.Video.validate("Video"); err != nil {
			return nil, err
		}
	}
	if request.Prompt != "" && request.PromptV2 != nil {
		return nil, validationError("PromptV2", "set either Prompt or PromptV2, not both")
	}
	if err := validateImages("Image", modelName, request.Image, request.Prompt, request.PromptV2 != nil); err != nil {
		return nil, err
	}
	if request.PromptV2 != nil {
		if len(request.Image) > 0 {
			return nil, validationError("PromptV2", "PromptV2 and Image are mutually exclusive")
		}
		if err := validatePromptV2("PromptV2", request.PromptV2); err != nil {
			return nil, err
		}
	}
	if err := request.ResponseFormat.validate("ResponseFormat", responseFormatOptions{
		modelName:               modelName,
		allowSegmentDefinitions: false,
		allowTimeArray:          false,
	}); err != nil {
		return nil, err
	}
	if err := validateTemperature(request.Temperature); err != nil {
		return nil, err
	}
	if request.MaxTokens < 0 {
		return nil, validationError("MaxTokens", "must be positive when set")
	}
	if request.MaxTokens > 0 && (request.MaxTokens < 512 || request.MaxTokens > 98304) {
		return nil, validationError("MaxTokens", "must be between 512 and 98304")
	}
	if err := validateTimeWindow(request.StartTime, request.EndTime, 1); err != nil {
		return nil, err
	}

	return &analyzeWire{AnalyzeRequest: request, Stream: stream}, nil
}

func validateImages(field, modelName string, images []AnalyzeImage, prompt string, hasPromptV2 bool) error {
	if len(images) == 0 {
		return nil
	}
	if len(images) > 20 {
		return validationError(field, "at most 20 images are allowed")
	}
	if modelName != ModelPegasus16 {
		return validationError(field, "image analysis requires model %q", ModelPegasus16)
	}
	if prompt == "" {
		return validationError("Prompt", "is required when analyzing images")
	}
	if hasPromptV2 {
		return validationError(field, "Image and PromptV2 are mutually exclusive")
	}
	names := make(map[string]bool, len(images))
	for i := range images {
		image := &images[i]
		if image.Name == "" {
			return validationError(field, "[%d].Name is required", i)
		}
		if !imageNamePattern.MatchString(image.Name) {
			return validationError(field, "[%d].Name must be 1-50 letters, digits, or underscores", i)
		}
		if names[image.Name] {
			return validationError(field, "[%d].Name %q is duplicated", i, image.Name)
		}
		names[image.Name] = true
		if err := validateImageSource(field, i, image.URL, image.AssetID, image.Base64String); err != nil {
			return err
		}
	}
	return nil
}

func validatePromptV2(field string, prompt *AnalyzePromptV2) error {
	if prompt.InputText == "" {
		return validationError(field, "InputText is required")
	}
	if len(prompt.MediaSources) > 4 {
		return validationError(field, "at most 4 media sources are allowed")
	}
	return validateMediaSourceRefs(field+".MediaSources", prompt.MediaSources)
}

func validateTemperature(temperature *float64) error {
	if temperature == nil {
		return nil
	}
	if math.IsNaN(*temperature) || math.IsInf(*temperature, 0) || *temperature < 0 || *temperature > 1 {
		return validationError("Temperature", "must be between 0 and 1")
	}
	return nil
}

func validateTimeWindow(start, end *float64, minimumSeconds float64) error {
	for name, value := range map[string]*float64{"StartTime": start, "EndTime": end} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0)) {
			return validationError(name, "must be a finite number")
		}
	}
	if start != nil && end != nil {
		if *end <= *start {
			return validationError("EndTime", "must be greater than StartTime")
		}
		if *end-*start < minimumSeconds {
			return validationError("EndTime", "the analysis window must be at least %g second(s) long", minimumSeconds)
		}
	}
	return nil
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
	// MaxTokens bounds the response. Zero uses the platform default. When set,
	// it must be between 512 and 98,304 in general mode and between 2,048 and
	// 98,304 in time-based mode.
	MaxTokens int `json:"max_tokens,omitempty"`
	// UnlimitedTokens removes the token limit. It requires
	// AnalysisModeTimeBasedMetadata and ModelPegasus16, cannot be combined with
	// MaxTokens, and cannot be combined with time ranges in the segment
	// definitions.
	UnlimitedTokens bool `json:"-"`
	// ResponseFormat requests structured output or segment extraction. General
	// analysis accepts json_schema; time-based analysis requires
	// segment_definitions.
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
	// MinSegmentDuration and MaxSegmentDuration constrain extracted segments,
	// in seconds. Both must be at least 2, MaxSegmentDuration must be greater
	// than or equal to MinSegmentDuration, and both require
	// AnalysisModeTimeBasedMetadata. They are mutually exclusive with time
	// ranges and with StartTime/EndTime.
	MinSegmentDuration float64 `json:"min_segment_duration,omitempty"`
	MaxSegmentDuration float64 `json:"max_segment_duration,omitempty"`
	// StartTime and EndTime narrow the analysis to a time window, in seconds.
	// They are mutually exclusive with segment time ranges.
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
	AnalysisMode   string           `json:"analysis_mode,omitempty"`
	Prompt         *string          `json:"prompt,omitempty"`
	PromptV2       *AnalyzePromptV2 `json:"prompt_v2,omitempty"`
	Image          []AnalyzeImage   `json:"image,omitempty"`
	ResponseFormat *ResponseFormat  `json:"response_format,omitempty"`
	Temperature    *float64         `json:"temperature,omitempty"`
	// MaxTokens is a number, or Unlimited when the task ran without a token
	// limit.
	MaxTokens          TokenLimit `json:"max_tokens,omitempty"`
	MinSegmentDuration float64    `json:"min_segment_duration,omitempty"`
	MaxSegmentDuration float64    `json:"max_segment_duration,omitempty"`
	StartTime          *float64   `json:"start_time,omitempty"`
	EndTime            *float64   `json:"end_time,omitempty"`
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
	// Status filters by one task status.
	Status       string
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
		if options.Status != "" {
			switch options.Status {
			case AnalyzeTaskStatusQueued, AnalyzeTaskStatusPending, AnalyzeTaskStatusProcessing, AnalyzeTaskStatusReady, AnalyzeTaskStatusFailed, AnalyzeTaskStatusCanceled:
				query.Set("status", options.Status)
			default:
				return nil, validationError("Status", "must be a valid analysis task status")
			}
		}
		setIfNotEmpty(query, "video_url", options.VideoURL)
		setIfNotEmpty(query, "asset_id", options.AssetID)
		if options.AnalysisMode != "" && options.AnalysisMode != AnalysisModeGeneral && options.AnalysisMode != AnalysisModeTimeBasedMetadata {
			return nil, validationError("AnalysisMode", "must be %q or %q", AnalysisModeGeneral, AnalysisModeTimeBasedMetadata)
		}
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
	switch request.ModelName {
	case ModelPegasus15, ModelPegasus16:
	default:
		return validationError("ModelName", "must be %q or %q", ModelPegasus15, ModelPegasus16)
	}
	if request.CustomID != "" && !customIDPattern.MatchString(request.CustomID) {
		return validationError("CustomID", "must match %s", customIDPattern)
	}
	if request.AnalysisMode == "" {
		request.AnalysisMode = AnalysisModeGeneral
	}
	switch request.AnalysisMode {
	case AnalysisModeGeneral, AnalysisModeTimeBasedMetadata:
	default:
		return validationError("AnalysisMode", "must be %q or %q", AnalysisModeGeneral, AnalysisModeTimeBasedMetadata)
	}

	if request.Video == nil && len(request.Image) == 0 {
		return validationError("Video", "a video or at least one image is required")
	}
	if request.Video != nil && len(request.Image) > 0 {
		return validationError("Image", "Video and Image are mutually exclusive")
	}
	if request.Video != nil {
		if err := request.Video.validate("Video"); err != nil {
			return err
		}
	}
	if request.Prompt != "" && request.PromptV2 != nil {
		return validationError("PromptV2", "set either Prompt or PromptV2, not both")
	}
	if len(request.Image) > 0 {
		if request.AnalysisMode != AnalysisModeGeneral {
			return validationError("Image", "images are only supported with AnalysisMode %q", AnalysisModeGeneral)
		}
		if err := validateImages("Image", request.ModelName, request.Image, request.Prompt, request.PromptV2 != nil); err != nil {
			return err
		}
	}
	if request.PromptV2 != nil {
		if len(request.Image) > 0 {
			return validationError("PromptV2", "PromptV2 and Image are mutually exclusive")
		}
		if err := validatePromptV2("PromptV2", request.PromptV2); err != nil {
			return err
		}
	}

	switch request.AnalysisMode {
	case AnalysisModeGeneral:
		if request.Prompt == "" && request.PromptV2 == nil {
			return validationError("Prompt", "general analysis requires Prompt or PromptV2")
		}
	case AnalysisModeTimeBasedMetadata:
		if request.Prompt != "" || request.PromptV2 != nil {
			return validationError("Prompt", "prompts are not supported with AnalysisMode %q", AnalysisModeTimeBasedMetadata)
		}
		if request.ResponseFormat == nil || request.ResponseFormat.Type != ResponseFormatSegmentDefinitions {
			return validationError("ResponseFormat", "analysis mode %q requires segment definitions", AnalysisModeTimeBasedMetadata)
		}
	}

	if err := request.ResponseFormat.validate("ResponseFormat", responseFormatOptions{
		modelName:               request.ModelName,
		allowSegmentDefinitions: request.AnalysisMode == AnalysisModeTimeBasedMetadata,
		allowTimeArray:          true,
	}); err != nil {
		return err
	}
	if err := validateTemperature(request.Temperature); err != nil {
		return err
	}
	if request.MaxTokens < 0 {
		return validationError("MaxTokens", "must be positive when set")
	}

	hasTimeRanges := false
	if request.ResponseFormat != nil {
		for _, definition := range request.ResponseFormat.SegmentDefinitions {
			if len(definition.TimeRanges) > 0 {
				hasTimeRanges = true
				break
			}
		}
	}

	if request.UnlimitedTokens {
		if request.AnalysisMode != AnalysisModeTimeBasedMetadata {
			return validationError("UnlimitedTokens", "requires AnalysisMode %q", AnalysisModeTimeBasedMetadata)
		}
		if request.ModelName != ModelPegasus16 {
			return validationError("UnlimitedTokens", "requires model %q", ModelPegasus16)
		}
		if request.MaxTokens != 0 {
			return validationError("MaxTokens", "set either MaxTokens or UnlimitedTokens, not both")
		}
		if hasTimeRanges {
			return validationError("UnlimitedTokens", "cannot be combined with segment time ranges")
		}
	} else if request.MaxTokens > 0 {
		minimum := 512
		if request.AnalysisMode == AnalysisModeTimeBasedMetadata {
			minimum = 2048
		}
		if request.MaxTokens < minimum || request.MaxTokens > 98304 {
			return validationError("MaxTokens", "must be between %d and 98304", minimum)
		}
	}

	if request.MinSegmentDuration < 0 || request.MaxSegmentDuration < 0 {
		return validationError("MinSegmentDuration", "segment durations must be positive when set")
	}
	if request.MinSegmentDuration > 0 || request.MaxSegmentDuration > 0 {
		if request.AnalysisMode != AnalysisModeTimeBasedMetadata {
			return validationError("MinSegmentDuration", "segment durations require AnalysisMode %q", AnalysisModeTimeBasedMetadata)
		}
		if request.MinSegmentDuration > 0 && request.MinSegmentDuration < 2 {
			return validationError("MinSegmentDuration", "must be at least 2 seconds")
		}
		if request.MaxSegmentDuration > 0 && request.MaxSegmentDuration < 2 {
			return validationError("MaxSegmentDuration", "must be at least 2 seconds")
		}
		if request.MinSegmentDuration > 0 && request.MaxSegmentDuration > 0 && request.MaxSegmentDuration < request.MinSegmentDuration {
			return validationError("MaxSegmentDuration", "must be greater than or equal to MinSegmentDuration")
		}
		if hasTimeRanges {
			return validationError("MinSegmentDuration", "segment durations cannot be combined with segment time ranges")
		}
		if request.StartTime != nil || request.EndTime != nil {
			return validationError("MinSegmentDuration", "segment durations cannot be combined with StartTime or EndTime")
		}
	}
	if hasTimeRanges && (request.StartTime != nil || request.EndTime != nil) {
		return validationError("TimeRanges", "segment time ranges cannot be combined with StartTime or EndTime")
	}
	if err := validateTimeWindow(request.StartTime, request.EndTime, 1); err != nil {
		return err
	}
	if request.StartTime != nil && request.EndTime != nil && *request.EndTime-*request.StartTime > 7200 {
		return validationError("EndTime", "the analysis window must be at most 2 hours")
	}
	return nil
}

func validateImageSource(field string, index int, rawURL, assetID, base64String string) error {
	provided := 0
	for _, value := range []string{rawURL, assetID, base64String} {
		if value != "" {
			provided++
		}
	}
	if provided != 1 {
		return validationError(field, "[%d] must set exactly one of URL, AssetID, or Base64String", index)
	}
	if rawURL != "" {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
			return validationError(field, "[%d].URL must be a publicly accessible HTTPS URL", index)
		}
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
