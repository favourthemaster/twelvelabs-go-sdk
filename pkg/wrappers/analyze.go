// Package wrappers provides high-level wrapper interfaces for TwelveLabs API services.
// These wrappers add convenience methods and enhanced error handling over the base services.
package wrappers

import (
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/errors"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/services"
)

// AnalyzeWrapper provides high-level video analysis capabilities including
// AI-powered content analysis, summarization, gist generation, and streaming responses.
type AnalyzeWrapper struct {
	service *services.AnalyzeService
}

// NewAnalyzeWrapper creates a new AnalyzeWrapper instance.
func NewAnalyzeWrapper(service *services.AnalyzeService) *AnalyzeWrapper {
	return &AnalyzeWrapper{service: service}
}

// Analyze performs AI-powered video analysis with a custom prompt.
// This method analyzes video content and returns insights based on your specific question or prompt.
//
// Parameters:
//   - request: AnalyzeRequest containing VideoID, Prompt, and optional parameters like Temperature and Stream
//
// Returns:
//   - AnalyzeResponse containing the analysis results and metadata
//   - error if the analysis fails
//
// Example:
//
//	response, err := client.Analyze.Analyze(&models.AnalyzeRequest{
//	    VideoID:     "video_id_here",
//	    Prompt:      "What objects and people can you see in this video?",
//	    Temperature: 0.7, // Optional: controls response creativity (0.0-1.0)
//	    Stream:      false, // Set to true for streaming responses
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Analysis:", response.Data)
func (aw *AnalyzeWrapper) Analyze(request *models.AnalyzeRequest) (*models.AnalyzeResponse, error) {
	result, err := aw.service.Analyze(request)
	if err != nil {
		return nil, errors.NewServiceError("Analyze", "video analysis failed: "+err.Error())
	}

	return result, nil
}

// AnalyzeStream performs streaming video analysis, calling the provided callback
// function for each chunk of the response as it arrives in real-time.
//
// This is useful for long analysis requests where you want to show progress
// or start processing results before the full analysis is complete.
//
// Parameters:
//   - request: AnalyzeRequest with VideoID and Prompt (Stream field is automatically set to true)
//   - callback: Function called for each streaming event with AnalyzeStreamResponse
//
// Returns:
//   - error if the streaming analysis fails
//
// The callback receives events with different EventType values:
//   - "stream_start": Analysis has begun
//   - "text_generation": New text chunk available in the Text field
//   - "stream_end": Analysis completed, check Metadata for usage info
//
// Example:
//
//	err := client.Analyze.AnalyzeStream(&models.AnalyzeRequest{
//	    VideoID: "video_id_here",
//	    Prompt:  "Describe what happens in this video step by step",
//	}, func(event *models.AnalyzeStreamResponse) error {
//	    switch event.EventType {
//	    case "text_generation":
//	        fmt.Print(event.Text) // Print each chunk as it arrives
//	    case "stream_end":
//	        fmt.Println("\nAnalysis completed!")
//	    }
//	    return nil
//	})
func (aw *AnalyzeWrapper) AnalyzeStream(request *models.AnalyzeRequest, callback func(*models.AnalyzeStreamResponse) error) error {
	err := aw.service.AnalyzeStream(request, callback)
	if err != nil {
		return errors.NewServiceError("Analyze", "streaming video analysis failed: "+err.Error())
	}

	return nil
}

// GenerateSummary creates various types of video summaries including general summaries,
// chapter breakdowns with timestamps, and highlight reels.
//
// Parameters:
//   - request: GenerateSummaryRequest with VideoID, Type, and optional Prompt
//
// Supported Types:
//   - "summary": General video summary
//   - "chapter": Chapter titles with timestamps
//   - "highlight": Key highlights and moments
//
// Returns:
//   - GenerateSummaryResponse with the generated content in the appropriate field
//   - error if summary generation fails
//
// Example:
//
//	// Generate a general summary
//	summary, err := client.Analyze.GenerateSummary(&models.GenerateSummaryRequest{
//	    VideoID: "video_id",
//	    Type:    "summary",
//	    Prompt:  "Provide a brief overview of the video content",
//	})
//
//	// Generate chapters with timestamps
//	chapters, err := client.Analyze.GenerateSummary(&models.GenerateSummaryRequest{
//	    VideoID: "video_id",
//	    Type:    "chapter",
//	})
func (aw *AnalyzeWrapper) GenerateSummary(request *models.GenerateSummaryRequest) (*models.GenerateSummaryResponse, error) {
	result, err := aw.service.GenerateSummary(request)
	if err != nil {
		return nil, errors.NewServiceError("Analyze", "video summary generation failed: "+err.Error())
	}

	return result, nil
}

// GenerateGist creates concise video metadata including titles, topics, and hashtags.
// This is useful for content organization, SEO, and social media optimization.
//
// Parameters:
//   - request: GenerateGistRequest with VideoID and Types array
//
// Supported Types:
//   - "title": Auto-generated video titles
//   - "topic": Main topics and themes
//   - "hashtag": Relevant hashtags for social media
//
// Returns:
//   - GenerateGistResponse with the requested gist information
//   - error if gist generation fails
//
// Example:
//
//	gist, err := client.Analyze.GenerateGist(&models.GenerateGistRequest{
//	    VideoID: "video_id",
//	    Types:   []string{"title", "topic", "hashtag"},
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Title: %s\n", gist.Title)
//	fmt.Printf("Topics: %s\n", gist.Topics)
//	fmt.Printf("Hashtags: %v\n", gist.Hashtags)
func (aw *AnalyzeWrapper) GenerateGist(request *models.GenerateGistRequest) (*models.GenerateGistResponse, error) {
	result, err := aw.service.GenerateGist(request)
	if err != nil {
		return nil, errors.NewServiceError("Analyze", "video gist generation failed: "+err.Error())
	}

	return result, nil
}
