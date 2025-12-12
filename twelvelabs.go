// Package twelvelabs provides a comprehensive Go SDK for the TwelveLabs API,
// enabling advanced video understanding, analysis, search, and embedding capabilities.
//
// The SDK supports:
// - Video analysis with custom prompts and streaming responses
// - Video summarization, chapter generation, and gist creation
// - Multi-modal search (text, image, video, audio)
// - Embedding generation for various media types
// - Asynchronous video upload and processing
// - Index and video management
//
// Example usage:
//
//	client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
//	    APIKey: "your-api-key",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Analyze a video
//	response, err := client.Analyze.Analyze(context.Background(), &models.AnalyzeRequest{
//	    VideoID: "video-id",
//	    Prompt:  "What objects are visible in this video?",
//	})
//
//	// Search for content
//	results, err := client.Search.SearchByText(context.Background(), "index-id", "person running", []string{"visual"})
//
//	// Generate embeddings
//	embedding, err := client.Embed.CreateTextEmbedding(context.Background(), "Marengo-retrieval-2.7", "sample text")
package twelvelabs

import (
	"os"
	"time"

	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/client"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/errors"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/wrappers"
)

// TwelveLabs is the main client that provides access to all TwelveLabs API services.
// It includes five core service areas:
//   - Analyze: Video analysis, summarization, and gist generation
//   - Search: Multi-modal video search capabilities
//   - Embed: Embedding generation for text, images, videos, and audio
//   - Tasks: Asynchronous video processing and upload management
//   - Indexes: Video index creation and management
type TwelveLabs struct {
	client  *client.Client
	options *Options
	Tasks   *wrappers.TasksWrapper
	Indexes *wrappers.IndexesWrapper
	Search  *wrappers.SearchWrapper
	Embed   *wrappers.EmbedWrapper
	Analyze *wrappers.AnalyzeWrapper
}

// Options represents configuration options for the TwelveLabs client.
// All fields are optional and will use sensible defaults or environment variables.
type Options struct {
	// APIKey is your TwelveLabs API key. If empty, uses TWELVE_LABS_API_KEY environment variable.
	APIKey string
	// BaseURL is the API base URL. If empty, uses TWELVELABS_BASE_URL environment variable or default.
	BaseURL string
	// Timeout is the HTTP client timeout. If zero, uses a default timeout.
	Timeout time.Duration
}

// NewTwelveLabs creates a new TwelveLabs client with the provided options.
// If options is nil, default configuration will be used.
//
// The client will automatically use environment variables:
//   - TWELVE_LABS_API_KEY: API key (required if not provided in options)
//   - TWELVELABS_BASE_URL: Custom base URL (optional)
//
// Example:
//
//	// Using environment variable
//	client, err := twelvelabs.NewTwelveLabs(nil)
//
//	// Using explicit options
//	client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
//	    APIKey: "your-api-key",
//	    BaseURL: "https://api.twelvelabs.io",
//	    Timeout: 30 * time.Second,
//	})
func NewTwelveLabs(options *Options) (*TwelveLabs, error) {
	if options == nil {
		options = &Options{}
	}

	// Use environment variable if API key is not provided
	apiKey := options.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("TWELVE_LABS_API_KEY")
	}

	if apiKey == "" {
		return nil, errors.NewUnauthorizedError("provide APIKey to initialize a client or set the TWELVE_LABS_API_KEY environment variable. You can see the API Key in the Dashboard page: https://dashboard.playground.io")
	}

	// Set default base URL if not provided
	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = os.Getenv("TWELVELABS_BASE_URL")
		if baseURL == "" {
			baseURL = client.DefaultBaseURL
		}
	}

	// Set default timeout if not provided
	timeout := options.Timeout

	clientOptions := &client.Options{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Timeout: timeout,
	}

	apiClient := client.NewClient(clientOptions)

	return &TwelveLabs{
		client:  apiClient,
		options: options,
		Tasks:   wrappers.NewTasksWrapper(apiClient.Tasks),
		Indexes: wrappers.NewIndexesWrapper(apiClient.Indexes),
		Search:  wrappers.NewSearchWrapper(apiClient.Search),
		Embed:   wrappers.NewEmbedWrapper(apiClient.Embed),
		Analyze: wrappers.NewAnalyzeWrapper(apiClient.Analyze),
	}, nil
}

// GetCustomAuthorizationHeaders returns the authorization headers used for API requests.
// This method is primarily for internal use but can be helpful for debugging.
func (t *TwelveLabs) GetCustomAuthorizationHeaders() map[string]string {
	return map[string]string{
		"x-api-key": t.options.APIKey,
	}
}
