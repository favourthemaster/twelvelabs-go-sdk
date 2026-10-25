package twelvelabs

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Webhook event types sent by the platform.
const (
	WebhookEventIndexTaskReady      = "index.task.ready"
	WebhookEventIndexTaskFailed     = "index.task.failed"
	WebhookEventAnalyzeTaskReady    = "analyze.task.ready"
	WebhookEventAnalyzeTaskFailed   = "analyze.task.failed"
	WebhookEventAnalyzeTaskCanceled = "analyze.task.canceled"
)

// DefaultWebhookTolerance is how old a webhook signature may be before it is
// rejected. It bounds replay attacks.
const DefaultWebhookTolerance = 5 * time.Minute

// WebhookEvent is a webhook notification envelope. Data is left raw because its
// shape depends on Type; use TaskData to decode it.
type WebhookEvent struct {
	ID        string          `json:"id"`
	CreatedAt string          `json:"created_at"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
}

// WebhookTaskData is the Data payload of the index.task.* and analyze.task.*
// events. It is a superset of both shapes: index events populate Metadata,
// Models, and Tags, while analyze events populate CustomID, CreatedAt,
// VideoSource, and Error.
type WebhookTaskData struct {
	ID          string                      `json:"id"`
	CustomID    *string                     `json:"custom_id"`
	Status      string                      `json:"status,omitempty"`
	CreatedAt   string                      `json:"created_at,omitempty"`
	VideoSource *WebhookVideoSource         `json:"video_source,omitempty"`
	Error       *ErrorBody                  `json:"error,omitempty"`
	Metadata    *IndexedAssetSystemMetadata `json:"metadata,omitempty"`
	Models      []WebhookModel              `json:"models,omitempty"`
	Tags        []string                    `json:"tags,omitempty"`
}

// WebhookModel is a model configuration reported in index.task.* events.
type WebhookModel struct {
	Name    string   `json:"name"`
	Options []string `json:"options,omitempty"`
}

// WebhookVideoSource identifies the media analyzed by an asynchronous analysis
// task. Type is "asset_id" or "url"; the matching field is set.
type WebhookVideoSource struct {
	Type           string                      `json:"type"`
	URL            string                      `json:"url,omitempty"`
	AssetID        string                      `json:"asset_id,omitempty"`
	SystemMetadata *IndexedAssetSystemMetadata `json:"system_metadata,omitempty"`
}

// TaskData decodes the event's Data field as task information.
func (e *WebhookEvent) TaskData() (*WebhookTaskData, error) {
	if e == nil {
		return nil, validationError("event", "is nil")
	}
	var data WebhookTaskData
	if err := json.Unmarshal(e.Data, &data); err != nil {
		return nil, fmt.Errorf("twelvelabs: decode webhook data: %w", err)
	}
	return &data, nil
}

// ParseWebhookEvent decodes a webhook payload without verifying its signature.
// Prefer WebhookVerifier.VerifyAndParse.
func ParseWebhookEvent(payload []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("twelvelabs: decode webhook payload: %w", err)
	}
	if event.Type == "" {
		return nil, validationError("type", "is missing from the webhook payload")
	}
	return &event, nil
}

// WebhookVerifier checks the TL-Signature header of incoming webhooks.
//
// TwelveLabs signs webhooks with HMAC-SHA256 over "<timestamp>.<raw body>" and
// sends the result in TL-Signature, formatted as "t=<unix>,v1=<hex>". Verify
// against the raw request body, before any JSON decoding.
type WebhookVerifier struct {
	// Secret is the webhook signing secret shown in the TwelveLabs console.
	Secret []byte
	// Tolerance rejects signatures whose timestamp is further than this from
	// the current time. Zero disables the check. NewWebhookVerifier sets
	// DefaultWebhookTolerance.
	Tolerance time.Duration
	// Now overrides the clock, for tests.
	Now func() time.Time
}

// NewWebhookVerifier returns a verifier with the default tolerance.
func NewWebhookVerifier(secret string) *WebhookVerifier {
	return &WebhookVerifier{Secret: []byte(secret), Tolerance: DefaultWebhookTolerance}
}

// Verify checks a TL-Signature header against the raw payload.
func (v *WebhookVerifier) Verify(payload []byte, signatureHeader string) error {
	if v == nil || len(v.Secret) == 0 {
		return ErrMissingWebhookSecret
	}
	timestamp, signatures, err := parseSignatureHeader(signatureHeader)
	if err != nil {
		return err
	}

	if v.Tolerance > 0 {
		now := time.Now()
		if v.Now != nil {
			now = v.Now()
		}
		age := now.Sub(time.Unix(timestamp, 0))
		if age < 0 {
			age = -age
		}
		if age > v.Tolerance {
			return ErrWebhookTimestampOutsideTolerance
		}
	}

	mac := hmac.New(sha256.New, v.Secret)
	fmt.Fprintf(mac, "%d.", timestamp)
	mac.Write(payload)
	expected := mac.Sum(nil)

	for _, signature := range signatures {
		decoded, err := hex.DecodeString(signature)
		if err != nil {
			continue
		}
		if hmac.Equal(decoded, expected) {
			return nil
		}
	}
	return ErrInvalidWebhookSignature
}

// VerifyAndParse verifies the signature and decodes the payload.
func (v *WebhookVerifier) VerifyAndParse(payload []byte, signatureHeader string) (*WebhookEvent, error) {
	if err := v.Verify(payload, signatureHeader); err != nil {
		return nil, err
	}
	return ParseWebhookEvent(payload)
}

// SignWebhookPayload builds a TL-Signature header value. It is useful for tests
// and for replaying payloads against a local handler.
func SignWebhookPayload(secret string, timestamp time.Time, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.", timestamp.Unix())
	mac.Write(payload)
	return fmt.Sprintf("t=%d,v1=%s", timestamp.Unix(), hex.EncodeToString(mac.Sum(nil)))
}

func parseSignatureHeader(header string) (int64, []string, error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0, nil, ErrMalformedWebhookSignature
	}

	var (
		timestamp  int64
		haveTime   bool
		signatures []string
	)
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return 0, nil, ErrMalformedWebhookSignature
			}
			timestamp = parsed
			haveTime = true
		case "v1":
			if value != "" {
				signatures = append(signatures, value)
			}
		}
	}

	if !haveTime || len(signatures) == 0 {
		return 0, nil, ErrMalformedWebhookSignature
	}
	return timestamp, signatures, nil
}
