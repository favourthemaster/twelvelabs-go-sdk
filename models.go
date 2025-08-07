package twelvelabs

type Task struct {
	ID        string `json:"_id"`
	Status    string `json:"status"`
	VideoID   string `json:"video_id"`
	IndexID   string `json:"index_id"`
	CreatedAt string `json:"created_at"`
}

type TasksCreateRequest struct {
	IndexID   string `json:"index_id"`
	VideoFile string `json:"-"` // This field will be handled separately for multipart/form-data
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

type IndexesCreateRequest struct {
	IndexName string  `json:"index_name"`
	Models    []Model `json:"models"`
}

type SearchResult struct {
	VideoID    string  `json:"video_id"`
	Score      float64 `json:"score"`
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Confidence string  `json:"confidence"`
}

type SearchQueryRequest struct {
	IndexID       string   `json:"index_id"`
	QueryText     string   `json:"query_text"`
	SearchOptions []string `json:"search_options"`
}

type EmbedRequest struct {
	VideoID string `json:"video_id"`
	ModelID string `json:"model_id"`
}

type EmbedResponse struct {
	Embeddings []float64 `json:"embeddings"`
}

type Video struct {
	ID        string  `json:"_id"`
	IndexID   string  `json:"index_id"`
	FileName  string  `json:"file_name"`
	Duration  float64 `json:"duration"`
	CreatedAt string  `json:"created_at"`
}
