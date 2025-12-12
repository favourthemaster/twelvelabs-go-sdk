package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/favourthemaster/twelvelabs-go-sdk"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
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

	fmt.Println("🔧 TwelveLabs Go SDK - Advanced Usage Examples")
	fmt.Println("===============================================")

	indexID := "your-index-id-here" // Replace with your actual index ID

	// 1. Bulk Task Creation
	fmt.Println("\n📦 Creating multiple tasks in bulk...")
	videoURLs := []string{
		"https://example.com/your-video-url.mp4",
	}

	tasks, err := client.Tasks.CreateBulk(context.Background(), &wrappers.CreateBulkRequest{
		IndexID:           indexID,
		VideoURLs:         videoURLs,
		EnableVideoStream: true,
	})
	if err != nil {
		log.Printf("Error creating bulk tasks: %v", err)
		return
	}
	fmt.Printf("✅ Created %d tasks successfully\n", len(tasks))

	// 2. Wait for Task Completion with Callback
	if len(tasks) > 0 {
		fmt.Println("\n⏳ Waiting for first task to complete...")

		completedTask, err := client.Tasks.WaitForDone(context.Background(), tasks[0].ID, &wrappers.WaitForDoneOptions{
			SleepInterval: 10 * time.Second,
			Callback: func(task *models.Task) error {
				fmt.Printf("   📊 Task %s status: %s\n", task.ID, task.Status)
				return nil
			},
		})
		if err != nil {
			log.Printf("Error waiting for task: %v", err)
		} else {
			fmt.Printf("✅ Task completed with status: %s\n", completedTask.Status)
		}
	}

	// 3. Advanced Search Patterns
	fmt.Println("\n🔍 Advanced Search Examples...")

	// Text search
	fmt.Println("   🔤 Text search:")
	textResults, err := client.Search.SearchByText(context.Background(), indexID, "your search query here", []string{"visual"})
	if err != nil {
		log.Printf("Error in text search: %v", err)
	} else {
		fmt.Printf("   ✅ Found %d text search results\n", len(textResults.Data))
	}

	// Image search
	fmt.Println("   🖼️  Image search:")
	imageResults, err := client.Search.SearchByImage(context.Background(), indexID, "https://example.com/your-image-url.jpg", []string{"visual"})
	if err != nil {
		log.Printf("Error in image search: %v", err)
	} else {
		fmt.Printf("   ✅ Found %d image search results\n", len(imageResults.Data))
	}

	// Advanced search with custom parameters
	fmt.Println("   ⚙️  Advanced search with custom parameters:")
	advancedResults, err := client.Search.Query(context.Background(), &models.SearchQueryRequest{
		IndexID:       indexID,
		QueryText:     "your advanced query here",
		SearchOptions: []string{"visual"},
	})
	if err != nil {
		log.Printf("Error in advanced search: %v", err)
	} else {
		fmt.Printf("   ✅ Found %d advanced search results\n", len(advancedResults.Data))
	}

	// 4. Video Management within Indexes
	fmt.Println("\n🎬 Video Management Examples...")

	// List videos in index
	videos, err := client.Indexes.Videos.List(context.Background(), indexID, map[string]string{
		"page_limit": "10",
	})
	if err != nil {
		log.Printf("Error listing videos: %v", err)
	} else {
		fmt.Printf("✅ Found %d videos in index\n", len(videos))

		if len(videos) > 0 {
			// Update video metadata
			fmt.Println("   📝 Updating video metadata...")
			updatedVideo, err := client.Indexes.Videos.Update(context.Background(), indexID, videos[0].ID, &models.VideoUpdateRequest{
				UserMetadata: map[string]string{
					"category":    "nature",
					"description": "Beautiful outdoor scene",
					"tags":        "nature,outdoor,peaceful",
				},
			})
			if err != nil {
				log.Printf("Error updating video: %v", err)
			} else {
				fmt.Printf("   ✅ Updated video: %s\n", updatedVideo.ID)
			}
		}
	}

	// 5. Multiple Embedding Types
	fmt.Println("\n🧠 Advanced Embedding Examples...")

	// Text embedding
	textEmbed, err := client.Embed.CreateTextEmbedding(context.Background(), "Marengo-retrieval-2.7", "your text content here")
	if err != nil {
		log.Printf("Error creating text embedding: %v", err)
	} else {
		fmt.Printf("✅ Text embedding created (dimension: %d)\n", len(textEmbed.GetEmbeddings()))
	}

	// Image embedding
	_, err = client.Embed.CreateImageEmbedding(context.Background(), "Marengo-retrieval-2.7", "https://example.com/your-image-url.jpg")
	if err != nil {
		log.Printf("Error creating image embedding: %v", err)
	} else {
		fmt.Printf("✅ Image embedding created\n")
	}

	// Generic embedding with custom request
	_, err = client.Embed.Create(context.Background(), &wrappers.EmbedWrapperRequest{
		ModelName: "Marengo-retrieval-2.7",
		Text:      "your custom text content here",
	})
	if err != nil {
		log.Printf("Error creating custom embedding: %v", err)
	} else {
		fmt.Printf("✅ Custom embedding created\n")
	}

	// 6. Error Handling Examples
	fmt.Println("\n⚠️  Error Handling Examples...")

	// Demonstrate proper error handling
	_, err = client.Indexes.Retrieve(context.Background(), "non-existent-index-id")
	if err != nil {
		fmt.Printf("✅ Properly caught error: %v\n", err)
	}

	fmt.Println("\n🎉 Advanced example completed successfully!")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("- ✅ Bulk task creation")
	fmt.Println("- ✅ Task completion waiting with callbacks")
	fmt.Println("- ✅ Advanced search patterns")
	fmt.Println("- ✅ Video management within indexes")
	fmt.Println("- ✅ Multiple embedding types")
	fmt.Println("- ✅ Proper error handling")
}
