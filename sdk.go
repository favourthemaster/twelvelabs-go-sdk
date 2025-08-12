/*
Package twelvelabs provides a Go SDK for the TwelveLabs API that matches the structure and functionality of the Node.js SDK.

This package offers a comprehensive wrapper around the TwelveLabs API with enhanced functionality including:
- Bulk operations for tasks
- Wait functionality for task completion
- Enhanced search capabilities with convenience methods
- Unified embedding interface for multiple content types
- Structured error handling

Main Components:
- TwelveLabs: Main client wrapper that provides access to all services
- TasksWrapper: Enhanced task operations with bulk creation and wait functionality
- IndexesWrapper: Index management with nested video operations
- SearchWrapper: Advanced search capabilities with convenience methods
- EmbedWrapper: Unified embedding interface for various content types

Usage Example:

	import "github.com/favourthemaster/twelvelabs-go-sdk"

	// Initialize the client
	client, err := twelvelabs.NewTwelveLabs(&twelvelabs.TwelveLabsOptions{
		APIKey: "your-api-key",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create an index
	index, err := client.Indexes.Create(&twelvelabs.IndexCreateRequest{
		IndexName: "my-index",
		Models: []twelvelabs.Model{
			{
				ModelName: "Marengo-retrieval-2.6",
				ModelOptions: []string{"visual", "conversation"},
			},
		},
	})

	// Create multiple tasks (bulk operation)
	tasks, err := client.Tasks.CreateBulk(&twelvelabs.CreateBulkRequest{
		IndexID: index.ID,
		VideoURLs: []string{
			"https://example.com/video1.mp4",
			"https://example.com/video2.mp4",
		},
	})

	// Wait for a task to complete
	for _, task := range tasks {
		completedTask, err := client.Tasks.WaitForDone(task.ID, &twelvelabs.WaitForDoneOptions{
			SleepInterval: 10 * time.Second,
			Callback: func(task *twelvelabs.Task) error {
				fmt.Printf("Task %s status: %s\n", task.ID, task.Status)
				return nil
			},
		})
		if err != nil {
			log.Printf("Error waiting for task %s: %v", task.ID, err)
			continue
		}
		fmt.Printf("Task %s completed with status: %s\n", completedTask.ID, completedTask.Status)
	}

	// Search with text query
	searchResults, err := client.Search.SearchByText(
		index.ID,
		"person running in the park",
		[]string{"visual"},
	)

	// Create embeddings
	embedResponse, err := client.Embed.CreateTextEmbedding(
		"Marengo-retrieval-2.6",
		"A person running in the park",
	)

Environment Variables:
- TWELVE_LABS_API_KEY: Your TwelveLabs API key
- TWELVELABS_BASE_URL: Custom base URL (optional, defaults to https://api.twelvelabs.io)

The SDK maintains compatibility with the existing API while providing enhanced functionality
that matches the Node.js SDK structure and capabilities.
*/
package twelvelabs

import (
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/client"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/wrappers"
)

// Re-export main types for convenience

// APIClient represents the main API client
type APIClient = client.Client

// Service wrapper aliases for easier access
type (
	Tasks   = wrappers.TasksWrapper
	Indexes = wrappers.IndexesWrapper
	Search  = wrappers.SearchWrapper
	Embed   = wrappers.EmbedWrapper
)

// Version information
const (
	SDKVersion = "1.0.0"
	APIVersion = "v1.3"
)
