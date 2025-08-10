package main

import (
	"fmt"
	"log"
	"os"

	"github.com/favourthemaster/twelvelabs-go-sdk"
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

	fmt.Println("🧠 TwelveLabs Go SDK - Embeddings Examples")
	fmt.Println("==========================================")

	modelName := "Marengo-retrieval-2.7"

	// 1. Text embeddings
	fmt.Println("\n📝 Creating text embeddings...")
	textQueries := []string{
		"A person walking in a beautiful park",
		"Sunset over mountain landscape",
		"Children playing in playground",
		"City traffic during rush hour",
		"Ocean waves crashing on beach",
	}

	fmt.Println("   Processing multiple text queries...")
	for i, query := range textQueries {
		embedding, err := client.Embed.CreateTextEmbedding(modelName, query)
		if err != nil {
			log.Printf("Error creating text embedding %d: %v", i+1, err)
			continue
		}

		fmt.Printf("   ✅ Text %d: \"%s...\" -> %d dimensions\n",
			i+1, query[:min(30, len(query))], len(embedding.Embeddings))

		if len(embedding.Embeddings) > 0 {
			fmt.Printf("      First few values: [%.4f, %.4f, %.4f...]\n",
				embedding.Embeddings[0], embedding.Embeddings[1], embedding.Embeddings[2])
		}
	}

	// 2. Image embeddings
	fmt.Println("\n🖼️ Creating image embeddings...")
	imageURLs := []string{
		"https://example.com/nature_scene.jpg",
		"https://example.com/city_skyline.jpg",
		"https://example.com/beach_sunset.jpg",
	}

	for i, imageURL := range imageURLs {
		embedding, err := client.Embed.CreateImageEmbedding(modelName, imageURL)
		if err != nil {
			log.Printf("Error creating image embedding %d: %v", i+1, err)
			continue
		}

		fmt.Printf("   ✅ Image %d: %s -> embedding created\n", i+1, imageURL)
		if embedding.ImageEmbedding != nil && len(embedding.ImageEmbedding.Embeddings) > 0 {
			fmt.Printf("      Dimensions: %d\n", len(embedding.ImageEmbedding.Embeddings))
		}
	}

	// 3. Local file embeddings
	fmt.Println("\n📁 Creating embeddings from local files...")

	// Local image embedding
	_, err = client.Embed.Create(&wrappers.EmbedWrapperRequest{
		ModelName: modelName,
		ImageFile: "./assets/search_sample.png",
	})
	if err != nil {
		log.Printf("Error creating local image embedding: %v", err)
	} else {
		fmt.Printf("   ✅ Local image embedding created\n")
	}

	// Local audio embedding
	_, err = client.Embed.Create(&wrappers.EmbedWrapperRequest{
		ModelName: modelName,
		AudioFile: "./assets/audio_sample.mp3",
	})
	if err != nil {
		log.Printf("Error creating local audio embedding: %v", err)
	} else {
		fmt.Printf("   ✅ Local audio embedding created\n")
	}

	// 4. Video embeddings
	fmt.Println("\n🎬 Creating video embeddings...")
	videoURLs := []string{
		"https://example.com/sample_video1.mp4",
		"https://example.com/sample_video2.mp4",
		"https://example.com/sample_video3.mp4",
	}

	for i, videoURL := range videoURLs {
		embedding, err := client.Embed.CreateVideoEmbedding(modelName, videoURL)
		if err != nil {
			log.Printf("Error creating video embedding %d: %v", i+1, err)
			continue
		}

		fmt.Printf("   ✅ Video %d: %s -> embedding created\n", i+1, videoURL)
		if embedding.VideoEmbedding != nil && len(embedding.VideoEmbedding.Embeddings) > 0 {
			fmt.Printf("      Video segments: %d\n", len(embedding.VideoEmbedding.Embeddings))
		}
	}

	// 5. Audio embeddings from URLs
	fmt.Println("\n🎵 Creating audio embeddings from URLs...")
	audioURLs := []string{
		"https://example.com/speech_sample.mp3",
		"https://example.com/music_sample.wav",
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
		{ModelName: modelName, Text: "Mountain hiking adventure"},
		{ModelName: modelName, Text: "Urban city exploration"},
		{ModelName: modelName, Text: "Peaceful beach relaxation"},
		{ModelName: modelName, ImageURL: "https://example.com/batch_image1.jpg"},
		{ModelName: modelName, ImageURL: "https://example.com/batch_image2.jpg"},
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
		"nature1": "Beautiful forest with tall trees",
		"nature2": "Lush green woodland area",
		"city1":   "Busy urban street with cars",
		"city2":   "Metropolitan downtown district",
	}

	embeddings := make(map[string][]float64)

	for label, text := range concepts {
		embedding, err := client.Embed.CreateTextEmbedding(modelName, text)
		if err != nil {
			log.Printf("Error creating embedding for %s: %v", label, err)
			continue
		}

		if len(embedding.Embeddings) > 0 {
			embeddings[label] = embedding.Embeddings
			fmt.Printf("   ✅ %s: \"%s\" -> %d dimensions\n",
				label, text, len(embedding.Embeddings))
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
