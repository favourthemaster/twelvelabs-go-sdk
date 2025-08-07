# TwelveLabs Go SDK

This is an unofficial Go SDK for the TwelveLabs Video Understanding Platform, designed to provide a convenient way to interact with the TwelveLabs API from Go applications.

## Features

- **Client Initialization**: Easily initialize the client with your API key.
- **Resource-Oriented Services**: Interact with Tasks, Indexes, Search, Embed, and Manage Videos services through dedicated methods.
- **Error Handling**: Custom error types for better handling of API errors.
- **File Uploads**: Support for video uploads.

## Installation

```bash
go get github.com/favourthemaster/twelvelabs-go-sdk
```

## Usage

### Initialize the Client

```go
package main

import (
	"fmt"
	"log"
	"os"
	"twelvelabs-go-sdk"
)

func main() {
	apiKey := os.Getenv("TWELVELABS_API_KEY")
	if apiKey == "" {
		log.Fatal("TWELVELABS_API_KEY environment variable not set")
	}

	client, err := twelvelabs.NewClient(&twelvelabs.ClientOptions{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	// Now you can use the client to interact with the TwelveLabs API
	// For example, listing tasks:
	tasks, err := client.Tasks.List()
	if err != nil {
		log.Fatalf("Error listing tasks: %v", err)
	}
	fmt.Printf("Found %d tasks\n", len(tasks))
}
```

### Create an Index

```go
package main

import (
	"fmt"
	"log"
	"os"
	"twelvelabs-go-sdk"
)

func main() {
	apiKey := os.Getenv("TWELVELABS_API_KEY")
	if apiKey == "" {
		log.Fatal("TWELVELABS_API_KEY environment variable not set")
	}

	client, err := twelvelabs.NewClient(&twelvelabs.ClientOptions{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	indexName := "My New Index"
	models := []twelvelabs.Model{
		{
			ModelName: "marengo2.7",
			ModelOptions: []string{"visual", "audio"},
		},
		{
			ModelName: "pegasus1.2",
			ModelOptions: []string{"visual", "audio"},
		},
	}

	createReq := &twelvelabs.IndexesCreateRequest{
		IndexName: indexName,
		Models:    models,
	}

	index, err := client.Indexes.Create(createReq)
	if err != nil {
		log.Fatalf("Error creating index: %v", err)
	}
	fmt.Printf("Created index: ID=%s Name=%s\n", index.ID, index.IndexName)
}
```

### Upload a Video

```go
package main

import (
	"fmt"
	"log"
	"os"
	"twelvelabs-go-sdk"
	"time"
)

func main() {
	apiKey := os.Getenv("TWELVELABS_API_KEY")
	if apiKey == "" {
		log.Fatal("TWELVELABS_API_KEY environment variable not set")
	}

	client, err := twelvelabs.NewClient(&twelvelabs.ClientOptions{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatalf("Error creating client: %v", err)
	}

	// Replace with your actual index ID and video file path
	indexID := "YOUR_INDEX_ID"
	videoFilePath := "./path/to/your/video.mp4"

	createTaskReq := &twelvelabs.TasksCreateRequest{
		IndexID:   indexID,
		VideoFile: videoFilePath,
	}

	task, err := client.Tasks.Create(createTaskReq)
	if err != nil {
		log.Fatalf("Error creating task: %v", err)
	}
	fmt.Printf("Created task: ID=%s Status=%s\n", task.ID, task.Status)

	// Wait for the task to complete
	completedTask, err := client.Tasks.WaitForDone(task.ID, 5*time.Second, func(t *twelvelabs.Task) {
		fmt.Printf("Task %s status: %s\n", t.ID, t.Status)
	})
	if err != nil {
		log.Fatalf("Error waiting for task: %v", err)
	}

	if completedTask.Status != "ready" {
		log.Fatalf("Video indexing failed with status: %s", completedTask.Status)
	}
	fmt.Printf("Video indexed successfully! Video ID: %s\n", completedTask.VideoID)
}
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

This project is licensed under the MIT License.


