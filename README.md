# TwelveLabs Go SDK

A comprehensive Go SDK for the TwelveLabs API, providing easy access to video understanding, search, analysis, and embedding capabilities.

## Features

- 🎬 **Video Management**: Upload, index, and manage video content
- 🔍 **Advanced Search**: Text, image, and video-based search capabilities
- 🤖 **AI Analysis**: Video analysis, summarization, and content understanding
- 🧠 **Embeddings**: Generate embeddings for text, images, videos, and audio
- 📋 **Task Management**: Handle asynchronous video processing tasks
- 🎯 **Type Safe**: Full Go type definitions for all API responses

## Installation

```bash
go get github.com/favourthemaster/twelvelabs-go-sdk
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/favourthemaster/twelvelabs-go-sdk"
    "github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
)

func main() {
    // Initialize the client
    client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
        APIKey: "your-api-key-here", // Replace with your actual API key
    })
    if err != nil {
        log.Fatalf("Failed to initialize client: %v", err)
    }

    // Create an index
    index, err := client.Indexes.Create(context.Background(), &models.IndexCreateRequest{
        IndexName: "my-videos",
        Models: []models.Model{
            {
                ModelName:    "marengo2.7",
                ModelOptions: []string{"visual", "audio"},
            },
        },
    })
    if err != nil {
        log.Fatalf("Failed to create index: %v", err)
    }

    fmt.Printf("Index created: %s\n", index.ID)
}
```

## Configuration

Before using the SDK, you need to:

