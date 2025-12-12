package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/errors"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
)

type AnalyzeService struct {
	Client ClientInterface
}

// Analyze performs video analysis with the given request parameters
func (s *AnalyzeService) Analyze(ctx context.Context, reqBody *models.AnalyzeRequest) (*models.AnalyzeResponse, error) {
	req, err := s.Client.NewRequest(ctx, "POST", "/analyze", reqBody)
	if err != nil {
		return nil, errors.NewRequestError("failed to create analyze request: " + err.Error())
	}

	var response models.AnalyzeResponse
	_, err = s.Client.Do(req, &response)
	if err != nil {
		return nil, errors.NewServiceError("Analyze", "analyze request failed: "+err.Error())
	}

	return &response, nil
}

// AnalyzeStream performs streaming video analysis
func (s *AnalyzeService) AnalyzeStream(ctx context.Context, reqBody *models.AnalyzeRequest, callback func(*models.AnalyzeStreamResponse) error) error {
	// Set stream to true for streaming requests
	streamReq := *reqBody
	streamReq.Stream = true

	// Handle JSON request for video_id or video_url
	req, err := s.Client.NewRequest(ctx, "POST", "/analyze", &streamReq)
	if err != nil {
		return errors.NewRequestError("failed to create analyze stream request: " + err.Error())
	}

	resp, err := s.Client.DoRaw(req)
	if err != nil {
		return errors.NewServiceError("Analyze", "analyze stream request failed: "+err.Error())
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
			return errors.NewServiceError("Analyze", "callback error: "+err.Error())
		}

		// Stop processing if we hit a stream_end event
		if streamResp.EventType == "stream_end" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return errors.NewServiceError("Analyze", "error reading stream response: "+err.Error())
	}

	return nil
}

func (s *AnalyzeService) GenerateGist(ctx context.Context, reqBody *models.GenerateGistRequest) (*models.GenerateGistResponse, error) {
	req, err := s.Client.NewRequest(ctx, "POST", "/gist", reqBody)
	if err != nil {
		return nil, errors.NewRequestError("failed to create gist request: " + err.Error())
	}

	var response models.GenerateGistResponse
	_, err = s.Client.Do(req, &response)
	if err != nil {
		return nil, errors.NewServiceError("Analyze", "gist request failed: "+err.Error())
	}

	return &response, nil
}

func (s *AnalyzeService) GenerateSummary(ctx context.Context, reqBody *models.GenerateSummaryRequest) (*models.GenerateSummaryResponse, error) {
	req, err := s.Client.NewRequest(ctx, "POST", "/summarize", reqBody)
	if err != nil {
		return nil, errors.NewRequestError("failed to create summarize request: " + err.Error())
	}

	var response models.GenerateSummaryResponse
	_, err = s.Client.Do(req, &response)
	if err != nil {
		return nil, errors.NewServiceError("Analyze", "summarize request failed: "+err.Error())
	}

	return &response, nil
}
