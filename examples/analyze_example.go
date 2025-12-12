package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/favourthemaster/twelvelabs-go-sdk"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
)

func main() {
	// Initialize client using placeholder API key
	client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
		APIKey: "your-api-key-here", // Replace with your actual API key
	})
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	fmt.Println("🤖 TwelveLabs Go SDK - Analyze Examples")
	fmt.Println("======================================")

	videoID := "your-video-id-here" // Replace with your actual video ID

	// 1. Basic video analysis with video ID
	fmt.Println("\n📹 Analyzing video by ID...")
	analyzeResp, err := client.Analyze.Analyze(context.Background(), &models.AnalyzeRequest{
		VideoID: videoID,
		Prompt:  "your analysis prompt here",
		Stream:  false,
	})
	if err != nil {
		log.Printf("Video ID analysis failed: %v", err)
	} else {
		fmt.Printf("✅ Analysis completed!\n")
		fmt.Printf("Analysis ID: %s\n", analyzeResp.ID)
		fmt.Printf("Response: %s\n", analyzeResp.Data)
	}

	gistResponse, err := client.Analyze.GenerateGist(context.Background(), &models.GenerateGistRequest{
		VideoID: videoID,
		Types: []string{
			"title",
			"topic",
			"hashtag",
		},
	})
	if err != nil {
		log.Printf("Gist generation failed: %v", err)
	} else {
		fmt.Printf("✅ Gist generation completed!\n")
		fmt.Printf("Gist ID: %s\n", gistResponse.ID)
		fmt.Printf("Title: %s\n", gistResponse.Title)
		fmt.Printf("Topics: %s\n", gistResponse.Topics)
		fmt.Printf("Hashtags: %v\n", gistResponse.Hashtags)
	}

	summary, err := client.Analyze.GenerateSummary(context.Background(), &models.GenerateSummaryRequest{
		VideoID: videoID,
		Type:    "summary",
		Prompt:  "your summary prompt here",
	})
	if err != nil {
		log.Printf("Summary generation failed: %v", err)
	} else {
		fmt.Printf("✅ Summary generation completed!\n")
		fmt.Printf("Summary ID: %s\n", summary.ID)
		fmt.Printf("Summary: %s\n", summary.Summary)
	}

	chapters, err := client.Analyze.GenerateSummary(context.Background(), &models.GenerateSummaryRequest{
		VideoID: videoID,
		Type:    "chapter",
		Prompt:  "your chapter generation prompt here",
	})
	if err != nil {
		log.Printf("Summary generation failed: %v", err)
	} else {
		fmt.Printf("✅ Summary generation completed!\n")
		fmt.Printf("Chapter ID: %s\n", chapters.ID)
		fmt.Printf("Chapters: %s\n", chapters.Chapters)
	}

	highlights, err := client.Analyze.GenerateSummary(context.Background(), &models.GenerateSummaryRequest{
		VideoID: videoID,
		Type:    "highlight",
		Prompt:  "your highlight identification prompt here",
	})
	if err != nil {
		log.Printf("Summary generation failed: %v", err)
	} else {
		fmt.Printf("✅ Summary generation completed!\n")
		fmt.Printf("Highlight ID: %s\n", highlights.ID)
		fmt.Printf("Hightlights: %s\n", highlights.Highlights)
	}

	// 3. Advanced analysis with custom parameters
	advancedResp, err := client.Analyze.Analyze(context.Background(), &models.AnalyzeRequest{
		VideoID:     videoID,
		Prompt:      "your detailed analysis prompt here",
		Temperature: 0.7,
	})
	if err != nil {
		log.Printf("Advanced analysis failed: %v", err)
	} else {
		fmt.Printf("✅ Advanced analysis completed!\n")
		fmt.Printf("Analysis ID: %s\n", advancedResp.ID)
		fmt.Printf("Response: %s\n", advancedResp.Data)
	}

	// 5. Streaming analysis
	fmt.Println("\n🔄 Streaming analysis...")

	fmt.Println("Streaming response events:")
	var generationID string
	var accumulatedText strings.Builder

	err = client.Analyze.AnalyzeStream(context.Background(), &models.AnalyzeRequest{
		VideoID: videoID,
		Prompt:  "your streaming analysis prompt here",
	}, func(event *models.AnalyzeStreamResponse) error {
		switch event.EventType {
		case "stream_start":
			if event.Metadata != nil {
				generationID = event.Metadata.GenerationID
				fmt.Printf("   🚀 Stream started (ID: %s)\n", generationID)
			}
		case "text_generation":
			fmt.Printf("   📝 Text: %s\n", event.Text)
			accumulatedText.WriteString(event.Text)
		case "stream_end":
			if event.Metadata != nil {
				fmt.Printf("   ✅ Stream ended (ID: %s)\n", event.Metadata.GenerationID)
				if event.Metadata.Usage != nil {
					fmt.Printf("   📊 Output tokens: %d\n", event.Metadata.Usage.OutputTokens)
				}
			}
		default:
			fmt.Printf("   ℹ️ Unknown event type: %s\n", event.EventType)
		}
		return nil
	})

	if err != nil {
		log.Printf("Streaming analysis failed: %v", err)
	} else {
		fmt.Printf("✅ Streaming analysis completed!\n")
		fmt.Printf("Full response: %s\n", accumulatedText.String())
	}

	// 6. Batch analysis with different prompts
	fmt.Println("\n📦 Batch analysis with different prompts...")

	prompts := []string{
		"your first analysis prompt here",
		"your second analysis prompt here",
		"your third analysis prompt here",
		"your fourth analysis prompt here",
		"your fifth analysis prompt here",
	}

	fmt.Printf("Processing %d different analysis prompts...\n", len(prompts))

	for i, prompt := range prompts {
		fmt.Printf("\n🔍 Analysis %d: %s\n", i+1, prompt)

		batchResp, err := client.Analyze.Analyze(context.Background(), &models.AnalyzeRequest{
			VideoID: videoID,
			Prompt:  prompt,
			Stream:  false,
		})
		if err != nil {
			log.Printf("   ❌ Failed: %v", err)
			continue
		}

		response := batchResp.Data
		// Truncate long responses for display
		if len(response) > 150 {
			response = response[:150] + "..."
		}
		fmt.Printf("   ✅ %s\n", response)
	}

	// 7. Error handling examples
	fmt.Println("\n⚠️ Testing error handling...")

	// Test with invalid video ID
	_, err = client.Analyze.Analyze(context.Background(), &models.AnalyzeRequest{
		VideoID: "invalid_video_id",
		Prompt:  "your test prompt here",
	})
	if err != nil {
		fmt.Printf("   ✅ Correctly handled invalid video ID error: %s\n",
			strings.Split(err.Error(), "\n")[0])
	}

	// Test with empty prompt
	_, err = client.Analyze.Analyze(context.Background(), &models.AnalyzeRequest{
		VideoID: videoID,
		Prompt:  "",
	})
	if err != nil {
		fmt.Printf("   ✅ Correctly handled empty prompt error: %s\n", err.Error())
	}

	fmt.Println("\n🎉 Analyze examples completed!")
	fmt.Println("\nAnalysis methods demonstrated:")
	fmt.Println("- ✅ Video ID analysis")
	fmt.Println("- ✅ Streaming analysis")
	fmt.Println("- ✅ Advanced parameters")
	fmt.Println("- ✅ Batch processing")
	fmt.Println("- ✅ Error handling")
}
