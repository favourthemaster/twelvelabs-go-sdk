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

	fmt.Println("🚀 TwelveLabs Go SDK - Basic Usage Example")
	fmt.Println("==========================================")

	// 1. Create an Index
	fmt.Println("\n📁 Creating an index...")
	index, err := client.Indexes.Create(context.Background(), &models.IndexCreateRequest{
		IndexName: "example_index", // Replace with your desired index name
		Models: []models.Model{
			{
				ModelName:    "marengo2.7",
				ModelOptions: []string{"visual", "audio"},
			},
			{
				ModelName:    "pegasus1.2",
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
	indexes, err := client.Indexes.List(context.Background(), map[string]string{})
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
	task, err := client.Tasks.Create(context.Background(), &models.TasksCreateRequest{
		IndexID:  index.ID,
		VideoURL: "https://example.com/your-video-url.mp4", // Replace with your actual video URL
	})
	if err != nil {
		log.Printf("Error creating task: %v", err)
		return
	}
	fmt.Printf("✅ Task created: %s\n", task.ID)

	// 4. Wait for the task to complete before proceeding
	fmt.Printf("\n⏳ Waiting for task %s to complete...\n", task.ID)
	completedTask, err := client.Tasks.WaitForDone(context.Background(), task.ID, &wrappers.WaitForDoneOptions{
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
	tasks, err := client.Tasks.List(context.Background(), map[string]string{
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
	searchResults, err := client.Search.SearchByText(context.Background(),
		index.ID,
		"your search query here", // Replace with your actual search query
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
	embedding, err := client.Embed.CreateTextEmbedding(context.Background(),
		"Marengo-retrieval-2.7",
		"your text here", // Replace with your actual text
	)
	if err != nil {
		log.Printf("Error creating embedding: %v", err)
		return
	}
	fmt.Printf("✅ Text embedding created successfully\n")
	if embedding.TextEmbedding != nil && len(embedding.GetAllTextSegments()) > 0 {
		fmt.Printf("   First few values: [%.4f, %.4f, %.4f...]\n",
			embedding.GetEmbeddings()[0], embedding.GetEmbeddings()[1], embedding.GetEmbeddings()[2])
	}

	fmt.Println("\n🎉 Basic example completed successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("- Check out the advanced examples for bulk operations")
	fmt.Println("- See task waiting examples for production workflows")
	fmt.Println("- Explore video management examples")
}
