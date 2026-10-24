package twelvelabs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

// Asset status values.
const (
	AssetStatusProcessing = "processing"
	AssetStatusReady      = "ready"
	AssetStatusFailed     = "failed"
)

// Asset upload methods.
const (
	AssetMethodDirect    = "direct"
	AssetMethodURL       = "url"
	AssetMethodMultipart = "multipart"
	AssetMethodConnector = "connector"
)

// AssetsService uploads and manages reusable media assets.
type AssetsService struct {
	client *Client
}

// Asset is a reusable file: video, audio, image, PDF, plain text, or Markdown.
type Asset struct {
	ID           string         `json:"_id"`
	Method       string         `json:"method"`
	Status       string         `json:"status"`
	Filename     string         `json:"filename"`
	FileType     string         `json:"file_type"`
	CreatedAt    string         `json:"created_at"`
	UserMetadata map[string]any `json:"user_metadata,omitempty"`
	Source       *AssetSource   `json:"source,omitempty"`
}

// AssetSource describes where an imported asset came from.
type AssetSource struct {
	Type    string         `json:"type,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

// AssetDetail is the full representation returned when retrieving an asset.
type AssetDetail struct {
	Asset
	HLS               *AssetHLS          `json:"hls,omitempty"`
	Thumbnail         *AssetThumbnail    `json:"thumbnail,omitempty"`
	TechnicalMetadata *TechnicalMetadata `json:"technical_metadata,omitempty"`
	Size              int64              `json:"size,omitempty"`
	Duration          float64            `json:"duration,omitempty"`
	Error             *ErrorBody         `json:"error,omitempty"`
}

// Asset HLS status values.
const (
	AssetHLSStatusPending    = "pending"
	AssetHLSStatusProcessing = "processing"
	AssetHLSStatusReady      = "ready"
	AssetHLSStatusError      = "error"
)

// AssetHLS describes HLS generation state for an asset.
type AssetHLS struct {
	// ManifestURL is the playback URL. It is set only when Status is ready.
	ManifestURL string `json:"manifest_url,omitempty"`
	Status      string `json:"status,omitempty"`
}

// AssetThumbnail describes thumbnail generation state.
type AssetThumbnail struct {
	RepresentativeURL string `json:"representative_url,omitempty"`
	Status            string `json:"status,omitempty"`
}

// TechnicalMetadata describes the media the platform inspected.
type TechnicalMetadata struct {
	FileSizeBytes            int64         `json:"file_size_bytes,omitempty"`
	FileMIMEType             string        `json:"file_mime_type,omitempty"`
	FileContainerFormat      string        `json:"file_container_format,omitempty"`
	ContainerCreation        string        `json:"container_creation_time,omitempty"`
	VideoStreams             []VideoStream `json:"video_streams,omitempty"`
	VideoCodec               string        `json:"video_codec,omitempty"`
	VideoWidth               int           `json:"video_width,omitempty"`
	VideoHeight              int           `json:"video_height,omitempty"`
	VideoFPS                 float64       `json:"video_fps,omitempty"`
	VideoDurationSeconds     float64       `json:"video_duration_seconds,omitempty"`
	VideoBitrateBPS          int64         `json:"video_bitrate_bps,omitempty"`
	AudioStreams             []AudioStream `json:"audio_streams,omitempty"`
	AudioCodec               string        `json:"audio_codec,omitempty"`
	AudioSampleRate          int           `json:"audio_sample_rate,omitempty"`
	AudioChannels            int           `json:"audio_channels,omitempty"`
	AudioDurationSeconds     float64       `json:"audio_duration_seconds,omitempty"`
	StartTimecode            string        `json:"start_timecode,omitempty"`
	TimecodeSource           string        `json:"timecode_source,omitempty"`
	DropFrame                bool          `json:"drop_frame,omitempty"`
	ImageWidth               int           `json:"image_width,omitempty"`
	ImageHeight              int           `json:"image_height,omitempty"`
	ImageFormat              string        `json:"image_format,omitempty"`
	ImageOrientation         int           `json:"image_orientation,omitempty"`
	ImageColorSpace          string        `json:"image_color_space,omitempty"`
	ImageBitDepth            int           `json:"image_bit_depth,omitempty"`
	IsHDR                    bool          `json:"is_hdr,omitempty"`
	HasAudio                 bool          `json:"has_audio,omitempty"`
	HasAlpha                 bool          `json:"has_alpha,omitempty"`
	IsAnimated               bool          `json:"is_animated,omitempty"`
	TotalVideoStreams        int           `json:"total_video_streams,omitempty"`
	TotalAudioStreams        int           `json:"total_audio_streams,omitempty"`
	StorageAspectRatio       float64       `json:"storage_aspect_ratio,omitempty"`
	GeospatialLatitude       *float64      `json:"geospatial_latitude,omitempty"`
	GeospatialLongitude      *float64      `json:"geospatial_longitude,omitempty"`
	GeospatialAltitudeM      *float64      `json:"geospatial_altitude_meters,omitempty"`
	DocumentPageCount        *int          `json:"document_page_count,omitempty"`
	DocumentPDFVersion       *string       `json:"document_pdf_version,omitempty"`
	DocumentPageWidthPt      *float64      `json:"document_page_width_pt,omitempty"`
	DocumentPageHeightPt     *float64      `json:"document_page_height_pt,omitempty"`
	DocumentIsTagged         *bool         `json:"document_is_tagged,omitempty"`
	DocumentIsEncrypted      *bool         `json:"document_is_encrypted,omitempty"`
	DocumentTitle            *string       `json:"document_title,omitempty"`
	DocumentAuthor           *string       `json:"document_author,omitempty"`
	DocumentLanguage         *string       `json:"document_language,omitempty"`
	DocumentSubject          *string       `json:"document_subject,omitempty"`
	DocumentKeywords         *string       `json:"document_keywords,omitempty"`
	DocumentCreatorTool      *string       `json:"document_creator_tool,omitempty"`
	DocumentProducer         *string       `json:"document_producer,omitempty"`
	DocumentCreationDate     *string       `json:"document_creation_date,omitempty"`
	DocumentModificationDate *string       `json:"document_modification_date,omitempty"`
}

// VideoStream describes one video stream of a media file.
type VideoStream struct {
	Index              int     `json:"index"`
	Codec              string  `json:"codec,omitempty"`
	Width              int     `json:"width,omitempty"`
	Height             int     `json:"height,omitempty"`
	FPS                float64 `json:"fps,omitempty"`
	AvgFPS             float64 `json:"avg_fps,omitempty"`
	DurationSeconds    float64 `json:"duration_seconds,omitempty"`
	BitrateBPS         int64   `json:"bitrate_bps,omitempty"`
	Rotation           int     `json:"rotation,omitempty"`
	PixelAspectRatio   string  `json:"pixel_aspect_ratio,omitempty"`
	DisplayAspectRatio string  `json:"display_aspect_ratio,omitempty"`
	ScanType           string  `json:"scan_type,omitempty"`
}

// AudioStream describes one audio stream of a media file.
type AudioStream struct {
	Index           int     `json:"index"`
	Codec           string  `json:"codec,omitempty"`
	CodecLong       string  `json:"codec_long,omitempty"`
	SampleRate      int     `json:"sample_rate,omitempty"`
	BitDepth        int     `json:"bit_depth,omitempty"`
	Channels        int     `json:"channels,omitempty"`
	ChannelLayout   string  `json:"channel_layout,omitempty"`
	BitrateBPS      int64   `json:"bitrate_bps,omitempty"`
	Language        string  `json:"language,omitempty"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
}

