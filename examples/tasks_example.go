package main

import (
	"fmt"
	"log"
	"os"
	"time"

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

	fmt.Println("🎬 TwelveLabs Go SDK - Tasks Management Example")
	fmt.Println("===============================================")

	indexID := "your-index-id-here" // Replace with your actual index ID

	// 1. Create a single task with local file
	fmt.Println("\n📁 Creating task with local video file...")
	singleTask, err := client.Tasks.Create(&models.TasksCreateRequest{
		IndexID:           indexID,
		VideoFile:         "./assets/example.mp4",
		EnableVideoStream: true,
		UserMetadata: map[string]string{
			"source":      "local_upload",
			"category":    "demo",
			"uploaded_by": "go_sdk_example",
		},
	})
	if err != nil {
		log.Printf("Error creating single task: %v", err)
	} else {
		fmt.Printf("✅ Single task created: %s\n", singleTask.ID)
	}

	// 2. Create a single task with URL
	fmt.Println("\n🌐 Creating task with video URL...")
	urlTask, err := client.Tasks.Create(&models.TasksCreateRequest{
		IndexID:             indexID,
		VideoURL:            "https://example.com/sample-video.mp4",
		VideoStartOffsetSec: 10,  // Start processing from 10 seconds
		VideoEndOffsetSec:   120, // Stop processing at 2 minutes
		EnableVideoStream:   true,
	})
	if err != nil {
		log.Printf("Error creating URL task: %v", err)
	} else {
		fmt.Printf("✅ URL task created: %s\n", urlTask.ID)
	}

	// 3. Bulk task creation with mixed sources
	fmt.Println("\n📦 Creating bulk tasks with mixed sources...")
	bulkTasks, err := client.Tasks.CreateBulk(&wrappers.CreateBulkRequest{
		IndexID: indexID,
		VideoURLs: []string{
			"https://example.com/video1.mp4",
			"https://example.com/video2.mp4",
			"https://example.com/video3.mp4",
		},
		VideoFiles: []string{
			"./assets/local_video1.mp4",
			"./assets/local_video2.mp4",
		},
		EnableVideoStream: true,
	})
	if err != nil {
		log.Printf("Error creating bulk tasks: %v", err)
	} else {
		fmt.Printf("✅ Bulk tasks created: %d tasks\n", len(bulkTasks))
		for i, task := range bulkTasks {
			fmt.Printf("   Task %d: %s (Status: %s)\n", i+1, task.ID, task.Status)
		}
	}

	// 4. List and filter tasks
	fmt.Println("\n📋 Listing and filtering tasks...")

	// List all tasks for the index
	allTasks, err := client.Tasks.List(map[string]string{
		"index_id": indexID,
	})
	if err != nil {
		log.Printf("Error listing tasks: %v", err)
	} else {
		fmt.Printf("✅ Total tasks in index: %d\n", len(allTasks))
	}

	// List only ready tasks
	readyTasks, err := client.Tasks.List(map[string]string{
		"index_id": indexID,
		"status":   "ready",
	})
	if err != nil {
		log.Printf("Error listing ready tasks: %v", err)
	} else {
		fmt.Printf("✅ Ready tasks: %d\n", len(readyTasks))
	}

	// 5. Wait for task completion with progress tracking
	if len(bulkTasks) > 0 {
		fmt.Printf("\n⏳ Waiting for task %s to complete...\n", bulkTasks[0].ID)

		startTime := time.Now()
		completedTask, err := client.Tasks.WaitForDone(bulkTasks[0].ID, &wrappers.WaitForDoneOptions{
			SleepInterval: 5 * time.Second,
			Callback: func(task *models.Task) error {
				elapsed := time.Since(startTime)
				fmt.Printf("   [%s] Task %s: %s\n",
					elapsed.Round(time.Second), task.ID, task.Status)

				// You can add custom logic here, like updating a progress bar
				// or sending notifications
				return nil
			},
		})

		if err != nil {
			log.Printf("Error waiting for task: %v", err)
		} else {
			totalTime := time.Since(startTime)
			fmt.Printf("✅ Task completed in %s with status: %s\n",
				totalTime.Round(time.Second), completedTask.Status)
		}
	}

	// 6. Wait for multiple tasks in parallel
	fmt.Println("\n🔄 Waiting for multiple tasks (first 3)...")
	if len(bulkTasks) >= 3 {
		tasksToWait := bulkTasks[:3]

		// Channel to collect results
		results := make(chan *models.Task, len(tasksToWait))
		errors := make(chan error, len(tasksToWait))

		// Start waiting for each task in a goroutine
		for _, task := range tasksToWait {
			go func(taskID string) {
				completedTask, err := client.Tasks.WaitForDone(taskID, &wrappers.WaitForDoneOptions{
					SleepInterval: 10 * time.Second,
					Callback: func(t *models.Task) error {
						fmt.Printf("   🔄 Task %s: %s\n", t.ID, t.Status)
						return nil
					},
				})

				if err != nil {
					errors <- err
				} else {
					results <- completedTask
				}
			}(task.ID)
		}

		// Collect results
		completed := 0
		for completed < len(tasksToWait) {
			select {
			case task := <-results:
				completed++
				fmt.Printf("✅ Task %s completed (%d/%d)\n", task.ID, completed, len(tasksToWait))
			case err := <-errors:
				completed++
				fmt.Printf("❌ Task failed: %v (%d/%d)\n", err, completed, len(tasksToWait))
			case <-time.After(5 * time.Minute): // Timeout after 5 minutes
				fmt.Println("⏰ Timeout waiting for tasks")
				return
			}
		}
	}

	// 7. Retrieve specific task details
	if len(bulkTasks) > 0 {
		fmt.Printf("\n🔍 Retrieving details for task %s...\n", bulkTasks[0].ID)
		taskDetails, err := client.Tasks.Retrieve(bulkTasks[0].ID)
		if err != nil {
			log.Printf("Error retrieving task: %v", err)
		} else {
			fmt.Printf("✅ Task Details:\n")
			fmt.Printf("   ID: %s\n", taskDetails.ID)
			fmt.Printf("   Status: %s\n", taskDetails.Status)
			fmt.Printf("   Video ID: %s\n", taskDetails.VideoID)
			fmt.Printf("   Created: %s\n", taskDetails.CreatedAt)
			fmt.Printf("   Updated: %s\n", taskDetails.UpdatedAt)
		}
	}

	fmt.Println("\n🎉 Task management example completed!")
	fmt.Println("\nFeatures demonstrated:")
	fmt.Println("- ✅ Single task creation (file & URL)")
	fmt.Println("- ✅ Bulk task creation")
	fmt.Println("- ✅ Task filtering and listing")
	fmt.Println("- ✅ Task completion waiting with callbacks")
	fmt.Println("- ✅ Parallel task waiting")
	fmt.Println("- ✅ Task detail retrieval")
}
