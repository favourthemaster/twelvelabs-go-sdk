# TwelveLabs Go SDK Examples

This directory contains comprehensive examples demonstrating all features of the TwelveLabs Go SDK. All examples have been sanitized for public GitHub usage with placeholder values.

## 🚀 Getting Started

### Prerequisites

1. **TwelveLabs API Key**: Sign up at [TwelveLabs](https://twelvelabs.io) to get your API key
2. **Go 1.19+**: Make sure you have Go installed

### Setup

Before running any examples, you need to replace placeholder values with your actual data:

```go
// Replace these placeholders in all example files:
APIKey: "your-api-key-here"     // → Your actual TwelveLabs API key
indexID := "your-index-id-here" // → Your actual index ID  
videoID := "your-video-id-here" // → Your actual video ID
"https://example.com/..."       // → Your actual media URLs
"your search query here"        // → Your actual search queries
```

## 📁 Example Files

### 1. **basic_usage.go** - Getting Started
**Purpose**: Introduction to core SDK functionality
**What it demonstrates**:
- Client initialization
- Creating indexes
- Uploading videos
- Basic search
- Text embeddings

```bash
go run basic_usage.go
```

### 2. **advanced_usage.go** - Advanced Patterns
**Purpose**: Advanced usage patterns and bulk operations
**What it demonstrates**:
- Bulk task creation
- Task completion callbacks
- Advanced search patterns
- Video management within indexes
- Multiple embedding types
- Error handling strategies

```bash
go run advanced_usage.go
```

### 3. **videos_example.go** - Video Management
**Purpose**: Comprehensive video management operations
**What it demonstrates**:
- Listing videos with pagination
- Retrieving detailed video information
- Updating video metadata (single and bulk)
- Filtering and analyzing video collections
- Video statistics and insights
- Safe deletion practices
- Organization recommendations

```bash
go run videos_example.go
```

### 4. **search_example.go** - Search Capabilities
**Purpose**: All types of search functionality
**What it demonstrates**:
- Text-based search
- Image-based search
- Video-based search
- Multi-modal search
- Local file search (commented)
- Advanced search with custom parameters
- Paginated search results
- Search result analysis

```bash
go run search_example.go
```

### 5. **analyze_example.go** - AI Analysis
**Purpose**: Video analysis and AI-powered features
**What it demonstrates**:
- Basic video analysis
- Video gist generation
- Summary generation
- Chapter generation
- Highlight identification
- Streaming analysis
- Batch analysis with multiple prompts
- Error handling for analysis

```bash
go run analyze_example.go
```

### 6. **embeddings_example.go** - Embeddings
**Purpose**: Embedding generation for all media types
**What it demonstrates**:
- Text embeddings (single and batch)
- Image embeddings from URLs
- Video embeddings
- Audio embeddings
- Local file embeddings (commented)
- Batch embedding creation
- Embedding similarity analysis
- Error handling for embeddings

```bash
go run embeddings_example.go
```

### 7. **tasks_example.go** - Task Management
**Purpose**: Asynchronous task management and monitoring
**What it demonstrates**:
- Single task creation (URL and file)
- Bulk task creation
- Task filtering and listing
- Task completion waiting with callbacks
- Parallel task waiting
- Task detail retrieval
- Progress tracking

```bash
go run tasks_example.go
```

## 🛠️ Setup Instructions

### Step 1: Replace Placeholder Values

Each example file contains placeholder values that need to be replaced:

```go
// In every example file, replace:

// 1. API Key
client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
    APIKey: "your-api-key-here", // ← Replace with your actual API key
})

// 2. Index ID
indexID := "your-index-id-here" // ← Replace with your actual index ID

// 3. Video ID (in analyze_example.go)
videoID := "your-video-id-here" // ← Replace with your actual video ID

// 4. Media URLs
VideoURL: "https://example.com/your-video-url.mp4", // ← Replace with actual URL
ImageURL: "https://example.com/your-image-url.jpg", // ← Replace with actual URL

// 5. Search Queries and Prompts
QueryText: "your search query here", // ← Replace with actual query
Prompt: "your analysis prompt here", // ← Replace with actual prompt
```

### Step 2: Remove Validation Checks (Optional)

Some examples include validation to prevent running with placeholder values:

```go
// You can remove or modify these checks after replacing placeholders:
if indexID == "your-index-id-here" {
    log.Fatal("Please replace 'your-index-id-here' with your actual index ID")
}
```

### Step 3: Run Examples

```bash
# Navigate to the examples directory
cd examples

# Run any example
go run basic_usage.go
go run advanced_usage.go
# ... etc
```

## 📋 Example Dependencies

All examples use only standard library and SDK dependencies:

```go
import (
    "fmt"
    "log"
    "time" // for some examples
    "strings" // for some examples
    
    "github.com/favourthemaster/twelvelabs-go-sdk"
    "github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
    "github.com/favourthemaster/twelvelabs-go-sdk/pkg/wrappers"
)
```

**Note**: We removed the `godotenv` dependency to simplify setup and make examples GitHub-safe.

## 🔧 Customization

### Adding Your Own Examples

1. Copy an existing example as a template
2. Replace placeholder values with your data
3. Modify the functionality as needed
4. Follow the same error handling patterns

### Local File Examples

Some examples include commented code for local file operations:

```go
// Uncomment and modify these sections if you have local files:
// VideoFile: "./assets/example.mp4",
// ImageFile: "./assets/search_sample.png",
// AudioFile: "./assets/audio_sample.mp3",
```

## ⚠️ Important Notes

### Security
- **Never commit actual API keys** to version control
- **Replace all placeholder values** before running
- **Use environment variables** in production applications

### Rate Limits
- Examples include proper error handling for rate limits
- Add delays between requests if needed
- Monitor your API usage

### Data Usage
- Examples use placeholder URLs that won't work
- Replace with your actual media URLs
- Ensure you have rights to the media you're processing

## 🎯 Common Patterns

### Error Handling
```go
if err != nil {
    log.Printf("Error: %v", err)
    return
}
```

### Task Waiting
```go
completedTask, err := client.Tasks.WaitForDone(task.ID, &wrappers.WaitForDoneOptions{
    SleepInterval: 10 * time.Second,
    Callback: func(task *models.Task) error {
        fmt.Printf("Task status: %s\n", task.Status)
        return nil
    },
})
```

### Pagination
```go
videos, err := client.Indexes.Videos.List(indexID, map[string]string{
    "page_limit": "10",
    "sort_by":    "created_at",
    "sort_order": "desc",
})
```

## 🆘 Troubleshooting

### Common Issues

1. **"Please replace placeholder values"** error
   - Replace all `"your-*-here"` values with actual data

2. **Authentication errors**
   - Verify your API key is correct
   - Check if your API key has necessary permissions

3. **Index/Video not found**
   - Ensure your index ID and video ID exist
   - Check that resources are in the correct region

4. **Rate limit errors**
   - Add delays between requests
   - Implement exponential backoff

### Getting Help

- 📖 [SDK Documentation](../README.md)
- 📖 [TwelveLabs API Docs](https://docs.twelvelabs.io/)
- 🐛 [Report Issues](https://github.com/favourthemaster/twelvelabs-go-sdk/issues)

## 🚀 Next Steps

After running the examples:

1. **Explore the SDK**: Check out the main [README](../README.md)
2. **Build your application**: Use these examples as starting points
3. **Contribute**: Submit improvements or additional examples
4. **Share feedback**: Let us know how we can improve the SDK