// AssetCreateRequest creates an asset from a local file or a public URL.
// Set exactly one of File or URL.
type AssetCreateRequest struct {
	// File uploads a local file (direct upload, up to 200 MB for video/audio
	// and documents, 32 MB for images).
	File *FileUpload
	// URL imports a publicly accessible file (up to 4 GB for video/audio).
	URL string
	// Filename overrides the name the platform stores. Optional.
	Filename string
	// EnableHLS generates an HLS playlist. The API defaults to true.
	EnableHLS *bool
	// EnableThumbnail generates thumbnails. The API defaults to true.
	EnableThumbnail *bool
	// UserMetadata is arbitrary key/value metadata.
	UserMetadata map[string]any
}

// AssetListOptions filters List.
type AssetListOptions struct {
	Pagination
	AssetIDs   []string
	AssetTypes []string
	Filename   string
}

// UpdateAssetUserMetadataRequest replaces or merges user metadata.
type UpdateAssetUserMetadataRequest struct {
	UserMetadata map[string]any `json:"user_metadata"`
}

// TranscriptionInclude selects which transcription segmentations to return.
type TranscriptionInclude string

const (
	TranscriptionWords      TranscriptionInclude = "words"
	TranscriptionSentences  TranscriptionInclude = "sentences"
	TranscriptionUtterances TranscriptionInclude = "utterances"
)

// TranscriptionEntry is one timestamped transcript fragment.
type TranscriptionEntry struct {
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	Value string  `json:"value"`
}

// TranscriptionUtterance is one speaker turn.
type TranscriptionUtterance struct {
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Value   string  `json:"value"`
	Speaker string  `json:"speaker,omitempty"`
}

// AssetTranscription is the transcription of an asset.
type AssetTranscription struct {
	Status     string                   `json:"status"`
	Words      []TranscriptionEntry     `json:"words,omitempty"`
	Sentences  []TranscriptionEntry     `json:"sentences,omitempty"`
	Utterances []TranscriptionUtterance `json:"utterances,omitempty"`
	Error      *ErrorBody               `json:"error,omitempty"`
}

