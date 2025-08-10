# TwelveLabs Go SDK

A Go client for the [TwelveLabs API](https://docs.twelvelabs.io/), enabling developers to interact with video understanding and search capabilities from Go applications.

## Features
- Video upload and management
- Video search (semantic, object, action, speech, text, etc.)
- Embeddings generation
- Index management
- Task management
- Idiomatic Go types and error handling

## Installation

```
go get github.com/favourthemaster/twelvelabs-go-sdk
```

## Usage

### 1. Import the SDK

```go
import (
    "github.com/favourthemaster/twelvelabs-go-sdk"
    "os"
)
```

### 2. Initialize the Client

```go
client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
    APIKey: os.Getenv("TWELVE_LABS_API_KEY"), // or leave blank to use env var
})
if err != nil {
    // handle error
}
```

> **Note:** If you do not provide an API key, the SDK will use the `TWELVE_LABS_API_KEY` environment variable. You can also set a custom base URL with the `BaseURL` field in `Options`.

### 3. Search Example

```go
import (
    "github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
)

req := models.SearchRequest{
    Query: "A woman vlogs about her summer day",
    IndexID: "your_index_id",
}
resp, err := client.Search.Search(req)
if err != nil {
    // handle error
}
for _, result := range resp.Data {
    fmt.Println(result.Transcription)
}
```

### 4. Video Upload Example

```go
videoID, err := client.ManageVideos.Upload("/path/to/video.mp4")
if err != nil {
    // handle error
}
fmt.Println("Uploaded video ID:", videoID)
```

### 5. Embeddings Example

```go
embedding, err := client.Embed.GetEmbedding("your_text_or_video_id")
if err != nil {
    // handle error
}
fmt.Println(embedding)
```

## Examples
See the [`examples/`](./examples/) directory for more usage patterns:
- `basic_usage.go`: Basic search and video upload
- `search_example.go`: Advanced search queries
- `embeddings_example.go`: Embedding generation
- `tasks_example.go`: Task management
- `videos_example.go`: Video management

## Error Handling
All errors are wrapped in the `pkg/errors` package for consistent error handling.

## Types
All request and response types are defined in `pkg/models/types.go`.

## Contributing
Pull requests are welcome! Please open issues for bugs or feature requests.

## License
MIT
