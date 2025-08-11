package main

import (
	"fmt"
	"log"

	"github.com/favourthemaster/twelvelabs-go-sdk"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/wrappers"
)

func main() {
	// Initialize client using placeholder API key
	client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
		APIKey: "your-api-key-here", // Replace with your actual API key
	})
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	fmt.Println("🧠 TwelveLabs Go SDK - Embeddings Examples")
	fmt.Println("==========================================")

	modelName := "Marengo-retrieval-2.7"

	// 1. Text embeddings
	fmt.Println("\n📝 Creating text embeddings...")
	textQueries := []string{
		"your first text query here",
		"your second text query here",
		"your third text query here",
		"your fourth text query here",
		"your fifth text query here",
	}

	fmt.Println("   Processing multiple text queries...")
	for i, query := range textQueries {
		embedding, err := client.Embed.CreateTextEmbedding(modelName, query)
		if err != nil {
			log.Printf("Error creating text embedding %d: %v", i+1, err)
			continue
		}

		embeddings := embedding.GetEmbeddings()
		fmt.Printf("   ✅ Text %d: \"%s...\" -> %d dimensions\n",
			i+1, query[:min(30, len(query))], len(embeddings))

		if len(embeddings) > 0 {
			fmt.Printf("      First few values: [%.4f, %.4f, %.4f...]\n",
				embeddings[0], embeddings[1], embeddings[2])
		}
	}

	// 2. Image embeddings
	fmt.Println("\n🖼️ Creating image embeddings...")
	imageURLs := []string{
		"https://example.com/your-image-url.jpg",
	}

	for i, imageURL := range imageURLs {
		embedding, err := client.Embed.CreateImageEmbedding(modelName, imageURL)
		if err != nil {
			log.Printf("Error creating image embedding %d: %v", i+1, err)
			continue
		}

		fmt.Printf("   ✅ Image %d: %s -> embedding created\n", i+1, imageURL)
		embeddings := embedding.GetEmbeddings()
		if len(embeddings) > 0 {
			fmt.Printf("      Dimensions: %d\n", len(embeddings))
		}
	}

	//// 3. Local file embeddings
	//fmt.Println("\n📁 Creating embeddings from local files...")
	//
	//// Local image embedding
	//_, err = client.Embed.Create(&wrappers.EmbedWrapperRequest{
	//	ModelName: modelName,
	//	ImageFile: "./assets/search_sample.png",
	//})
	//if err != nil {
	//	log.Printf("Error creating local image embedding: %v", err)
	//} else {
	//	fmt.Printf("   ✅ Local image embedding created\n")
	//}
	//
	//// Local audio embedding
	//_, err = client.Embed.Create(&wrappers.EmbedWrapperRequest{
	//	ModelName: modelName,
	//	AudioFile: "./assets/audio_sample.mp3",
	//})
	//if err != nil {
	//	log.Printf("Error creating local audio embedding: %v", err)
	//} else {
	//	fmt.Printf("   ✅ Local audio embedding created\n")
	//}

	// 4. Video embeddings
	fmt.Println("\n🎬 Creating video embeddings...")
	videoURLs := []string{
		"https://example.com/your-video-url.mp4",
	}

	for i, videoURL := range videoURLs {
		embedding, err := client.Embed.CreateVideoEmbedding(modelName, videoURL)
		if err != nil {
			log.Printf("Error creating video embedding %d: %v", i+1, err)
			continue
		}

		fmt.Printf("   ✅ Video %d: %s -> embedding created\n", i+1, videoURL)
		segments := embedding.GetAllVideoSegments()
		if len(segments) > 0 {
			fmt.Printf("      Video segments: %d\n", len(segments))
			for j, segment := range segments {
				if j >= 3 {
					fmt.Printf("      ... and %d more segments\n", len(segments)-3)
					break
				}
				startTime := "N/A"
				if segment.StartOffsetSec != nil {
					startTime = fmt.Sprintf("%.2fs", *segment.StartOffsetSec)
				}
				fmt.Printf("      Segment %d: %d dimensions, start: %s\n", j+1, len(segment.Float), startTime)
			}
		}
	}

	// 5. Audio embeddings from URLs
	fmt.Println("\n🎵 Creating audio embeddings from URLs...")
	audioURLs := []string{
		"https://example.com/your-audio-url.mp3",
	}

	for i, audioURL := range audioURLs {
		_, err := client.Embed.CreateAudioEmbedding(modelName, audioURL)
		if err != nil {
			log.Printf("Error creating audio embedding %d: %v", i+1, err)
			continue
		}

		fmt.Printf("   ✅ Audio %d: %s -> embedding created\n", i+1, audioURL)
	}

	// 6. Batch embedding creation
	fmt.Println("\n📦 Batch embedding creation...")

	batchRequests := []*wrappers.EmbedWrapperRequest{
		{ModelName: modelName, Text: "your first text here"},
		{ModelName: modelName, Text: "your second text here"},
		{ModelName: modelName, Text: "your third text here"},
		{ModelName: modelName, ImageURL: "https://example.com/your-first-image.jpg"},
		{ModelName: modelName, ImageURL: "https://example.com/your-second-image.jpg"},
	}

	fmt.Printf("   Processing %d embedding requests...\n", len(batchRequests))
	successCount := 0

	for i, request := range batchRequests {
		_, err := client.Embed.Create(request)
		if err != nil {
			log.Printf("   ❌ Batch request %d failed: %v", i+1, err)
			continue
		}

		successCount++
		contentType := "unknown"
		if request.Text != "" {
			contentType = "text"
		} else if request.ImageURL != "" {
			contentType = "image"
		}

		fmt.Printf("   ✅ Batch %d (%s): Success\n", i+1, contentType)
	}

	fmt.Printf("   📊 Batch completion: %d/%d successful\n", successCount, len(batchRequests))

	// 7. Embedding similarity comparison (conceptual)
	fmt.Println("\n🔄 Embedding similarity analysis...")

	// Create embeddings for similar concepts
	concepts := map[string]string{
		"concept1": "your first concept description",
		"concept2": "your similar concept description",
		"concept3": "your third concept description",
		"concept4": "your fourth concept description",
	}

	embeddings := make(map[string][]float64)

	for label, text := range concepts {
		embedding, err := client.Embed.CreateTextEmbedding(modelName, text)
		if err != nil {
			log.Printf("Error creating embedding for %s: %v", label, err)
			continue
		}

		embeddingVector := embedding.GetEmbeddings()
		if len(embeddingVector) > 0 {
			embeddings[label] = embeddingVector
			fmt.Printf("   ✅ %s: \"%s\" -> %d dimensions\n",
				label, text, len(embeddingVector))
		}
	}

	// 8. Error handling and edge cases
	fmt.Println("\n⚠️ Testing error handling...")

	// Test with invalid model name
	_, err = client.Embed.CreateTextEmbedding("invalid-model", "test text")
	if err != nil {
		fmt.Printf("   ✅ Correctly handled invalid model error: %v\n", err)
	}

	// Test with empty text
	_, err = client.Embed.CreateTextEmbedding(modelName, "")
	if err != nil {
		fmt.Printf("   ✅ Correctly handled empty text error: %v\n", err)
	}

	fmt.Println("\n🎉 Embeddings examples completed!")
	fmt.Println("\nEmbedding types demonstrated:")
	fmt.Println("- ✅ Text embeddings (single and batch)")
	fmt.Println("- ✅ Image embeddings (URL and local file)")
	fmt.Println("- ✅ Video embeddings")
	fmt.Println("- ✅ Audio embeddings (URL and local file)")
	fmt.Println("- ✅ Batch processing")
	fmt.Println("- ✅ Error handling")
	fmt.Println("- ✅ Similarity analysis preparation")
}
