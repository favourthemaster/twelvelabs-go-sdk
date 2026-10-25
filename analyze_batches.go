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
	"time"
)

// Batch status values. Completed, canceled, and expired are final.
const (
	BatchStatusPending    = "pending"
	BatchStatusProcessing = "processing"
	BatchStatusCanceling  = "canceling"
	BatchStatusCanceled   = "canceled"
	BatchStatusCompleted  = "completed"
	BatchStatusExpired    = "expired"
)

// Batch item status values.
const (
	BatchItemStatusQueued     = "queued"
	BatchItemStatusProcessing = "processing"
	BatchItemStatusReady      = "ready"
	BatchItemStatusFailed     = "failed"
	BatchItemStatusCanceled   = "canceled"
)

// BatchVideoContext identifies the asset to analyze in a batch.
type BatchVideoContext struct {
	// Type must be "asset_id".
	Type string `json:"type"`
	// AssetID is the asset to analyze. It must be ready.
	AssetID string `json:"asset_id"`
}

// BatchDefaults holds per-batch settings that items inherit unless they
// override them.
type BatchDefaults struct {
	Prompt             *AnalyzePromptV2 `json:"prompt,omitempty"`
	ResponseFormat     *ResponseFormat  `json:"response_format,omitempty"`
	Temperature        *float64         `json:"temperature,omitempty"`
	MaxTokens          int              `json:"max_tokens,omitempty"`
	MinSegmentDuration float64          `json:"min_segment_duration,omitempty"`
	MaxSegmentDuration float64          `json:"max_segment_duration,omitempty"`
	StartTime          *float64         `json:"start_time,omitempty"`
	EndTime            *float64         `json:"end_time,omitempty"`
}

// BatchItem is one video to analyze within a batch.
type BatchItem struct {
	Video              *BatchVideoContext `json:"video"`
	CustomID           string             `json:"custom_id,omitempty"`
	Prompt             *AnalyzePromptV2   `json:"prompt,omitempty"`
	ResponseFormat     *ResponseFormat    `json:"response_format,omitempty"`
	Temperature        *float64           `json:"temperature,omitempty"`
	MaxTokens          int                `json:"max_tokens,omitempty"`
	MinSegmentDuration float64            `json:"min_segment_duration,omitempty"`
	MaxSegmentDuration float64            `json:"max_segment_duration,omitempty"`
	StartTime          *float64           `json:"start_time,omitempty"`
	EndTime            *float64           `json:"end_time,omitempty"`
}

// AnalyzeBatchCreateRequest creates a batch. Batches accept only ModelPegasus15
// and a single analysis mode.
type AnalyzeBatchCreateRequest struct {
	// ModelName must be ModelPegasus15.
	ModelName string `json:"model_name"`
	// AnalysisMode is AnalysisModeGeneral or AnalysisModeTimeBasedMetadata.
	AnalysisMode string         `json:"analysis_mode"`
	Defaults     *BatchDefaults `json:"defaults,omitempty"`
	// Requests must contain at least one item.
	Requests []BatchItem `json:"requests"`
}

// CreatedBatchItem maps a submitted item to the task the platform created.
type CreatedBatchItem struct {
	TaskID   string  `json:"task_id"`
	CustomID *string `json:"custom_id"`
}

// AnalyzeBatchCreateResponse is returned when a batch is created.
type AnalyzeBatchCreateResponse struct {
	BatchID    string             `json:"batch_id"`
	Status     string             `json:"status"`
	TotalItems int                `json:"total_items"`
	CreatedAt  string             `json:"created_at"`
	ExpiresAt  string             `json:"expires_at"`
	Items      []CreatedBatchItem `json:"items"`
}