// Create uploads or imports an asset. Processing is asynchronous: poll Get or
// call WaitForReady before using the asset.
func (s *AssetsService) Create(ctx context.Context, request *AssetCreateRequest) (*Asset, error) {
	ctx = ensureContext(ctx)
	if request == nil {
		return nil, validationError("request", "is required")
	}
	if request.File != nil && request.URL != "" {
		return nil, validationError("request", "set either File or URL, not both")
	}
	if request.File == nil && request.URL == "" {
		return nil, validationError("request", "one of File or URL is required")
	}
	if request.File != nil {
		if err := request.File.validate("File"); err != nil {
			return nil, err
		}
	}

	method := AssetMethodDirect
	if request.URL != "" {
		method = AssetMethodURL
	}

	body := &multipartBody{write: func(w *multipart.Writer) error {
		if err := w.WriteField("method", method); err != nil {
			return err
		}
		if request.File != nil {
			if err := writeFilePart(w, "file", request.File); err != nil {
				return err
			}
		} else if err := w.WriteField("url", request.URL); err != nil {
			return err
		}
		if request.Filename != "" {
			if err := w.WriteField("filename", request.Filename); err != nil {
				return err
			}
		}
		if request.EnableHLS != nil {
			if err := w.WriteField("enable_hls", boolString(*request.EnableHLS)); err != nil {
				return err
			}
		}
		if request.EnableThumbnail != nil {
			if err := w.WriteField("enable_thumbnail", boolString(*request.EnableThumbnail)); err != nil {
				return err
			}
		}
		return writeUserMetadataField(w, request.UserMetadata)
	}}

	var asset Asset
	if err := s.client.transport.doMultipart(ctx, http.MethodPost, "/assets", nil, body, &asset); err != nil {
		return nil, err
	}
	return &asset, nil
}