1. **Get an API Key**: Sign up at [TwelveLabs](https://twelvelabs.io) and obtain your API key
2. **Replace Placeholder Values**: Update all example code with your actual API keys and IDs

```go
client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
    APIKey: "your-actual-api-key-here", // Replace with your real API key
})
```

## Core Services

### 🗂️ Index Management

```go
// Create an index
index, err := client.Indexes.Create(context.Background(), &models.IndexCreateRequest{
    IndexName: "videos",
    Models: []models.Model{
        {
            ModelName:    "marengo2.7",
            ModelOptions: []string{"visual", "audio"},
        },
    },
})

// List all indexes
indexes, err := client.Indexes.List(context.Background(), map[string]string{})

// Get specific index
index, err := client.Indexes.Retrieve(context.Background(), "your-index-id")
```

### 🎬 Video Management

```go
// List videos in an index
videos, err := client.Indexes.Videos.List(context.Background(), "your-index-id", map[string]string{
    "page_limit": "10",
})

// Get video details
video, err := client.Indexes.Videos.Retrieve(context.Background(), "your-index-id", "your-video-id")

// Update video metadata
updatedVideo, err := client.Indexes.Videos.Update(context.Background(), "your-index-id", "your-video-id", &models.VideoUpdateRequest{
    UserMetadata: map[string]string{
        "title": "My Video Title",
        "category": "educational",
    },
})
```

### 📋 Task Management

```go
// Create a video indexing task
task, err := client.Tasks.Create(context.Background(), &models.TasksCreateRequest{
    IndexID:  "your-index-id",
    VideoURL: "https://example.com/your-video.mp4",
})

// Wait for task completion
completedTask, err := client.Tasks.WaitForDone(context.Background(), task.ID, &wrappers.WaitForDoneOptions{
    SleepInterval: 10 * time.Second,
    Callback: func(task *models.Task) error {
        fmt.Printf("Task status: %s\n", task.Status)
        return nil
    },
})
```

### 🔍 Search

```go
// Text search
results, err := client.Search.SearchByText(context.Background(),
    "your-index-id",
    "your search query",
    []string{"visual", "audio"},
)

// Image search
results, err := client.Search.SearchByImage(context.Background(),
    "your-index-id",
    "https://example.com/image.jpg",
    []string{"visual"},
)

// Advanced search
results, err := client.Search.Query(context.Background(), &models.SearchQueryRequest{
    IndexID:       "your-index-id",
    QueryText:     "your query",
    SearchOptions: []string{"visual", "audio"},
})
```

### 🤖 AI Analysis

```go
// Basic video analysis
response, err := client.Analyze.Analyze(context.Background(), &models.AnalyzeRequest{
    VideoID: "your-video-id",
    Prompt:  "your analysis prompt",
})

// Generate video summary
summary, err := client.Analyze.GenerateSummary(context.Background(), &models.GenerateSummaryRequest{
    VideoID: "your-video-id",
    Type:    "summary",
    Prompt:  "your summary prompt",
})

// Generate video gist
gist, err := client.Analyze.GenerateGist(context.Background(), &models.GenerateGistRequest{
    VideoID: "your-video-id",
    Types:   []string{"title", "topic", "hashtag"},
})
```

### 🧠 Embeddings

```go
// Text embedding
embedding, err := client.Embed.CreateTextEmbedding(context.Background(),
    "Marengo-retrieval-2.7",
    "your text content",
)

// Image embedding
embedding, err := client.Embed.CreateImageEmbedding(context.Background(),
    "Marengo-retrieval-2.7",
    "https://example.com/image.jpg",
)

// Video embedding
embedding, err := client.Embed.CreateVideoEmbedding(context.Background(),
    "Marengo-retrieval-2.7",
    "https://example.com/video.mp4",
)
```

## Examples

The `examples/` directory contains comprehensive examples for each service:

- **[basic_usage.go](examples/basic_usage.go)** - Getting started with the SDK
- **[advanced_usage.go](examples/advanced_usage.go)** - Advanced patterns and bulk operations
- **[videos_example.go](examples/videos_example.go)** - Video management and metadata operations
- **[search_example.go](examples/search_example.go)** - All types of search functionality
- **[analyze_example.go](examples/analyze_example.go)** - Video analysis and AI features
- **[embeddings_example.go](examples/embeddings_example.go)** - Embedding generation for all media types
- **[tasks_example.go](examples/tasks_example.go)** - Task management and monitoring

### Running Examples

1. Replace placeholder values in the example files:
   - `"your-api-key-here"` → Your actual TwelveLabs API key
   - `"your-index-id-here"` → Your actual index ID
   - `"your-video-id-here"` → Your actual video ID
   - URLs → Your actual media URLs

2. Run any example:
```bash
go run examples/basic_usage.go
```

## Error Handling

The SDK provides comprehensive error handling:

```go
client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
    APIKey: "your-api-key",
})
if err != nil {
    log.Fatalf("Failed to initialize client: %v", err)
}

result, err := client.Search.SearchByText("index-id", "query", []string{"visual"})
if err != nil {
    // Handle specific error types
    switch err.(type) {
    case *errors.AuthenticationError:
        log.Printf("Authentication failed: %v", err)
    case *errors.RateLimitError:
        log.Printf("Rate limit exceeded: %v", err)
    default:
        log.Printf("API error: %v", err)
    }
    return
}
```

## Models and Types

The SDK includes full Go type definitions for all API requests and responses. Key types include:

- `models.Index` - Index information and configuration
- `models.Video` - Video metadata and details
- `models.Task` - Task status and information
- `models.SearchResult` - Search result data
- `models.AnalyzeResponse` - Analysis results
- `models.EmbeddingResponse` - Embedding vectors

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

- 📖 [TwelveLabs Documentation](https://docs.twelvelabs.io/)
- 💬 [TwelveLabs Community](https://discord.gg/7KyJbgBJ)
- 🐛 [Report Issues](https://github.com/favourthemaster/twelvelabs-go-sdk/issues)

## Changelog

### Latest Changes
- ✅ Sanitized all examples for public GitHub usage
- ✅ Removed dependency on environment variables
- ✅ Added comprehensive placeholder value system
- ✅ Improved error handling and validation
- ✅ Enhanced documentation and examples
