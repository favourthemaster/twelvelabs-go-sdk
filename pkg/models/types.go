package models

// Core data types and models for the TwelveLabs Go SDK

type Task struct {
	ID             string                 `json:"_id"`
	Status         string                 `json:"status"`
	VideoID        string                 `json:"video_id"`
	IndexID        string                 `json:"index_id"`
	SystemMetadata map[string]interface{} `json:"system_metadata"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
}

type Index struct {
	ID        string  `json:"_id"`
	IndexName string  `json:"index_name"`
	Models    []Model `json:"models"`
	CreatedAt string  `json:"created_at"`
}

type Model struct {
	ModelName    string   `json:"model_name"`
	ModelOptions []string `json:"model_options"`
}

type Video struct {
	ID        string  `json:"_id"`
	IndexID   string  `json:"index_id"`
	FileName  string  `json:"file_name"`
	Duration  float64 `json:"duration"`
	CreatedAt string  `json:"created_at"`
}

type SearchResult struct {
	VideoID       string                 `json:"video_id"`
	Score         float64                `json:"score"`
	Start         float64                `json:"start"`
	End           float64                `json:"end"`
	Confidence    string                 `json:"confidence"`
	ThumbnailURL  string                 `json:"thumbnail_url,omitempty"`
	Transcription string                 `json:"transcription,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Search pool information
type SearchPool struct {
	TotalCount    int    `json:"total_count"`
	TotalDuration int    `json:"total_duration"`
	IndexID       string `json:"index_id"`
}

// Request types
type TasksCreateRequest struct {
	IndexID             string            `json:"index_id"`
	VideoFile           string            `json:"video_file,omitempty"`
	VideoURL            string            `json:"video_url,omitempty"`
	VideoStartOffsetSec int               `json:"video_start_offset_sec,omitempty"`
	VideoEndOffsetSec   int               `json:"video_end_offset_sec,omitempty"`
	VideoClipLength     int               `json:"video_clip_length,omitempty"`
	VideoEmbeddingScope []string          `json:"video_embedding_scope,omitempty"`
	EnableVideoStream   bool              `json:"enable_video_stream,omitempty"`
	UserMetadata        map[string]string `json:"user_metadata,omitempty"`
}

type IndexCreateRequest struct {
	IndexName string  `json:"index_name"`
	Models    []Model `json:"models"`
}

type IndexUpdateRequest struct {
	IndexName string  `json:"index_name,omitempty"`
	Models    []Model `json:"models,omitempty"`
}

type EmbedRequest struct {
	ModelName    string `json:"model_name"`
	VideoID      string `json:"video_id,omitempty"`
	Text         string `json:"text,omitempty"`
	TextTruncate string `json:"text_truncate,omitempty"`
	ImageURL     string `json:"image_url,omitempty"`
	ImageFile    string `json:"image_file,omitempty"`
	AudioURL     string `json:"audio_url,omitempty"`
	AudioFile    string `json:"audio_file,omitempty"`
	VideoURL     string `json:"video_url,omitempty"`
	VideoFile    string `json:"video_file,omitempty"`
}

type VideoUpdateRequest struct {
	FileName     string            `json:"file_name,omitempty"`
	UserMetadata map[string]string `json:"user_metadata,omitempty"`
}

type SearchQueryRequest struct {
	IndexID               string   `json:"index_id"`
	QueryText             string   `json:"query_text,omitempty"`
	QueryMediaType        string   `json:"query_media_type,omitempty"`
	QueryMediaFile        string   `json:"query_media_file,omitempty"`
	QueryMediaURL         string   `json:"query_media_url,omitempty"`
	ConversationOption    string   `json:"conversation_option,omitempty"`
	Filter                string   `json:"filter,omitempty"`
	SearchOptions         []string `json:"search_options,omitempty"`
	Threshold             string   `json:"threshold,omitempty"`
	SortOption            string   `json:"sort_option,omitempty"`
	AdjustConfidenceLevel float64  `json:"adjust_confidence_level,omitempty"`
	IncludeClips          bool     `json:"include_clips,omitempty"`
}

type SearchRequest struct {
	IndexID               string   `json:"index_id"`
	QueryText             string   `json:"query_text,omitempty"`
	QueryMediaType        string   `json:"query_media_type,omitempty"`
	QueryMediaFile        string   `json:"query_media_file,omitempty"`
	QueryMediaURL         string   `json:"query_media_url,omitempty"`
	ConversationOption    string   `json:"conversation_option,omitempty"`
	Filter                string   `json:"filter,omitempty"`
	SearchOptions         []string `json:"search_options,omitempty"`
	Threshold             string   `json:"threshold,omitempty"`
	SortOption            string   `json:"sort_option,omitempty"`
	AdjustConfidenceLevel float64  `json:"adjust_confidence_level,omitempty"`
	IncludeClips          bool     `json:"include_clips,omitempty"`
	PageLimit             int      `json:"page_limit,omitempty"`
	PageToken             string   `json:"page_token,omitempty"`
}

// Response types
type SearchResponse struct {
	SearchID   string         `json:"search_id,omitempty"`
	Data       []SearchResult `json:"data"`
	SearchPool *SearchPool    `json:"search_pool,omitempty"`
	PageInfo   *PageInfo      `json:"page_info,omitempty"`
}

type PageInfo struct {
	LimitPerPage  int    `json:"limit_per_page"`
	TotalResults  int    `json:"total_results"`
	PageExpiredAt string `json:"page_expired_at"`
	NextPageToken string `json:"next_page_token,omitempty"`
	PrevPageToken string `json:"prev_page_token,omitempty"`
}

type EmbedResponse struct {
	ModelName      string                `json:"model_name,omitempty"`
	VideoEmbedding *VideoEmbeddingResult `json:"video_embedding,omitempty"`
	TextEmbedding  *TextEmbeddingResult  `json:"text_embedding,omitempty"`
	AudioEmbedding *AudioEmbeddingResult `json:"audio_embedding,omitempty"`
	ImageEmbedding *ImageEmbeddingResult `json:"image_embedding,omitempty"`
}

type VideoEmbeddingResult struct {
	Segments []EmbeddingSegment     `json:"segments"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type TextEmbeddingResult struct {
	Segments []EmbeddingSegment     `json:"segments"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type AudioEmbeddingResult struct {
	Segments []EmbeddingSegment     `json:"segments"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type ImageEmbeddingResult struct {
	Segments []EmbeddingSegment     `json:"segments"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type EmbeddingSegment struct {
	Float          []float64 `json:"float"`
	StartOffsetSec *float64  `json:"start_offset_sec,omitempty"`
	EndOffsetSec   *float64  `json:"end_offset_sec,omitempty"`
}

// Legacy EmbeddingData struct for backward compatibility
type EmbeddingData struct {
	Embedding []float64 `json:"embedding"`
	StartTime float64   `json:"start_time"`
	EndTime   float64   `json:"end_time"`
}

// Analyze request and response types

type GenerateGistRequest struct {
	VideoID string   `json:"video_id"`
	Types   []string `json:"types"` // title, topic, hashtag
}

type GenerateGistResponse struct {
	ID       string   `json:"id"`
	Title    string   `json:"title,omitempty"`
	Topics   []string `json:"topics,omitempty"`
	Hashtags []string `json:"hashtags,omitempty"`
	Usage    *struct {
		OutputTokens int `json:"output_tokens,omitempty"`
	}
}

type AnalyzeRequest struct {
	VideoID     string  `json:"video_id,omitempty"`
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature,omitempty"`
	Stream      bool    `json:"stream,omitempty"`
}

type GenerateSummaryRequest struct {
	VideoID     string  `json:"video_id,omitempty"`
	Type        string  `json:"type"` //summary, chapters, highlights
	Prompt      string  `json:"prompt"`
	Temperature float64 `json:"temperature,omitempty"`
}

type GenerateSummaryResponse struct {
	Type     string `json:"summarize_type"` // summary, chapters, highlights
	ID       string `json:"id"`
	Summary  string `json:"summary,omitempty"`
	Chapters []struct {
		Number  string  `json:"chapter_number"`
		Title   string  `json:"chapter_title"`
		Start   float64 `json:"start_sec"`
		End     float64 `json:"end_sec"`
		Summary string  `json:"chapter_summary"`
	} `json:"chapters,omitempty"`
	Highlights []struct {
		Start   float64 `json:"start_sec"`
		End     float64 `json:"end_sec"`
		Title   string  `json:"highlight"`
		Summary string  `json:"highlight_summary"`
	} `json:"highlights,omitempty"`
	Usage *struct {
		OutputTokens int `json:"output_tokens,omitempty"`
	}
}

type AnalyzeResponse struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

type AnalyzeStreamResponse struct {
	EventType string `json:"event_type"`
	Text      string `json:"text,omitempty"`
	Metadata  *struct {
		GenerationID string `json:"generation_id,omitempty"`
		Usage        *struct {
			OutputTokens int `json:"output_tokens,omitempty"`
		} `json:"usage,omitempty"`
	} `json:"metadata,omitempty"`
}

// Helper methods for EmbedResponse to provide consistent access to embeddings
func (e *EmbedResponse) GetEmbeddings() []float64 {
	// Return the appropriate embeddings based on which type was created
	if e.TextEmbedding != nil && len(e.TextEmbedding.Segments) > 0 {
		return e.TextEmbedding.Segments[0].Float
	}
	if e.ImageEmbedding != nil && len(e.ImageEmbedding.Segments) > 0 {
		return e.ImageEmbedding.Segments[0].Float
	}
	if e.VideoEmbedding != nil && len(e.VideoEmbedding.Segments) > 0 {
		return e.VideoEmbedding.Segments[0].Float
	}
	if e.AudioEmbedding != nil && len(e.AudioEmbedding.Segments) > 0 {
		return e.AudioEmbedding.Segments[0].Float
	}
	return nil
}

func (e *EmbedResponse) GetAllVideoSegments() []EmbeddingSegment {
	if e.VideoEmbedding != nil {
		return e.VideoEmbedding.Segments
	}
	return nil
}

func (e *EmbedResponse) GetAllAudioSegments() []EmbeddingSegment {
	if e.AudioEmbedding != nil {
		return e.AudioEmbedding.Segments
	}
	return nil
}

func (e *EmbedResponse) GetAllTextSegments() []EmbeddingSegment {
	if e.TextEmbedding != nil {
		return e.TextEmbedding.Segments
	}
	return nil
}

func (e *EmbedResponse) GetAllImageSegments() []EmbeddingSegment {
	if e.ImageEmbedding != nil {
		return e.ImageEmbedding.Segments
	}
	return nil
}
