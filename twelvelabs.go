package twelvelabs

import (
	"fmt"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/client"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/wrappers"
	"os"
	"time"
)

// TwelveLabs is the main client wrapper that provides access to all TwelveLabs API services
type TwelveLabs struct {
	client  *client.Client
	options *Options
	Tasks   *wrappers.TasksWrapper
	Indexes *wrappers.IndexesWrapper
	Search  *wrappers.SearchWrapper
	Embed   *wrappers.EmbedWrapper
	Analyze *wrappers.AnalyzeWrapper
}

// Options represents configuration options for the TwelveLabs client
type Options struct {
	APIKey  string
	BaseURL string
	Timeout time.Duration
}

// NewTwelveLabs creates a new TwelveLabs client with the provided options
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
		return nil, fmt.Errorf("provide APIKey to initialize a client or set the TWELVE_LABS_API_KEY environment variable. You can see the API Key in the Dashboard page: https://dashboard.playground.io")
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
	if timeout == 0 {
		timeout = client.DefaultTimeout
	}

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

// GetCustomAuthorizationHeaders returns the authorization headers
func (t *TwelveLabs) GetCustomAuthorizationHeaders() map[string]string {
	return map[string]string{
		"x-api-key": t.options.APIKey,
	}
}
