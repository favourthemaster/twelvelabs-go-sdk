package wrappers

import (
	"fmt"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/services"
)

// AnalyzeWrapper wraps the basic AnalyzeService with additional functionality
type AnalyzeWrapper struct {
	service *services.AnalyzeService
}

// NewAnalyzeWrapper creates a new AnalyzeWrapper
func NewAnalyzeWrapper(service *services.AnalyzeService) *AnalyzeWrapper {
	return &AnalyzeWrapper{service: service}
}

// Analyze performs video analysis with the given request
//
// Parameters:
//   - request: Analyze request containing the videoid, temperature, stream and prompt
//
// Returns: AnalyzeResponse containing the analysis results

//	response, err := client.Analyze.Analyze(&models.AnalyzeRequest{
//	    VideoID:   "video_id_here",
//	    Prompt:    "What objects can you see in this video?",
//	})
func (aw *AnalyzeWrapper) Analyze(request *models.AnalyzeRequest) (*models.AnalyzeResponse, error) {
	result, err := aw.service.Analyze(request)
	if err != nil {
		return nil, fmt.Errorf("video analysis failed: %w", err)
	}

	return result, nil
}

// AnalyzeStream performs streaming video analysis with the given request
//
// Parameters:
//   - request: Analyze request containing the video source, model, and prompt
//   - callback: Function called for each streaming chunk
//
// Returns: Error if the streaming fails
//
// Example for streaming analysis:
//
//	err := client.Analyze.AnalyzeStream(&AnalyzeWrapperRequest{
//	    VideoID:   "video_id_here",
//	    ModelName: "pegasus-1",
//	    Prompt:    "Describe what happens in this video",
//	}, func(chunk *models.AnalyzeStreamResponse) error {
//	    if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
//	        fmt.Print(chunk.Choices[0].Delta.Content)
//	    }
//	    return nil
//	})
func (aw *AnalyzeWrapper) AnalyzeStream(request *models.AnalyzeRequest, callback func(*models.AnalyzeStreamResponse) error) error {
	err := aw.service.AnalyzeStream(request, callback)
	if err != nil {
		return fmt.Errorf("streaming video analysis failed: %w", err)
	}

	return nil
}

func (aw *AnalyzeWrapper) GenerateSummary(request *models.GenerateSummaryRequest) (*models.GenerateSummaryResponse, error) {
	result, err := aw.service.GenerateSummary(request)
	if err != nil {
		return nil, fmt.Errorf("video summary generation failed: %w", err)
	}

	return result, nil
}

func (aw *AnalyzeWrapper) GenerateGist(request *models.GenerateGistRequest) (*models.GenerateGistResponse, error) {
	result, err := aw.service.GenerateGist(request)
	if err != nil {
		return nil, fmt.Errorf("video gist generation failed: %w", err)
	}

	return result, nil
}
