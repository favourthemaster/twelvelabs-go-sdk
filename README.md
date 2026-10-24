# TwelveLabs Go SDK

A Go client for the [TwelveLabs Video Understanding Platform](https://docs.twelvelabs.io), targeting API **v1.3**.

TwelveLabs publishes official SDKs for Python and Node.js. There is no official Go SDK; this package is a community client that follows the v1.3 HTTP API closely.

```go
client, err := twelvelabs.NewClient(twelvelabs.Options{APIKey: os.Getenv("TWELVE_LABS_API_KEY")})
```

- Typed request and response models for every endpoint in the v1.3 spec except Agents/Jockey.
- Context-first methods, no hidden globals.
- Bounded retries with `Retry-After` support.
- Streaming analysis (NDJSON), batch results (JSON Lines), and resumable multipart uploads.
- Webhook signature verification.

## Installation

```bash
go get github.com/favourthemaster/twelvelabs-go-sdk
```

Requires Go 1.22 or newer.

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/favourthemaster/twelvelabs-go-sdk"
)

func main() {
	ctx := context.Background()

	client, err := twelvelabs.NewClient(twelvelabs.Options{
		APIKey: os.Getenv("TWELVE_LABS_API_KEY"),
	})
	if err != nil {
		log.Fatal(err)
	}

	index, err := client.Indexes.Create(ctx, &twelvelabs.IndexCreateRequest{
		IndexName: "product-demos",
		Models: []twelvelabs.IndexModel{{
			ModelName:    twelvelabs.ModelMarengo30,
			ModelOptions: []string{twelvelabs.ModalityVisual, twelvelabs.ModalityAudio},
		}},
	})
	if err != nil {
		log.Fatal(err)
	}

	file, err := twelvelabs.NewFileUpload("demo.mp4")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	asset, err := client.Assets.Create(ctx, &twelvelabs.AssetCreateRequest{File: file})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := client.Assets.WaitForReady(ctx, asset.ID, nil); err != nil {
		log.Fatal(err)
	}

	indexed, err := client.IndexedAssets.Create(ctx, index.ID, &twelvelabs.IndexedAssetCreateRequest{
		AssetID: asset.ID,
	})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := client.IndexedAssets.WaitForReady(ctx, index.ID, indexed.ID, nil); err != nil {
		log.Fatal(err)
	}

	results, err := client.Search.Query(ctx, &twelvelabs.SearchRequest{
		IndexID:       index.ID,
		QueryText:     "the pricing slide",
		SearchOptions: []string{twelvelabs.ModalityVisual, twelvelabs.ModalityTranscription},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, result := range results.Data {
		fmt.Printf("%.1fs-%.1fs %s\n", result.Start, result.End, result.ThumbnailURL)
	}
}
```

## Client options

| Option | Default | Description |
| --- | --- | --- |
| `APIKey` | `TWELVE_LABS_API_KEY` | Sent as `x-api-key` on every request. |
| `BaseURL` | `TWELVELABS_BASE_URL`, then `https://api.twelvelabs.io/v1.3` | Override for proxies and tests. |
| `HTTPClient` | `&http.Client{}` | Reuse your own transport, tracing, or timeouts. No overall timeout is set by default so streams and uploads are not cut off. |
| `MaxRetries` | `3` | Retries after the first attempt. Pass `twelvelabs.NoRetries` to disable. |
| `UserAgent` | `twelvelabs-go-sdk/<version>` | Override the `User-Agent`. |

## Services

| Service | Field | Covers |
| --- | --- | --- |
| Assets | `client.Assets` | Direct upload, URL import, multipart upload, metadata, transcription |
| Indexes | `client.Indexes` | Create, list, update, delete indexes; deprecated `/videos` aliases |
| Indexed assets | `client.IndexedAssets` | Add assets to indexes, poll status, set metadata |
| Search | `client.Search` | Text, image, and composed queries; page tokens |
| Analyze | `client.Analyze` | Sync, streaming, async tasks, batch jobs |
| Embed | `client.Embed` | Embed API v2, synchronous and asynchronous |
| Entities | `client.Entities` | Entity collections, bulk creation, asset association |
| Tasks | `client.Tasks` | Legacy `/tasks` indexing (deprecated by TwelveLabs) |

Not implemented: Agents/Jockey (`/responses`, knowledge stores) and data connectors. Those are outside the SDK's scope.

## Assets

### Direct upload and URL import

```go
asset, err := client.Assets.Create(ctx, &twelvelabs.AssetCreateRequest{
	URL:      "https://example.com/talk.mp4",
	Filename: "talk.mp4",
})

asset, err = client.Assets.Create(ctx, &twelvelabs.AssetCreateRequest{
	File: file, // *twelvelabs.FileUpload
})
```

The API caps direct uploads at 200 MB for video, audio, and documents, and 32 MB for images. Use the multipart flow for anything larger.

### Multipart upload

`Upload` opens a session, streams the file in parallel chunks, reports the ETags, and returns the asset:

```go
file, err := twelvelabs.NewFileUpload("webinar.mp4")
if err != nil {
	return err
}
defer file.Close()

asset, err := client.Assets.Upload(ctx, &twelvelabs.MultipartUploadRequest{
	File:        file,
	Concurrency: 4,
	OnProgress: func(done, total int) {
		fmt.Printf("%d/%d chunks\n", done, total)
	},
})
```

The lower-level methods (`CreateMultipartUpload`, `GetUploadStatus`, `ReportChunks`, `RequestPresignedURLs`, `ListIncompleteUploads`) are available when you need to resume an interrupted upload yourself. Chunk indices are 1-based, and the reported proof is the ETag with surrounding quotes removed.

### Waiting

Every asynchronous resource has a wait helper. All of them accept `*WaitOptions`:

```go
asset, err := client.Assets.WaitForReady(ctx, assetID, &twelvelabs.WaitOptions{
	PollInterval: 5 * time.Second,
	Timeout:      10 * time.Minute,
	OnPoll:       func(status string) error { log.Println(status); return nil },
})
```

A failed resource returns the resource *and* an error, so callers can inspect `asset.Error` before discarding it. A timeout returns `twelvelabs.ErrTimeout`.

## Search

Search is always sent as `multipart/form-data`, because the API accepts image files:

```go
results, err := client.Search.Query(ctx, &twelvelabs.SearchRequest{
	IndexID:       indexID,
	QueryText:     "a person explaining a chart",
	SearchOptions: []string{twelvelabs.ModalityVisual, twelvelabs.ModalityAudio},
	GroupBy:       twelvelabs.GroupByVideo,
	Filter: map[string]any{
		"duration": map[string]any{"gte": 60},
	},
})
```

Composed image + text queries set both `QueryText` and `QueryMediaURLs` (or `QueryMediaFiles`). Search result pages are cached by the platform and expire; continue with:

```go
next, err := client.Search.GetPage(ctx, results.PageInfo.NextPageToken, nil)
```

## Analyze

### Synchronous

```go
response, err := client.Analyze.Analyze(ctx, &twelvelabs.AnalyzeRequest{
	ModelName: twelvelabs.ModelPegasus16,
	Video:     twelvelabs.VideoFromAssetID(assetID),
	Prompt:    "Summarize this video in three bullets.",
})
fmt.Println(response.Data)
```

### Streaming

Streams are NDJSON, not server-sent events. The handler is called for `stream_start`, every `text_generation` chunk, and `stream_end`:

```go
err := client.Analyze.Stream(ctx, request, func(event twelvelabs.AnalyzeStreamEvent) error {
	switch event.EventType {
	case twelvelabs.StreamEventText:
		fmt.Print(event.Text)
	case twelvelabs.StreamEventEnd:
		fmt.Println()
	}
	return nil
})
```

If generation fails mid-stream, the final event carries the error and `Stream` returns `*twelvelabs.AnalyzeStreamError` after invoking the handler.

### Async tasks

```go
task, err := client.Analyze.CreateTask(ctx, &twelvelabs.AnalyzeTaskCreateRequest{
	ModelName: twelvelabs.ModelPegasus16,
	Video:     twelvelabs.VideoFromURL("https://example.com/talk.mp4"),
	Prompt:    "List every product shown.",
	CustomID:  "talk-2026-05",
})
task, err = client.Analyze.WaitForTask(ctx, task.TaskID, nil)
fmt.Println(task.Result.Data)
```

### Segment extraction

Set `AnalysisModeTimeBasedMetadata` and describe the segments to pull out. The result is a JSON string; `ParseAnalyzeSegments` decodes it:

```go
task, err := client.Analyze.CreateTask(ctx, &twelvelabs.AnalyzeTaskCreateRequest{
	ModelName:    twelvelabs.ModelPegasus16,
	Video:        twelvelabs.VideoFromAssetID(assetID),
	AnalysisMode: twelvelabs.AnalysisModeTimeBasedMetadata,
	ResponseFormat: &twelvelabs.ResponseFormat{
		Type: twelvelabs.ResponseFormatSegmentDefinitions,
		SegmentDefinitions: []twelvelabs.SegmentDefinition{{
			ID:          "goal",
			Description: "A goal being scored",
			Fields: []twelvelabs.SegmentField{{
				Name: "team", Type: twelvelabs.SegmentFieldString, Description: "Scoring team",
			}},
		}},
	},
})

task, err = client.Analyze.WaitForTask(ctx, task.TaskID, nil)
segments, err := twelvelabs.ParseAnalyzeSegments(task.Result.Data)
```

### Batches

Batches apply the same prompt to many assets. Results are JSON Lines and can be large; `BatchResults` collects them, `StreamBatchResults` handles them one at a time:

```go
batch, err := client.Analyze.CreateBatch(ctx, &twelvelabs.AnalyzeBatchCreateRequest{
	ModelName:    twelvelabs.ModelPegasus15,
	AnalysisMode: twelvelabs.AnalysisModeGeneral,
	Defaults: &twelvelabs.BatchDefaults{
		Prompt: &twelvelabs.AnalyzePromptV2{InputText: "Summarize this video."},
	},
	Requests: []twelvelabs.BatchItem{
		{Video: &twelvelabs.BatchVideoContext{Type: "asset_id", AssetID: assetA}, CustomID: "a"},
		{Video: &twelvelabs.BatchVideoContext{Type: "asset_id", AssetID: assetB}, CustomID: "b"},
	},
})

if _, err := client.Analyze.WaitForBatch(ctx, batch.BatchID, nil); err != nil {
	return err
}
err = client.Analyze.StreamBatchResults(ctx, batch.BatchID, func(item twelvelabs.BatchResultItem) error {
	fmt.Println(item.CustomID, item.Status)
	return nil
})
```

Batch requests only accept `pegasus1.5`, and a batch cannot mix analysis modes.

## Embed (v2)

```go
response, err := client.Embed.Embed(ctx, &twelvelabs.EmbedRequest{
	InputType:          twelvelabs.EmbedInputText,
	ModelName:          twelvelabs.ModelMarengo35,
	Text:               &twelvelabs.EmbedTextInput{InputText: "a red bicycle"},
	EmbeddingDimension: twelvelabs.EmbeddingDimensions256,
})
fmt.Println(len(response.Data[0].Embedding)) // 256
```

Video and audio embeddings use `EmbedAudioVideoInput` with `EmbeddingOption`, `EmbeddingScope`, and `Segmentation`:

```go
response, err := client.Embed.Embed(ctx, &twelvelabs.EmbedRequest{
	InputType: twelvelabs.EmbedInputVideo,
	ModelName: twelvelabs.ModelMarengo30,
	Video: &twelvelabs.EmbedAudioVideoInput{
		MediaSource:     twelvelabs.MediaSourceFromAssetID(assetID),
		EmbeddingOption: []string{twelvelabs.EmbeddingOptionVisual, twelvelabs.EmbeddingOptionAudio},
		EmbeddingScope:  []string{twelvelabs.EmbeddingScopeClip},
		Segmentation:    twelvelabs.FixedSegments(6),
	},
})
```

Long media belongs on the asynchronous endpoint:

```go
task, err := client.Embed.CreateTask(ctx, &twelvelabs.EmbedTaskCreateRequest{
	InputType: twelvelabs.EmbedTaskInputVideo,
	ModelName: twelvelabs.ModelMarengo30,
	Video:     &twelvelabs.EmbedAudioVideoInput{MediaSource: twelvelabs.MediaSourceFromURL(videoURL)},
})
task, err = client.Embed.WaitForTask(ctx, task.ID, nil)
```

Marengo 3.5 adds `EmbeddingDimension`, `EmbeddingUncertainty`, documents, and multi-input embeddings. Marengo 3.0 rejects those fields.

## Entities

```go
collection, err := client.Entities.CreateCollection(ctx, &twelvelabs.EntityCollectionCreateRequest{
	Name: "Speakers",
})

entity, err := client.Entities.CreateEntity(ctx, collection.ID, &twelvelabs.EntityCreateRequest{
	Name:     "Ada Lovelace",
	AssetIDs: []string{assetID},
})
```

Bulk creation reports partial success instead of failing the whole request:

```go
result, err := client.Entities.BulkCreateEntities(ctx, collection.ID, &twelvelabs.EntityBulkCreateRequest{
	Entities: []twelvelabs.EntityCreateRequest{
		{Name: "Grace Hopper", AssetIDs: []string{assetA}},
		{Name: "Alan Turing", AssetIDs: []string{assetB}},
	},
})
fmt.Println(result.SuccessCount, result.FailedCount, result.Errors)
```

## Webhooks

TwelveLabs signs webhooks with HMAC-SHA256 over `<timestamp>.<raw body>` and sends the value in `TL-Signature: t=<unix>,v1=<hex>`. Verify against the raw body before decoding:

```go
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	event, err := twelvelabs.NewWebhookVerifier(webhookSecret).
		VerifyAndParse(body, r.Header.Get("TL-Signature"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	switch event.Type {
	case twelvelabs.WebhookEventIndexTaskReady:
		data, _ := event.TaskData()
		log.Printf("task %s is ready, duration %.1fs", data.ID, data.Metadata.Duration)
	case twelvelabs.WebhookEventAnalyzeTaskFailed:
		data, _ := event.TaskData()
		log.Printf("analysis %s failed: %s", data.ID, data.Error.Message)
	}

	w.WriteHeader(http.StatusOK)
}
```

`WebhookVerifier.Tolerance` defaults to five minutes and rejects replayed payloads outside that window. Set it to zero to disable the check.

## Pagination

List methods return `*Page[T]` with `Data` and `PageInfo`:

```go
page, err := client.Indexes.List(ctx, &twelvelabs.IndexListOptions{
	Pagination: twelvelabs.Pagination{Page: 1, PageLimit: 20},
})
for page.PageInfo.Page < page.PageInfo.TotalPage {
	page, err = client.Indexes.List(ctx, &twelvelabs.IndexListOptions{
		Pagination: twelvelabs.Pagination{Page: page.PageInfo.Page + 1, PageLimit: 20},
	})
}
```

Search is the exception: it pages by opaque token, not number.

## Errors

Every non-2xx response becomes `*twelvelabs.APIError`:

```go
_, err := client.Assets.Get(ctx, assetID)
switch {
case twelvelabs.IsNotFound(err):
	// 404
case twelvelabs.IsUnauthorized(err):
	// 401 or 403
case twelvelabs.IsRateLimited(err):
	// 429
default:
	var apiErr *twelvelabs.APIError
	if errors.As(err, &apiErr) {
		log.Printf("code=%s message=%s request_id=%s", apiErr.Code, apiErr.Message, apiErr.RequestID)
	}
}
```

Requests rejected locally (missing IDs, conflicting fields, out-of-range values) return `*twelvelabs.ValidationError` and never reach the network.

## Retries

- `429 Too Many Requests` is always retried, honouring `Retry-After`.
- `5xx` is retried only for idempotent methods (`GET`, `HEAD`, `PUT`, `DELETE`).
- Multipart and streaming bodies are never retried, because they cannot be replayed. A failed multipart upload must be resumed with the session API.

## Migrating from 0.2.x

Version 0.3 is a breaking rewrite of the v1.2-era client. See [MIGRATION.md](MIGRATION.md).

## Development

```bash
gofmt -l .
go vet ./...
go test -race ./...
```

The tests use `httptest` and do not touch the network or require an API key.
