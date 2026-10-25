# Migrating from 0.2.x to 0.3.0

0.3.0 is a clean rewrite for **TwelveLabs API v1.3**. The 0.2.x client targeted v1.2 endpoints, and several of them no longer work: for example, `POST /indexes` now accepts only `marengo3.0`, and video analysis moved from `video_id` to assets.

There are no compatibility shims. The package path is unchanged:

```go
import "github.com/favourthemaster/twelvelabs-go-sdk"
```

## Constructor

```go
// 0.2.x
client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{APIKey: key})

// 0.3.0
client, err := twelvelabs.NewClient(twelvelabs.Options{APIKey: key})
```

`Options` now takes `HTTPClient` instead of `Timeout`. To keep an overall timeout, supply a configured client:

```go
client, err := twelvelabs.NewClient(twelvelabs.Options{
	APIKey:     key,
	HTTPClient: &http.Client{Timeout: 60 * time.Second},
})
```

The default client has no overall timeout so that long uploads and streams are not truncated; use `context.WithTimeout` for per-call deadlines.

## Contexts

Every method takes a `context.Context` as its first argument. The wrapper layer that omitted contexts is gone.

## Package layout

| 0.2.x | 0.3.0 |
| --- | --- |
| `pkg/client` | root package `twelvelabs` |
| `pkg/models` | root package `twelvelabs` |
| `pkg/services` | root package `twelvelabs` |
| `pkg/wrappers` | root package `twelvelabs` |
| `pkg/errors` | root package `twelvelabs` |

Imports of the old subpackages must be deleted. All types are now `twelvelabs.X`.

## Services

| 0.2.x | 0.3.0 |
| --- | --- |
| `client.Indexes.Create` | `client.Indexes.Create` (same name, new model constants) |
| `client.Indexes.Retrieve` | `client.Indexes.Get` |
| `client.Indexes.Videos.List` | `client.IndexedAssets.List` |
| `client.Indexes.Videos.Retrieve` | `client.IndexedAssets.Get` (now takes `*IndexedAssetGetOptions`) |
| `client.Tasks.Create` | `client.Assets.Create` + `client.IndexedAssets.Create` |
| `client.Tasks.CreateBulk` | `client.Analyze.CreateBatch` (for analysis) or a loop over assets (for indexing) |
| `client.Tasks.WaitForDone` | `client.IndexedAssets.WaitForReady` |
| `client.Search.Query` | `client.Search.Query` (multipart, image support) |
| `client.Search.Retrieve` | `client.Search.GetPage` |
| `client.Search.SearchByText` | `client.Search.Query` with `QueryText` |
| `client.Search.SearchByImage` | `client.Search.Query` with `QueryMediaURLs` |
| `client.Embed.CreateTextEmbedding` | `client.Embed.Embed` with `Text` |
| `client.Embed.CreateImageEmbedding` | `client.Embed.Embed` with `Image` |
| `client.Embed.CreateVideoEmbedding` | `client.Embed.Embed` with `Video` |
| `client.Embed.CreateAudioEmbedding` | `client.Embed.Embed` with `Audio` |
| `client.Analyze.Analyze` | `client.Analyze.Analyze` |
| `client.Analyze.AnalyzeStream` | `client.Analyze.Stream` |
| `client.Analyze.GenerateSummary` | `client.Analyze.Analyze` with a prompt |
| `client.Analyze.GenerateGist` | `client.Analyze.Analyze` with a prompt |

## Return types

List methods used to return `[]models.T` and accept `map[string]string` filters. They now return `*twelvelabs.Page[T]` and take typed options:

```go
// 0.2.x
indexes, err := client.Indexes.List(ctx, map[string]string{"page": "1"})

// 0.3.0
page, err := client.Indexes.List(ctx, &twelvelabs.IndexListOptions{
	Pagination: twelvelabs.Pagination{Page: 1, PageLimit: 20},
})
indexes := page.Data
```

`page.PageInfo` carries `TotalPage`, `TotalResults`, and, for search, `NextPageToken`.

`Entities.AddEntityAssets` and `RemoveEntityAssets` now return the updated `*Entity`, matching the API response, instead of only an error.

## Index models

The v1.3 API accepts a single model name for indexes:

```go
// 0.2.x
Models: []models.Model{{ModelName: "Marengo-retrieval-2.6", ModelOptions: []string{"visual", "conversation"}}}

// 0.3.0
Models: []twelvelabs.IndexModel{{
	ModelName:    twelvelabs.ModelMarengo30,
	ModelOptions: []string{twelvelabs.ModalityVisual, twelvelabs.ModalityAudio},
}}
```

`conversation` is no longer a valid model option; it was replaced by `audio`. If `Models` is omitted, `Create` defaults to Marengo 3.0 with `visual` and `audio`.

## Tasks are deprecated

`POST /tasks` still exists in the v1.3 spec and remains available as `client.Tasks`, but the platform's documentation treats assets as the supported path. New code should:

1. `client.Assets.Create` (or `Assets.Upload` for large files)
2. `client.Assets.WaitForReady`
3. `client.IndexedAssets.Create`
4. `client.IndexedAssets.WaitForReady`

The legacy service is marked `Deprecated:` in godoc so `staticcheck` and editors flag it.

## Analysis

Async analysis no longer takes `video_id` directly. Use an asset ID, a URL, or base64 data:

```go
// 0.2.x
&models.AnalyzeRequest{VideoID: videoID, Prompt: "..."}

// 0.3.0
&twelvelabs.AnalyzeRequest{
	ModelName: twelvelabs.ModelPegasus16,
	Video:     twelvelabs.VideoFromAssetID(assetID),
	Prompt:    "...",
}
```

`AnalyzeStream` took a callback over raw stream responses. `Stream` delivers typed `AnalyzeStreamEvent` values, and streams are NDJSON. Summaries and gists are prompts now; the dedicated helpers were removed because they were thin prompt wrappers around removed v1.2 endpoints.

## Embeddings

The old helpers hard-coded v1.2 model names. Embed v2 uses `marengo3.0` or `marengo3.5`, an explicit `InputType`, and nested inputs:

```go
response, err := client.Embed.Embed(ctx, &twelvelabs.EmbedRequest{
	InputType: twelvelabs.EmbedInputText,
	ModelName: twelvelabs.ModelMarengo30,
	Text:      &twelvelabs.EmbedTextInput{InputText: "hello"},
})
```

Marengo 3.0 uses `text`, `image`, `text_image`, `audio`, `video`, and `multi_input` (image sources only). The Marengo 3.5 synchronous endpoint accepts only `multi_input`, where sources may also be audio, video, or documents; on the asynchronous endpoint, documents and images require 3.5, while audio and video work with either model.

## Errors

The old package exposed one struct per status code (`BadRequestError`, `NotFoundError`, and so on). All HTTP failures are now a single `*twelvelabs.APIError` with `StatusCode`, `Code`, `Message`, `Details`, `RequestID`, and `RetryAfter`.

```go
// 0.2.x
if _, ok := err.(*errors.NotFoundError); ok { ... }

// 0.3.0
if twelvelabs.IsNotFound(err) { ... }

var apiErr *twelvelabs.APIError
if errors.As(err, &apiErr) {
	log.Println(apiErr.StatusCode, apiErr.Code, apiErr.RequestID)
}
```

Locally rejected requests return `*twelvelabs.ValidationError`, which replaces `errors.NewValidationError`. Timeouts from wait helpers return `twelvelabs.ErrTimeout`.

## Removed

- `pkg/` subpackages and their constructors.
- `client.GetCustomAuthorizationHeaders`; the transport sets `x-api-key` itself.
- Analysis summary/gist helpers.
- `Tasks.CreateBulk`, `Tasks.WaitForCompletion`, and `Tasks.WaitForCompletionWithTimeout`; use `Analyze.CreateBatch` and `WaitForBatch`, or `IndexedAssets.WaitForReady` in a loop.
- The `Timeout` option; use `HTTPClient` or a context.

## New in 0.3.0

- Assets API: direct, URL, and multipart uploads, plus transcription.
- Indexed assets, including `embedding` and `hls` data.
- Search with image queries, filters, grouping, and page tokens.
- Async analysis tasks and batch jobs, including segment extraction.
- Embed API v2, synchronous and asynchronous.
- Entity collections.
- Webhook signature verification.
- Retries with `Retry-After` support.