// AnalyzeBatch is the current state of a batch.
type AnalyzeBatch struct {
	BatchID         string            `json:"batch_id"`
	AnalysisMode    string            `json:"analysis_mode"`
	ModelName       string            `json:"model_name"`
	Status          string            `json:"status"`
	TotalItems      int               `json:"total_items"`
	CreatedAt       string            `json:"created_at"`
	ExpiresAt       string            `json:"expires_at"`
	CompletedAt     string            `json:"completed_at,omitempty"`
	CanceledAt      string            `json:"canceled_at,omitempty"`
	ExpiredAt       string            `json:"expired_at,omitempty"`
	QueuedItems     int               `json:"queued_items"`
	ProcessingItems int               `json:"processing_items"`
	ReadyItems      int               `json:"ready_items"`
	FailedItems     int               `json:"failed_items"`
	CanceledItems   int               `json:"canceled_items"`
	Webhooks        []WebhookDelivery `json:"webhooks,omitempty"`
}

// BatchResultItem is one line of the JSON Lines results stream.
type BatchResultItem struct {
	TaskID   string             `json:"task_id"`
	CustomID *string            `json:"custom_id"`
	Status   string             `json:"status"`
	Data     *AnalyzeTaskResult `json:"data,omitempty"`
	Error    *AnalyzeError      `json:"error,omitempty"`
}

// BatchListOptions filters ListBatches.
type BatchListOptions struct {
	Pagination
	Status       []string
	AnalysisMode []string
}

