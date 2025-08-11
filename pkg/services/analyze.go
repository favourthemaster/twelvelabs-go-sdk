package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"io"
)

type AnalyzeService struct {
	Client ClientInterface
}

// Analyze performs video analysis with the given request parameters
func (s *AnalyzeService) Analyze(reqBody *models.AnalyzeRequest) (*models.AnalyzeResponse, error) {
	// Handle JSON request for video_id or video_url
	req, err := s.Client.NewRequest("POST", "/analyze", reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create analyze request: %w", err)
	}

	var response models.AnalyzeResponse
	_, err = s.Client.Do(req, &response)
	if err != nil {
		return nil, fmt.Errorf("analyze request failed: %w", err)
	}

	return &response, nil
}

// AnalyzeStream performs streaming video analysis
func (s *AnalyzeService) AnalyzeStream(reqBody *models.AnalyzeRequest, callback func(*models.AnalyzeStreamResponse) error) error {
	// Set stream to true for streaming requests
	streamReq := *reqBody
	streamReq.Stream = true

	// Handle JSON request for video_id or video_url
	req, err := s.Client.NewRequest("POST", "/analyze", &streamReq)
	if err != nil {
		return fmt.Errorf("failed to create analyze stream request: %w", err)
	}

	resp, err := s.Client.DoRaw(req)
	if err != nil {
		return fmt.Errorf("analyze stream request failed: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}(resp.Body)

	return s.processStreamResponse(resp.Body, callback)
}

// processStreamResponse processes the streaming response
func (s *AnalyzeService) processStreamResponse(body io.Reader, callback func(*models.AnalyzeStreamResponse) error) error {
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse JSON directly (no SSE format, just JSON objects)
		var streamResp models.AnalyzeStreamResponse
		if err := json.Unmarshal([]byte(line), &streamResp); err != nil {
			// Skip lines that aren't valid JSON (might be connection keep-alives)
			continue
		}

		if err := callback(&streamResp); err != nil {
			return fmt.Errorf("callback error: %w", err)
		}

		// Stop processing if we hit a stream_end event
		if streamResp.EventType == "stream_end" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream response: %w", err)
	}

	return nil
}

func (s *AnalyzeService) GenerateGist(reqBody *models.GenerateGistRequest) (*models.GenerateGistResponse, error) {
	req, err := s.Client.NewRequest("POST", "/gist", reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create gist request: %w", err)
	}

	var response models.GenerateGistResponse
	_, err = s.Client.Do(req, &response)
	if err != nil {
		return nil, fmt.Errorf("gist request failed: %w", err)
	}

	return &response, nil
}

func (s *AnalyzeService) GenerateSummary(reqBody *models.GenerateSummaryRequest) (*models.GenerateSummaryResponse, error) {
	req, err := s.Client.NewRequest("POST", "/summarize", reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create summarize request: %w", err)
	}

	var response models.GenerateSummaryResponse
	_, err = s.Client.Do(req, &response)
	if err != nil {
		return nil, fmt.Errorf("summarize request failed: %w", err)
	}

	return &response, nil
}
