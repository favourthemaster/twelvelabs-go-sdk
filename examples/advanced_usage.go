package main

import (
	"fmt"
	"log"
	"time"

	"github.com/favourthemaster/twelvelabs-go-sdk"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/wrappers"
)

func main() {
	// Initialize the TwelveLabs client
	client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
		BaseURL: "https://api.twelvelabs.io/v1.3", // Optional, defaults to "https://api.twelvelabs.io"
		APIKey:  "tlk_01TR6NQ15T8GSK2P5MJBM0E63GM5",
		//APIKey: os.Getenv("TWELVE_LABS_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	fmt.Println("🚀 TwelveLabs Go SDK - Advanced Usage Example")
	fmt.Println("============================================")

	// Replace with your actual index ID
	indexID := "6897e2123e195789d467560b"

	// 1. Bulk Task Creation
	fmt.Println("\n📦 Creating multiple tasks in bulk...")
	videoURLs := []string{
		"https://res.cloudinary.com/dkasavogz/video/upload/v1754476322/d80fc467-b2ef-4e92-a928-ad8be42aef10/veadsunro41crxu4qsjy.mp4",
		//"https://res.cloudinary.com/dkasavogz/video/upload/v1754476322/d80fc467-b2ef-4e92-a928-ad8be42aef10/veadsunro41crxu4qsjy.mp4",
		//"https://res.cloudinary.com/dkasavogz/video/upload/v1754476322/d80fc467-b2ef-4e92-a928-ad8be42aef10/veadsunro41crxu4qsjy.mp4",
	}

	tasks, err := client.Tasks.CreateBulk(&wrappers.CreateBulkRequest{
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

		completedTask, err := client.Tasks.WaitForDone(tasks[0].ID, &wrappers.WaitForDoneOptions{
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
	textResults, err := client.Search.SearchByText(indexID, "Unreal Engine", []string{"visual"})
	if err != nil {
		log.Printf("Error in text search: %v", err)
	} else {
		fmt.Printf("   ✅ Found %d text search results\n", len(textResults.Data))
	}

	// Image search
	fmt.Println("   🖼️  Image search:")
	imageResults, err := client.Search.SearchByImage(indexID, "https://download.logo.wine/logo/Unreal_Engine/Unreal_Engine-Logo.wine.png", []string{"visual"})
	if err != nil {
		log.Printf("Error in image search: %v", err)
	} else {
		fmt.Printf("   ✅ Found %d image search results\n", len(imageResults.Data))
	}

	// Advanced search with custom parameters
	fmt.Println("   ⚙️  Advanced search with custom parameters:")
	advancedResults, err := client.Search.Query(&models.SearchQueryRequest{
		IndexID:       indexID,
		QueryText:     "Torus",
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
	videos, err := client.Indexes.Videos.List(indexID, map[string]string{
		"page_limit": "10",
	})
	if err != nil {
		log.Printf("Error listing videos: %v", err)
	} else {
		fmt.Printf("✅ Found %d videos in index\n", len(videos))

		if len(videos) > 0 {
			// Update video metadata
			fmt.Println("   📝 Updating video metadata...")
			updatedVideo, err := client.Indexes.Videos.Update(indexID, videos[0].ID, &models.VideoUpdateRequest{
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
	textEmbed, err := client.Embed.CreateTextEmbedding("Marengo-retrieval-2.7", "A serene mountain landscape")
	if err != nil {
		log.Printf("Error creating text embedding: %v", err)
	} else {
		fmt.Printf("✅ Text embedding created (dimension: %d)\n", len(textEmbed.Embeddings))
	}

	// Image embedding
	_, err = client.Embed.CreateImageEmbedding("Marengo-retrieval-2.7", "https://download.logo.wine/logo/Unreal_Engine/Unreal_Engine-Logo.wine.png")
	if err != nil {
		log.Printf("Error creating image embedding: %v", err)
	} else {
		fmt.Printf("✅ Image embedding created\n")
	}

	// Generic embedding with custom request
	_, err = client.Embed.Create(&wrappers.EmbedWrapperRequest{
		ModelName: "Marengo-retrieval-2.7",
		Text:      "Custom embedding request with specific parameters",
	})
	if err != nil {
		log.Printf("Error creating custom embedding: %v", err)
	} else {
		fmt.Printf("✅ Custom embedding created\n")
	}

	// 6. Error Handling Examples
	fmt.Println("\n⚠️  Error Handling Examples...")

	// Demonstrate proper error handling
	_, err = client.Indexes.Retrieve("non-existent-index-id")
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