// CreateBatch queues the same prompt across many assets.
func (s *AnalyzeService) CreateBatch(ctx context.Context, request *AnalyzeBatchCreateRequest) (*AnalyzeBatchCreateResponse, error) {
	ctx = ensureContext(ctx)
	if request == nil {
		return nil, validationError("request", "is required")
	}
	if request.ModelName == "" {
		request.ModelName = ModelPegasus15
	}
	if request.ModelName != ModelPegasus15 {
		return nil, validationError("ModelName", "batches only accept %q", ModelPegasus15)
	}
	if request.AnalysisMode == "" {
		request.AnalysisMode = AnalysisModeGeneral
	}
	switch request.AnalysisMode {
	case AnalysisModeGeneral, AnalysisModeTimeBasedMetadata:
	default:
		return nil, validationError("AnalysisMode", "must be %q or %q", AnalysisModeGeneral, AnalysisModeTimeBasedMetadata)
	}
	if len(request.Requests) == 0 {
		return nil, validationError("Requests", "at least one item is required")
	}
	if len(request.Requests) > 1000 {
		return nil, validationError("Requests", "at most 1000 items are allowed")
	}
	if err := request.Defaults.validateDefaults(request.AnalysisMode); err != nil {
		return nil, err
	}
	customIDs := make(map[string]bool, len(request.Requests))
	for i := range request.Requests {
		item := &request.Requests[i]
		if item.Video == nil || item.Video.AssetID == "" {
			return nil, validationError("Requests", "item %d requires Video.AssetID", i)
		}
		if item.Video.Type == "" {
			item.Video.Type = "asset_id"
		}
		if item.Video.Type != "asset_id" {
			return nil, validationError("Requests", `item %d Video.Type must be "asset_id"`, i)
		}
		if item.CustomID != "" {
			if !customIDPattern.MatchString(item.CustomID) {
				return nil, validationError("Requests", "item %d CustomID must match %s", i, customIDPattern)
			}
			if customIDs[item.CustomID] {
				return nil, validationError("Requests", "item %d CustomID %q is duplicated", i, item.CustomID)
			}
			customIDs[item.CustomID] = true
		}
		if err := validateBatchPrompt(fmt.Sprintf("Requests[%d].Prompt", i), item.Prompt, request.AnalysisMode); err != nil {
			return nil, err
		}
		if err := validateBatchResponseFormat(fmt.Sprintf("Requests[%d].ResponseFormat", i), item.ResponseFormat, request.AnalysisMode); err != nil {
			return nil, err
		}
		if err := validateBatchItem(request.Defaults, item, i, request.AnalysisMode); err != nil {
			return nil, err
		}
	}

	var response AnalyzeBatchCreateResponse
	if err := s.client.transport.doJSON(ctx, http.MethodPost, "/analyze/batches", nil, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (d *BatchDefaults) validateDefaults(mode string) error {
	if d == nil {
		return nil
	}
	if err := validateBatchPrompt("Defaults.Prompt", d.Prompt, mode); err != nil {
		return err
	}
	if err := validateBatchResponseFormat("Defaults.ResponseFormat", d.ResponseFormat, mode); err != nil {
		return err
	}
	if err := validateBatchTemperature("Defaults.Temperature", d.Temperature); err != nil {
		return err
	}
	if err := validateBatchMaxTokens("Defaults.MaxTokens", d.MaxTokens, mode); err != nil {
		return err
	}
	if err := validateSegmentDurationOptions("Defaults", d.MinSegmentDuration, d.MaxSegmentDuration, mode); err != nil {
		return err
	}
	if err := validateBatchTimeWindow("Defaults", d.StartTime, d.EndTime); err != nil {
		return err
	}
	if responseFormatHasTimeRanges(d.ResponseFormat) && (d.StartTime != nil || d.EndTime != nil) {
		return validationError("Defaults", "segment time ranges cannot be combined with StartTime or EndTime")
	}
	if responseFormatHasTimeRanges(d.ResponseFormat) && (d.MinSegmentDuration > 0 || d.MaxSegmentDuration > 0) {
		return validationError("Defaults", "segment time ranges cannot be combined with segment durations")
	}
	return nil
}

func validateBatchPrompt(field string, prompt *AnalyzePromptV2, mode string) error {
	if prompt == nil {
		return nil
	}
	if mode == AnalysisModeTimeBasedMetadata {
		return validationError(field, "prompts are not supported with AnalysisMode %q", mode)
	}
	return validatePromptV2(field, prompt)
}

func validateBatchResponseFormat(field string, format *ResponseFormat, mode string) error {
	if format == nil {
		return nil
	}
	if err := format.validate(field, responseFormatOptions{
		modelName:               ModelPegasus15,
		allowSegmentDefinitions: mode == AnalysisModeTimeBasedMetadata,
		allowTimeArray:          false,
	}); err != nil {
		return err
	}
	if mode == AnalysisModeTimeBasedMetadata && format.Type != ResponseFormatSegmentDefinitions {
		return validationError(field, "analysis mode %q requires segment definitions", mode)
	}
	if len(format.SegmentDefinitions) > 20 {
		return validationError(field, "at most 20 segment definitions are allowed")
	}
	return nil
}

func validateBatchTemperature(field string, temperature *float64) error {
	if temperature == nil {
		return nil
	}
	if math.IsNaN(*temperature) || math.IsInf(*temperature, 0) || *temperature < 0 || *temperature > 1 {
		return validationError(field, "must be between 0 and 1")
	}
	return nil
}

func validateBatchMaxTokens(field string, maxTokens int, mode string) error {
	if maxTokens < 0 {
		return validationError(field, "must be positive when set")
	}
	if maxTokens == 0 {
		return nil
	}
	minimum := 512
	if mode == AnalysisModeTimeBasedMetadata {
		minimum = 2048
	}
	if maxTokens < minimum || maxTokens > 98304 {
		return validationError(field, "must be between %d and 98304", minimum)
	}
	return nil
}

func validateSegmentDurationOptions(field string, minDuration, maxDuration float64, mode string) error {
	if minDuration < 0 || maxDuration < 0 {
		return validationError(field, "segment durations must be positive when set")
	}
	if minDuration == 0 && maxDuration == 0 {
		return nil
	}
	if mode != AnalysisModeTimeBasedMetadata {
		return validationError(field, "segment durations require AnalysisMode %q", AnalysisModeTimeBasedMetadata)
	}
	if minDuration > 0 && minDuration < 2 {
		return validationError(field, "MinSegmentDuration must be at least 2 seconds")
	}
	if maxDuration > 0 && maxDuration < 2 {
		return validationError(field, "MaxSegmentDuration must be at least 2 seconds")
	}
	if minDuration > 0 && maxDuration > 0 && maxDuration < minDuration {
		return validationError(field, "MaxSegmentDuration must be greater than or equal to MinSegmentDuration")
	}
	return nil
}

func validateBatchTimeWindow(field string, start, end *float64) error {
	if err := validateTimeWindow(start, end, 1); err != nil {
		return err
	}
	if start != nil && end != nil && *end-*start > 7200 {
		return validationError(field+".EndTime", "the analysis window must be at most 2 hours")
	}
	return nil
}

func validateBatchItem(defaults *BatchDefaults, item *BatchItem, index int, mode string) error {
	prefix := fmt.Sprintf("Requests[%d]", index)

	prompt := item.Prompt
	if prompt == nil && defaults != nil {
		prompt = defaults.Prompt
	}
	if mode == AnalysisModeGeneral && prompt == nil {
		return validationError(prefix, "general analysis requires Prompt or Defaults.Prompt")
	}

	format := item.ResponseFormat
	if format == nil && defaults != nil {
		format = defaults.ResponseFormat
	}
	if mode == AnalysisModeTimeBasedMetadata {
		if format == nil || format.Type != ResponseFormatSegmentDefinitions {
			return validationError(prefix, "analysis mode %q requires segment definitions", mode)
		}
	}

	var defaultTemperature *float64
	var defaultMaxTokens int
	var defaultMinDuration, defaultMaxDuration float64
	var defaultStart, defaultEnd *float64
	if defaults != nil {
		defaultTemperature = defaults.Temperature
		defaultMaxTokens = defaults.MaxTokens
		defaultMinDuration = defaults.MinSegmentDuration
		defaultMaxDuration = defaults.MaxSegmentDuration
		defaultStart = defaults.StartTime
		defaultEnd = defaults.EndTime
	}

	temperature := item.Temperature
	if temperature == nil {
		temperature = defaultTemperature
	}
	if err := validateBatchTemperature(prefix+".Temperature", temperature); err != nil {
		return err
	}

	maxTokens := item.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}
	if err := validateBatchMaxTokens(prefix+".MaxTokens", maxTokens, mode); err != nil {
		return err
	}

	minDuration := item.MinSegmentDuration
	if minDuration == 0 {
		minDuration = defaultMinDuration
	}
	maxDuration := item.MaxSegmentDuration
	if maxDuration == 0 {
		maxDuration = defaultMaxDuration
	}
	if err := validateSegmentDurationOptions(prefix, minDuration, maxDuration, mode); err != nil {
		return err
	}

	start := item.StartTime
	if start == nil {
		start = defaultStart
	}
	end := item.EndTime
	if end == nil {
		end = defaultEnd
	}
	if err := validateBatchTimeWindow(prefix, start, end); err != nil {
		return err
	}

	hasTimeRanges := responseFormatHasTimeRanges(format)
	if hasTimeRanges && (start != nil || end != nil) {
		return validationError(prefix, "segment time ranges cannot be combined with StartTime or EndTime")
	}
	if hasTimeRanges && (minDuration > 0 || maxDuration > 0) {
		return validationError(prefix, "segment time ranges cannot be combined with segment durations")
	}
	return nil
}

func responseFormatHasTimeRanges(format *ResponseFormat) bool {
	if format == nil || format.Type != ResponseFormatSegmentDefinitions {
		return false
	}
	for _, definition := range format.SegmentDefinitions {
		if len(definition.TimeRanges) > 0 {
			return true
		}
	}
	return false
}

// ListBatches returns batch jobs, most recent first.
func (s *AnalyzeService) ListBatches(ctx context.Context, options *BatchListOptions) (*Page[AnalyzeBatch], error) {
	ctx = ensureContext(ctx)
	query := url.Values{}
	if options != nil {
		query = encodeQuery(&options.Pagination)
		allowedStatuses := map[string]bool{
			BatchStatusPending:    true,
			BatchStatusProcessing: true,
			BatchStatusCanceling:  true,
			BatchStatusCanceled:   true,
			BatchStatusCompleted:  true,
			BatchStatusExpired:    true,
		}
		for i, status := range options.Status {
			if !allowedStatuses[status] {
				return nil, validationError("Status", "[%d] must be a valid batch status", i)
			}
			query.Add("status", status)
		}
		for i, mode := range options.AnalysisMode {
			if mode != AnalysisModeGeneral && mode != AnalysisModeTimeBasedMetadata {
				return nil, validationError("AnalysisMode", "[%d] must be %q or %q", i, AnalysisModeGeneral, AnalysisModeTimeBasedMetadata)
			}
			query.Add("analysis_mode", mode)
		}
	}

	var page Page[AnalyzeBatch]
	if err := s.client.transport.doJSON(ctx, http.MethodGet, "/analyze/batches", query, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// GetBatch retrieves a batch.
func (s *AnalyzeService) GetBatch(ctx context.Context, batchID string) (*AnalyzeBatch, error) {
	ctx = ensureContext(ctx)
	if batchID == "" {
		return nil, validationError("batchID", "is required")
	}
	var batch AnalyzeBatch
	path := "/analyze/batches/" + escapePath(batchID)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, nil, nil, &batch); err != nil {
		return nil, err
	}
	return &batch, nil
}

// CancelBatch cancels a batch and returns its new state. Items that already
// completed keep their results.
func (s *AnalyzeService) CancelBatch(ctx context.Context, batchID string) (*AnalyzeBatch, error) {
	ctx = ensureContext(ctx)
	if batchID == "" {
		return nil, validationError("batchID", "is required")
	}
	var batch AnalyzeBatch
	path := "/analyze/batches/" + escapePath(batchID) + "/cancel"
	if err := s.client.transport.doJSON(ctx, http.MethodPost, path, nil, nil, &batch); err != nil {
		return nil, err
	}
	return &batch, nil
}

// DeleteBatch removes a batch. Results of its tasks are deleted with it.
func (s *AnalyzeService) DeleteBatch(ctx context.Context, batchID string) error {
	ctx = ensureContext(ctx)
	if batchID == "" {
		return validationError("batchID", "is required")
	}
	return s.client.transport.doJSON(ctx, http.MethodDelete, "/analyze/batches/"+escapePath(batchID), nil, nil, nil)
}

// BatchResults returns every result entry of a batch. The platform streams the
// results as JSON Lines; this method collects them into a slice.
//
// Use StreamBatchResults for batches large enough to be worth streaming.
func (s *AnalyzeService) BatchResults(ctx context.Context, batchID string) ([]BatchResultItem, error) {
	var results []BatchResultItem
	err := s.StreamBatchResults(ctx, batchID, func(item BatchResultItem) error {
		results = append(results, item)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// StreamBatchResults calls handler for each JSON Lines result entry as it
// arrives. Entries include queued and processing items, not only finished ones.
func (s *AnalyzeService) StreamBatchResults(ctx context.Context, batchID string, handler func(BatchResultItem) error) error {
	ctx = ensureContext(ctx)
	if batchID == "" {
		return validationError("batchID", "is required")
	}

	reader, err := s.client.transport.stream(ctx, &request{
		method: http.MethodGet,
		path:   "/analyze/batches/" + escapePath(batchID) + "/results",
		accept: "application/jsonl, application/json",
	})
	if err != nil {
		return err
	}
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	for {
		var item BatchResultItem
		if err := decoder.Decode(&item); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("twelvelabs: decode batch result: %w", err)
		}
		if handler != nil {
			if err := handler(item); err != nil {
				return err
			}
		}
	}
}

// WaitForBatch polls a batch until it completes, expires, or is canceled.
func (s *AnalyzeService) WaitForBatch(ctx context.Context, batchID string, options *WaitOptions) (*AnalyzeBatch, error) {
	ctx = ensureContext(ctx)
	if batchID == "" {
		return nil, validationError("batchID", "is required")
	}

	var deadline time.Time
	if options != nil && options.Timeout > 0 {
		deadline = time.Now().Add(options.Timeout)
	}

	for {
		batch, err := s.GetBatch(ctx, batchID)
		if err != nil {
			return nil, err
		}

		switch batch.Status {
		case BatchStatusCompleted:
			return batch, nil
		case BatchStatusCanceled:
			return batch, fmt.Errorf("twelvelabs: batch %s was canceled", batchID)
		case BatchStatusExpired:
			return batch, fmt.Errorf("twelvelabs: batch %s expired", batchID)
		}

		if options != nil && options.OnPoll != nil {
			if err := options.OnPoll(batch.Status); err != nil {
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
