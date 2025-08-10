package wrappers

import (
	"fmt"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/services"
)

// EmbedWrapper wraps the basic EmbedService with additional functionality
type EmbedWrapper struct {
	service *services.EmbedService
}

// NewEmbedWrapper creates a new EmbedWrapper
func NewEmbedWrapper(service *services.EmbedService) *EmbedWrapper {
	return &EmbedWrapper{service: service}
}

// EmbedWrapperRequest represents an embedding request with enhanced functionality
type EmbedWrapperRequest struct {
	ModelName string `json:"model_name"`
	// For video embeddings
	VideoID   string `json:"video_id,omitempty"`
	VideoFile string `json:"video_file,omitempty"`
	VideoURL  string `json:"video_url,omitempty"`
	// For text embeddings
	Text string `json:"text,omitempty"`
	// For audio embeddings
	AudioFile string `json:"audio_file,omitempty"`
	AudioURL  string `json:"audio_url,omitempty"`
	// For image embeddings
	ImageFile string `json:"image_file,omitempty"`
	ImageURL  string `json:"image_url,omitempty"`
}

// Create generates embeddings based on the request type and content
// This method can handle video, text, audio, and image embeddings.
//
// Parameters:
//   - request: Embedding request containing the model name and content
//
// Returns: EmbedResponse containing the generated embeddings
//
// Example for video embedding:
//
//	response, err := client.Embed.Create(&EmbedWrapperRequest{
//	    ModelName: "Marengo-retrieval-2.6",
//	    VideoURL: "https://example.com/video.mp4",
//	})
//
// Example for text embedding:
//
//	response, err := client.Embed.Create(&EmbedWrapperRequest{
//	    ModelName: "Marengo-retrieval-2.6",
//	    Text: "A person running in the park",
//	})
func (ew *EmbedWrapper) Create(request *EmbedWrapperRequest) (*models.EmbedResponse, error) {
	// Convert to the base service request format
	baseRequest := &models.EmbedRequest{
		ModelName: request.ModelName,
		VideoID:   request.VideoID,
		VideoFile: request.VideoFile,
		VideoURL:  request.VideoURL,
		Text:      request.Text,
		ImageURL:  request.ImageURL,
		ImageFile: request.ImageFile,
		AudioURL:  request.AudioURL,
		AudioFile: request.AudioFile,
	}

	// Use the existing Create method from the base service
	result, err := ew.service.Create(baseRequest)
	if err != nil {
		return nil, fmt.Errorf("embedding creation failed: %w", err)
	}

	return result, nil
}

// CreateVideoEmbedding is a convenience method for video embeddings
func (ew *EmbedWrapper) CreateVideoEmbedding(modelName, videoURL string) (*models.EmbedResponse, error) {
	request := &EmbedWrapperRequest{
		ModelName: modelName,
		VideoURL:  videoURL,
	}
	return ew.Create(request)
}

// CreateTextEmbedding is a convenience method for text embeddings
func (ew *EmbedWrapper) CreateTextEmbedding(modelName, text string) (*models.EmbedResponse, error) {
	request := &EmbedWrapperRequest{
		ModelName: modelName,
		Text:      text,
	}
	return ew.Create(request)
}

// CreateImageEmbedding is a convenience method for image embeddings
func (ew *EmbedWrapper) CreateImageEmbedding(modelName, imageURL string) (*models.EmbedResponse, error) {
	request := &EmbedWrapperRequest{
		ModelName: modelName,
		ImageURL:  imageURL,
	}
	return ew.Create(request)
}

// CreateAudioEmbedding is a convenience method for audio embeddings
func (ew *EmbedWrapper) CreateAudioEmbedding(modelName, audioURL string) (*models.EmbedResponse, error) {
	request := &EmbedWrapperRequest{
		ModelName: modelName,
		AudioURL:  audioURL,
	}
	return ew.Create(request)
}
