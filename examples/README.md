# TwelveLabs Go SDK Examples

This directory contains comprehensive examples demonstrating all features of the TwelveLabs Go SDK.

## Quick Start

1. Set your API key:
```bash
export TWELVE_LABS_API_KEY="your-api-key-here"
```

2. Replace `"your-index-id-here"` with your actual index ID in the examples.

3. Run any example:
```bash
go run basic_usage.go
go run advanced_usage.go
go run tasks_example.go
# ... etc
```

## Example Files

### 🌟 **basic_usage.go**
Perfect starting point demonstrating:
- Client initialization
- Index creation and listing
- Simple task creation
- Basic text search
- Text embeddings

### 🚀 **advanced_usage.go**
Advanced patterns including:
- Bulk task operations
- Task completion waiting with callbacks
- Advanced search patterns
- Video management within indexes
- Multiple embedding types
- Error handling

### 🎬 **tasks_example.go**
Complete task management:
- Single task creation (file & URL)
- Bulk task creation with mixed sources
- Task filtering and listing
- Progress tracking with callbacks
- Parallel task waiting
- Task detail retrieval

### 🔍 **search_example.go**
Comprehensive search capabilities:
- Text, image, and video-based search
- Multi-modal search combinations
- Local file search
- Paginated results
- Search result analysis

### 🧠 **embeddings_example.go**
All embedding types:
- Text, image, video, and audio embeddings
- Local file and URL processing
- Batch embedding creation
- Similarity analysis preparation
- Error handling

### 🎥 **videos_example.go**
Video management operations:
- Video listing with pagination
- Metadata updates (single and bulk)
- Video filtering and analysis
- Collection statistics
- Organization recommendations

## Asset Files

The `assets/` directory should contain sample files for testing:
- `example.mp4` - Sample video file
- `search_sample.png` - Sample image for search
- `audio_sample.mp3` - Sample audio file

## Usage Patterns

### Initialization
```go
client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
    APIKey: os.Getenv("TWELVE_LABS_API_KEY"),
})
```

### Environment Variables
- `TWELVE_LABS_API_KEY` - Your API key
- `TWELVELABS_BASE_URL` - Custom base URL (optional)

### Error Handling
All examples include proper error handling patterns and demonstrate how to handle various API error conditions.

## Next Steps

1. Start with `basic_usage.go` to understand the fundamentals
2. Move to `advanced_usage.go` for production patterns
3. Explore specific examples based on your use case
4. Adapt the patterns to your specific requirements

## Support

For more information, see the main SDK documentation and the TwelveLabs API reference.
