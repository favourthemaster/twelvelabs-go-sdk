package main

import (
	"fmt"
	"github.com/favourthemaster/twelvelabs-go-sdk"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/wrappers"
	"log"
	"os"
	"time"
)

func main() {
	// Initialize the TwelveLabs client
	//err := godotenv.Load()
	//if err != nil {
	//	log.Fatal("Error loading .env file")
	//} // Load environment variables from .env file if needed
	client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
		APIKey: os.Getenv("TWELVE_LABS_API_KEY"), // Or provide directly: "your-api-key"
	})
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	fmt.Println("🚀 TwelveLabs Go SDK - Basic Usage Example")
	fmt.Println("==========================================")

	// 1. Create an Index
	fmt.Println("\n📁 Creating an index...")
	index, err := client.Indexes.Create(&models.IndexCreateRequest{
		IndexName: "my-video-index5",
		Models: []models.Model{
			{
				ModelName:    "marengo2.7",
				ModelOptions: []string{"visual", "audio"},
			},
		},
	})
	if err != nil {
		log.Printf("Error creating index: %v", err)
		return
	}
	fmt.Printf("✅ Index created: %s (ID: %s)\n", index.IndexName, index.ID)

	// 2. List all indexes
	fmt.Println("\n📋 Listing all indexes...")
	indexes, err := client.Indexes.List(map[string]string{})
	if err != nil {
		log.Printf("Error listing indexes: %v", err)
		return
	}
	fmt.Printf("✅ Found %d indexes\n", len(indexes))
	for _, idx := range indexes {
		fmt.Printf("   - %s (ID: %s)\n", idx.IndexName, idx.ID)
	}

	// 3. Create a single task
	fmt.Println("\n🎬 Creating a video indexing task...")
	task, err := client.Tasks.Create(&models.TasksCreateRequest{
		IndexID:  index.ID,
		VideoURL: "https://www.example.com/sample-video.mp4",
	})
	if err != nil {
		log.Printf("Error creating task: %v", err)
		return
	}
	fmt.Printf("✅ Task created: %s (Status: %s)\n", task.ID, task.Status)

	// 4. Wait for the task to complete before proceeding
	fmt.Printf("\n⏳ Waiting for task %s to complete...\n", task.ID)
	completedTask, err := client.Tasks.WaitForDone(task.ID, &wrappers.WaitForDoneOptions{
		SleepInterval: 10 * time.Second,
		Callback: func(task *models.Task) error {
			fmt.Printf("   📊 Task status: %s\n", task.Status)
			return nil
		},
	})
	if err != nil {
		log.Printf("Error waiting for task completion: %v", err)
		return
	}

	if completedTask.Status != "ready" {
		log.Printf("Task failed with status: %s", completedTask.Status)
		return
	}
	fmt.Printf("✅ Task completed successfully: %s\n", completedTask.ID)

	// 5. List tasks
	fmt.Println("\n📝 Listing recent tasks...")
	tasks, err := client.Tasks.List(map[string]string{
		"index_id": index.ID,
	})
	if err != nil {
		log.Printf("Error listing tasks: %v", err)
		return
	}
	for _, t := range tasks {
		fmt.Printf("   - Task ID: %s, Status: %s, Created At: %s\n", t.ID, t.Status, t.CreatedAt)
	}
	fmt.Printf("✅ Found %d tasks for this index\n", len(tasks))

	// 6. Search with text query (basic)
	fmt.Println("\n🔍 Performing text search...")
	searchResults, err := client.Search.SearchByText(
		index.ID,
		"Rotation",
		[]string{"visual"},
	)
	if err != nil {
		log.Printf("Error performing search: %v", err)
		return
	}
	for _, result := range searchResults.Data {
		fmt.Printf("   - Video ID: %s, Score: %.4f, Start Time: %.2f, End Time: %.2f\n",
			result.VideoID, result.Score, result.Start, result.End)
	}

	fmt.Printf("✅ Search completed, found %d results\n", len(searchResults.Data))

	// 7. Create text embedding
	fmt.Println("\n🧠 Creating text embedding...")
	embedding, err := client.Embed.CreateTextEmbedding(
		"Marengo-retrieval-2.7",
		"Rotating Objects",
	)
	if err != nil {
		log.Printf("Error creating embedding: %v", err)
		return
	}
	fmt.Printf("✅ Text embedding created successfully\n")
	if embedding.TextEmbedding != nil && len(embedding.TextEmbedding.Embeddings) > 0 {
		fmt.Printf("   First few values: [%.4f, %.4f, %.4f...]\n",
			embedding.TextEmbedding.Embeddings[0], embedding.TextEmbedding.Embeddings[1], embedding.TextEmbedding.Embeddings[2])
	}

	fmt.Println("\n🎉 Basic example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Check out the advanced examples for bulk operations")
	fmt.Println("- See task waiting examples for production workflows")
	fmt.Println("- Explore video management examples")
}