// List returns assets, most recent first.
func (s *AssetsService) List(ctx context.Context, options *AssetListOptions) (*Page[AssetDetail], error) {
	ctx = ensureContext(ctx)
	query := url.Values{}
	if options != nil {
		query = encodeQuery(&options.Pagination)
		addStrings(query, "asset_ids", options.AssetIDs)
		addStrings(query, "asset_types", options.AssetTypes)
		if options.Filename != "" {
			query.Set("filename", options.Filename)
		}
	}

	var page Page[AssetDetail]
	if err := s.client.transport.doJSON(ctx, http.MethodGet, "/assets", query, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// Get retrieves an asset.
func (s *AssetsService) Get(ctx context.Context, assetID string) (*AssetDetail, error) {
	ctx = ensureContext(ctx)
	if assetID == "" {
		return nil, validationError("assetID", "is required")
	}
	var asset AssetDetail
	path := "/assets/" + escapePath(assetID)
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, nil, nil, &asset); err != nil {
		return nil, err
	}
	return &asset, nil
}

// Delete removes an asset.
func (s *AssetsService) Delete(ctx context.Context, assetID string) error {
	ctx = ensureContext(ctx)
	if assetID == "" {
		return validationError("assetID", "is required")
	}
	return s.client.transport.doJSON(ctx, http.MethodDelete, "/assets/"+escapePath(assetID), nil, nil, nil)
}

// UpdateUserMetadata merges keys into the asset's user metadata.
func (s *AssetsService) UpdateUserMetadata(ctx context.Context, assetID string, metadata map[string]any) error {
	ctx = ensureContext(ctx)
	if assetID == "" {
		return validationError("assetID", "is required")
	}
	body := UpdateAssetUserMetadataRequest{UserMetadata: metadata}
	path := "/assets/" + escapePath(assetID) + "/user-metadata"
	return s.client.transport.doJSON(ctx, http.MethodPatch, path, nil, &body, nil)
}

// ReplaceUserMetadata replaces the asset's user metadata.
func (s *AssetsService) ReplaceUserMetadata(ctx context.Context, assetID string, metadata map[string]any) error {
	ctx = ensureContext(ctx)
	if assetID == "" {
		return validationError("assetID", "is required")
	}
	body := UpdateAssetUserMetadataRequest{UserMetadata: metadata}
	path := "/assets/" + escapePath(assetID) + "/user-metadata"
	return s.client.transport.doJSON(ctx, http.MethodPut, path, nil, &body, nil)
}

// DeleteUserMetadata removes all user metadata from the asset.
func (s *AssetsService) DeleteUserMetadata(ctx context.Context, assetID string) error {
	ctx = ensureContext(ctx)
	if assetID == "" {
		return validationError("assetID", "is required")
	}
	path := "/assets/" + escapePath(assetID) + "/user-metadata"
	return s.client.transport.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// Transcription retrieves the transcription of an asset.
func (s *AssetsService) Transcription(ctx context.Context, assetID string, include ...TranscriptionInclude) (*AssetTranscription, error) {
	ctx = ensureContext(ctx)
	if assetID == "" {
		return nil, validationError("assetID", "is required")
	}
	query := url.Values{}
	for _, value := range include {
		if value != "" {
			query.Add("include", string(value))
		}
	}
	var transcription AssetTranscription
	path := "/assets/" + escapePath(assetID) + "/transcription"
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, query, nil, &transcription); err != nil {
		return nil, err
	}
	return &transcription, nil
}

// ListIndexedAssets returns the indexed assets created from this asset.
func (s *AssetsService) ListIndexedAssets(ctx context.Context, assetID string, options *Pagination) (*Page[IndexedAsset], error) {
	ctx = ensureContext(ctx)
	if assetID == "" {
		return nil, validationError("assetID", "is required")
	}
	var page Page[IndexedAsset]
	path := "/assets/" + escapePath(assetID) + "/indexed-assets"
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, encodeQuery(options), nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// ListEntities returns the entities that reference this asset.
func (s *AssetsService) ListEntities(ctx context.Context, assetID string, options *Pagination) (*Page[Entity], error) {
	ctx = ensureContext(ctx)
	if assetID == "" {
		return nil, validationError("assetID", "is required")
	}
	var page Page[Entity]
	path := "/assets/" + escapePath(assetID) + "/entities"
	if err := s.client.transport.doJSON(ctx, http.MethodGet, path, encodeQuery(options), nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// AssetStorageAccounting reports how an asset counts toward the storage
// allowance.
type AssetStorageAccounting struct {
	AssetID           string `json:"asset_id"`
	StorageAccounting bool   `json:"storage_accounting"`
	Bytes             int64  `json:"bytes"`
}

// EnableStorageAccounting marks the asset as counted toward the storage
// allowance. The only accepted value is true.
func (s *AssetsService) EnableStorageAccounting(ctx context.Context, assetID string) (*AssetStorageAccounting, error) {
	ctx = ensureContext(ctx)
	if assetID == "" {
		return nil, validationError("assetID", "is required")
	}
	body := struct {
		Enabled bool `json:"enabled"`
	}{Enabled: true}
	var response AssetStorageAccounting
	path := "/assets/" + escapePath(assetID) + "/storage-accounting"
	if err := s.client.transport.doJSON(ctx, http.MethodPut, path, nil, &body, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// WaitForReady polls the asset until it is ready or failed.
func (s *AssetsService) WaitForReady(ctx context.Context, assetID string, options *WaitOptions) (*AssetDetail, error) {
	ctx = ensureContext(ctx)
	if assetID == "" {
		return nil, validationError("assetID", "is required")
	}

	var deadline time.Time
	if options != nil && options.Timeout > 0 {
		deadline = time.Now().Add(options.Timeout)
	}

	for {
		asset, err := s.Get(ctx, assetID)
		if err != nil {
			return nil, err
		}

		switch asset.Status {
		case AssetStatusReady:
			return asset, nil
		case AssetStatusFailed:
			return asset, fmt.Errorf("twelvelabs: asset %s failed: %s", assetID, asset.Error)
		}

		if options != nil && options.OnPoll != nil {
			if err := options.OnPoll(asset.Status); err != nil {
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

func writeFilePart(w *multipart.Writer, field string, file *FileUpload) error {
	filename := file.Filename
	if filename == "" {
		filename = "upload"
	}
	header := make(map[string][]string)
	header["Content-Disposition"] = []string{
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`, escapeQuotes(field), escapeQuotes(filename)),
	}
	header["Content-Type"] = []string{file.contentType()}

	part, err := w.CreatePart(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file.Content)
	return err
}

func writeUserMetadataField(w *multipart.Writer, metadata map[string]any) error {
	if len(metadata) == 0 {
		return nil
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode user metadata: %w", err)
	}
	return w.WriteField("user_metadata", string(encoded))
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func escapeQuotes(s string) string {
	replacer := strings.NewReplacer("\\", "\\\\", `"`, "\\\"")
	return replacer.Replace(s)
}

// assetTypeFromFilename guesses the multipart upload asset type from the file
// extension. The multipart session API requires an explicit type.
func assetTypeFromFilename(filename string) (string, error) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".mp4", ".mov", ".m4v", ".webm", ".mkv", ".avi", ".mpeg", ".mpg", ".ts", ".m3u8":
		return "video", nil
	case ".mp3", ".wav", ".m4a", ".aac", ".flac", ".ogg", ".opus":
		return "audio", nil
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".bmp", ".tiff":
		return "image", nil
	default:
		return "", validationError("Filename", "cannot infer asset type for %q; set AssetType explicitly", filename)
	}
}
