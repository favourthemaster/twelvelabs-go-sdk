package twelvelabs

import (
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the base URL of the TwelveLabs API v1.3.
	DefaultBaseURL = "https://api.twelvelabs.io/v1.3"
	// APIVersion is the API version this SDK targets.
	APIVersion = "v1.3"
	// SDKVersion is the version of this SDK.
	SDKVersion = "0.3.0"

	defaultMaxRetries = 3
	defaultUserAgent  = "twelvelabs-go-sdk/" + SDKVersion
)

// NoRetries disables automatic retries when passed as Options.MaxRetries.
const NoRetries = -1

// Options configures a Client.
type Options struct {
	// APIKey is the TwelveLabs API key. When empty, the client reads
	// TWELVE_LABS_API_KEY from the environment.
	APIKey string

	// BaseURL overrides the API base URL. Useful for tests and proxies.
	// Defaults to TWELVELABS_BASE_URL, then DefaultBaseURL.
	BaseURL string

	// HTTPClient is used for all requests. When nil, a client without an
	// overall timeout is used so that long-running uploads and streams are not
	// cut off; pass a configured client or use contexts with deadlines.
	HTTPClient *http.Client

	// MaxRetries is the number of retries after the first attempt.
	// Zero uses the default of 3. Use NoRetries to disable retries.
	MaxRetries int

	// UserAgent overrides the User-Agent header.
	UserAgent string
}

// Client is a TwelveLabs API client. It is safe for concurrent use.
type Client struct {
	transport *transport

	// Assets uploads and manages video, audio, image, and document assets.
	Assets *AssetsService
	// Indexes creates and manages indexes.
	Indexes *IndexesService
	// IndexedAssets makes assets searchable within an index.
	IndexedAssets *IndexedAssetsService
	// Search finds moments in indexed content.
	Search *SearchService
	// Analyze generates text and structured segments from video and images.
	Analyze *AnalyzeService
	// Embed creates multimodal embeddings with the Embed API v2.
	Embed *EmbedService
	// Entities identifies specific people in video.
	Entities *EntitiesService
	// Tasks manages legacy video indexing tasks. Deprecated by TwelveLabs.
	Tasks *TasksService
}

// NewClient creates a new TwelveLabs API client.
func NewClient(options Options) (*Client, error) {
	apiKey := strings.TrimSpace(options.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("TWELVE_LABS_API_KEY"))
	}
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	baseURL := strings.TrimSpace(options.BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("TWELVELABS_BASE_URL"))
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	maxRetries := options.MaxRetries
	switch {
	case maxRetries == NoRetries:
		maxRetries = 0
	case maxRetries == 0:
		maxRetries = defaultMaxRetries
	}

	userAgent := options.UserAgent
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	client := &Client{
		transport: &transport{
			apiKey:     apiKey,
			baseURL:    baseURL,
			httpClient: httpClient,
			maxRetries: maxRetries,
			userAgent:  userAgent,
		},
	}

	client.Assets = &AssetsService{client: client}
	client.Indexes = &IndexesService{client: client}
	client.IndexedAssets = &IndexedAssetsService{client: client}
	client.Search = &SearchService{client: client}
	client.Analyze = &AnalyzeService{client: client}
	client.Embed = &EmbedService{client: client}
	client.Entities = &EntitiesService{client: client}
	client.Tasks = &TasksService{client: client}

	return client, nil
}

// BaseURL returns the API base URL the client is configured to use.
func (c *Client) BaseURL() string { return c.transport.baseURL }

// waitInterval returns the polling interval to use.
func (o *WaitOptions) waitInterval() time.Duration {
	if o == nil || o.PollInterval <= 0 {
		return 5 * time.Second
	}
	return o.PollInterval
}
