package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/favourthemaster/twelvelabs-go-sdk"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/wrappers"
)

func main() {
	// Initialize client
	client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
		APIKey: os.Getenv("TWELVE_LABS_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	fmt.Println("🤖 TwelveLabs Go SDK - Analyze Examples")
	fmt.Println("======================================")

	// Example model and video (replace with your actual values)
	modelName := "pegasus-1"
	videoID := "your_video_id_here"
	videoURL := "https://example.com/sample_video.mp4"

	// 1. Basic video analysis with video ID
	fmt.Println("\n📹 Analyzing video by ID...")
	analyzeResp, err := client.Analyze.AnalyzeByVideoID(
		videoID,
		modelName,
		"Describe what happens in this video in detail.",
	)
	if err != nil {
		log.Printf("Video ID analysis failed: %v", err)
	} else {
		fmt.Printf("✅ Analysis completed!\n")
		fmt.Printf("Analysis ID: %s\n", analyzeResp.ID)
		fmt.Printf("Response: %s\n", analyzeResp.Data)
	}

	// 2. Video analysis with URL
	fmt.Println("\n🌐 Analyzing video by URL...")
	urlAnalyzeResp, err := client.Analyze.AnalyzeByVideoURL(
		videoURL,
		modelName,
		"What objects and people can you see in this video?",
	)
	if err != nil {
		log.Printf("Video URL analysis failed: %v", err)
	} else {
		fmt.Printf("✅ URL Analysis completed!\n")
		fmt.Printf("Analysis ID: %s\n", urlAnalyzeResp.ID)
		fmt.Printf("Response: %s\n", urlAnalyzeResp.Data)
	}

	// 3. Advanced analysis with custom parameters
	fmt.Println("\n⚙️ Advanced analysis with custom parameters...")
	advancedReq := &wrappers.AnalyzeWrapperRequest{
		VideoID:     videoID,
		ModelName:   modelName,
		Prompt:      "Provide a detailed summary of the key events in this video.",
		Temperature: 0.7,
		MaxTokens:   500,
		ModelParams: map[string]interface{}{
			"detail_level": "high",
		},
	}

	advancedResp, err := client.Analyze.Analyze(advancedReq)
	if err != nil {
		log.Printf("Advanced analysis failed: %v", err)
	} else {
		fmt.Printf("✅ Advanced analysis completed!\n")
		fmt.Printf("Analysis ID: %s\n", advancedResp.ID)
		fmt.Printf("Response: %s\n", advancedResp.Data)
	}

	// 4. Local file analysis
	fmt.Println("\n📁 Analyzing local video file...")
	localFileResp, err := client.Analyze.AnalyzeByVideoFile(
		"./assets/sample_video.mp4",
		modelName,
		"Analyze the content of this video and identify the main themes.",
	)
	if err != nil {
		log.Printf("Local file analysis failed: %v", err)
	} else {
		fmt.Printf("✅ Local file analysis completed!\n")
		fmt.Printf("Analysis ID: %s\n", localFileResp.ID)
		fmt.Printf("Response: %s\n", localFileResp.Data)
	}

	// 5. Streaming analysis
	fmt.Println("\n🔄 Streaming analysis...")
	streamReq := &wrappers.AnalyzeWrapperRequest{
		VideoID:   videoID,
		ModelName: modelName,
		Prompt:    "Provide a detailed analysis of this video, describing each scene.",
	}

	fmt.Println("Streaming response events:")
	var generationID string
	var accumulatedText strings.Builder

	err = client.Analyze.AnalyzeStream(streamReq, func(event *models.AnalyzeStreamResponse) error {
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
		"What are the main colors visible in this video?",
		"Identify any text or writing that appears in the video.",
		"Describe the setting and location of this video.",
		"What emotions or moods does this video convey?",
		"List any products or brands visible in the video.",
	}

	fmt.Printf("Processing %d different analysis prompts...\n", len(prompts))

	for i, prompt := range prompts {
		fmt.Printf("\n🔍 Analysis %d: %s\n", i+1, prompt)

		batchResp, err := client.Analyze.AnalyzeByVideoID(videoID, modelName, prompt)
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
	_, err = client.Analyze.AnalyzeByVideoID("invalid_video_id", modelName, "Test prompt")
	if err != nil {
		fmt.Printf("   ✅ Correctly handled invalid video ID error: %s\n",
			strings.Split(err.Error(), "\n")[0])
	}

	// Test with invalid model
	_, err = client.Analyze.AnalyzeByVideoID(videoID, "invalid_model", "Test prompt")
	if err != nil {
		fmt.Printf("   ✅ Correctly handled invalid model error: %s\n",
			strings.Split(err.Error(), "\n")[0])
	}

	// Test with empty prompt
	_, err = client.Analyze.AnalyzeByVideoID(videoID, modelName, "")
	if err != nil {
		fmt.Printf("   ✅ Correctly handled empty prompt error: %s\n", err.Error())
	}

	// 8. Analysis with different video sources comparison
	fmt.Println("\n🔄 Comparing different video source methods...")

	testPrompt := "Describe the first 30 seconds of this video."

	// Method 1: Video ID
	fmt.Println("   Method 1: Video ID")
	idResp, err := client.Analyze.AnalyzeByVideoID(videoID, modelName, testPrompt)
	if err != nil {
		fmt.Printf("   ❌ Video ID method failed: %v\n", err)
	} else {
		fmt.Printf("   ✅ Success (ID: %s)\n", idResp.ID)
	}

	// Method 2: Video URL
	fmt.Println("   Method 2: Video URL")
	urlResp, err := client.Analyze.AnalyzeByVideoURL(videoURL, modelName, testPrompt)
	if err != nil {
		fmt.Printf("   ❌ Video URL method failed: %v\n", err)
	} else {
		fmt.Printf("   ✅ Success (ID: %s)\n", urlResp.ID)
	}

	// Method 3: Generic wrapper
	fmt.Println("   Method 3: Generic wrapper")
	genericResp, err := client.Analyze.Analyze(&wrappers.AnalyzeWrapperRequest{
		VideoID:   videoID,
		ModelName: modelName,
		Prompt:    testPrompt,
	})
	if err != nil {
		fmt.Printf("   ❌ Generic wrapper method failed: %v\n", err)
	} else {
		fmt.Printf("   ✅ Success (ID: %s)\n", genericResp.ID)
	}

	fmt.Println("\n🎉 Analyze examples completed!")
	fmt.Println("\nAnalysis methods demonstrated:")
	fmt.Println("- ✅ Video ID analysis")
	fmt.Println("- ✅ Video URL analysis")
	fmt.Println("- ✅ Local file analysis")
	fmt.Println("- ✅ Streaming analysis")
	fmt.Println("- ✅ Advanced parameters")
	fmt.Println("- ✅ Batch processing")
	fmt.Println("- ✅ Error handling")
	fmt.Println("- ✅ Multiple input methods")
}
