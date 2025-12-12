package wrappers

import (
	"context"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/errors"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/services"
)

// EmbedWrapper provides high-level embedding generation capabilities for multiple media types
// including text, images, videos, and audio content using TwelveLabs foundation models.
type EmbedWrapper struct {
	service *services.EmbedService
}

// NewEmbedWrapper creates a new EmbedWrapper instance.
func NewEmbedWrapper(service *services.EmbedService) *EmbedWrapper {
	return &EmbedWrapper{service: service}
}

// EmbedWrapperRequest represents a comprehensive embedding request supporting all media types.
// Only specify the fields relevant to your embedding type (e.g., Text for text embeddings).
type EmbedWrapperRequest struct {
	// ModelName specifies the embedding model to use (e.g., "Marengo-retrieval-2.7")
	ModelName string `json:"model_name"`

	// Video embedding options (use one of: VideoID, VideoFile, or VideoURL)
	VideoID   string `json:"video_id"`   // Video ID from uploaded content
	VideoFile string `json:"video_file"` // Local video file path
	VideoURL  string `json:"video_url"`  // Publicly accessible video URL

	// Text embedding option
	Text string `json:"text"` // Text content to embed

	// Audio embedding options (use one of: AudioFile or AudioURL)
	AudioFile string `json:"audio_file"` // Local audio file path
	AudioURL  string `json:"audio_url"`  // Publicly accessible audio URL

	// Image embedding options (use one of: ImageFile or ImageURL)
	ImageFile string `json:"image_file"` // Local image file path
	ImageURL  string `json:"image_url"`  // Publicly accessible image URL
}

// Create generates embeddings for any supported media type based on the request content.
// This unified method automatically detects the embedding type from the provided fields.
//
// Supported Models:
//   - "Marengo-retrieval-2.7": Latest multimodal embedding model
//   - "Marengo-retrieval-2.6": Previous generation model
//
// Parameters:
//   - request: EmbedWrapperRequest with ModelName and content (text, video, audio, or image)
//
// Returns:
//   - EmbedResponse containing the generated embedding vector and metadata
//   - error if embedding generation fails
//
// Examples:
//
//	// Text embedding
//	response, err := client.Embed.Create(&wrappers.EmbedWrapperRequest{
//	    ModelName: "Marengo-retrieval-2.7",
//	    Text:      "A person running through a forest trail",
//	})
//
//	// Video embedding from uploaded content
//	response, err := client.Embed.Create(&wrappers.EmbedWrapperRequest{
//	    ModelName: "Marengo-retrieval-2.7",
//	    VideoID:   "your_video_id",
//	})
//
//	// Image embedding from URL
//	response, err := client.Embed.Create(&wrappers.EmbedWrapperRequest{
//	    ModelName: "Marengo-retrieval-2.7",
//	    ImageURL:  "https://example.com/image.jpg",
//	})
//
//	// Audio embedding from local file
//	response, err := client.Embed.Create(&wrappers.EmbedWrapperRequest{
//	    ModelName: "Marengo-retrieval-2.7",
//	    AudioFile: "./audio/sample.mp3",
//	})
func (ew *EmbedWrapper) Create(ctx context.Context, request *EmbedWrapperRequest) (*models.EmbedResponse, error) {
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
	result, err := ew.service.Create(ctx, baseRequest)
	if err != nil {
		return nil, errors.NewServiceError("Embed", "embedding creation failed: "+err.Error())
	}

	return result, nil
}

// CreateVideoEmbedding is a convenience method for video embeddings
func (ew *EmbedWrapper) CreateVideoEmbedding(ctx context.Context, modelName, videoURL string) (*models.EmbedResponse, error) {
	request := &EmbedWrapperRequest{
		ModelName: modelName,
		VideoURL:  videoURL,
	}
	return ew.Create(ctx, request)
}

// CreateTextEmbedding is a convenience method for generating text embeddings.
// This method simplifies the most common embedding use case.
//
// Parameters:
//   - modelName: The embedding model to use (e.g., "Marengo-retrieval-2.7")
//   - text: The text content to embed
//
// Returns:
//   - EmbedResponse containing the text embedding vector
//   - error if embedding generation fails
//
// Example:
//
//	embedding, err := client.Embed.CreateTextEmbedding(
//	    "Marengo-retrieval-2.7",
//	    "A beautiful sunset over the ocean",
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Generated %d-dimensional embedding\n", len(embedding.Embeddings))
func (ew *EmbedWrapper) CreateTextEmbedding(ctx context.Context, modelName, text string) (*models.EmbedResponse, error) {
	request := &EmbedWrapperRequest{
		ModelName: modelName,
		Text:      text,
	}
	return ew.Create(ctx, request)
}

// CreateImageEmbedding is a convenience method for image embeddings
func (ew *EmbedWrapper) CreateImageEmbedding(ctx context.Context, modelName, imageURL string) (*models.EmbedResponse, error) {
	request := &EmbedWrapperRequest{
		ModelName: modelName,
		ImageURL:  imageURL,
	}
	return ew.Create(ctx, request)
}

// CreateAudioEmbedding is a convenience method for audio embeddings
func (ew *EmbedWrapper) CreateAudioEmbedding(ctx context.Context, modelName, audioURL string) (*models.EmbedResponse, error) {
	request := &EmbedWrapperRequest{
		ModelName: modelName,
		AudioURL:  audioURL,
	}
	return ew.Create(ctx, request)
}
