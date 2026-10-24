// Package twelvelabs is a Go client for the TwelveLabs Video Understanding
// Platform API (v1.3).
//
// The SDK is organised around the platform's resources:
//
//   - Assets: upload video, audio, images, and documents, then poll until ready.
//   - Indexes: group related assets and configure the Marengo model.
//   - IndexedAssets: index an asset so it becomes searchable.
//   - Search: find moments in indexed video using text and/or image queries.
//   - Analyze: generate text from video or images, synchronously, as a stream,
//     as an async task, or as a batch.
//   - Embed: create multimodal embeddings with the Embed API v2.
//   - Entities: identify specific people in video with reference images.
//   - Tasks: legacy video indexing tasks (deprecated by TwelveLabs).
//
// # Getting started
//
//	client, err := twelvelabs.NewClient(twelvelabs.Options{
//		APIKey: os.Getenv("TWELVE_LABS_API_KEY"),
//	})
//	if err != nil {
//		return err
//	}
//
//	index, err := client.Indexes.Create(ctx, &twelvelabs.IndexCreateRequest{
//		IndexName: "lectures",
//		Models: []twelvelabs.IndexModel{{
//			ModelName:    twelvelabs.ModelMarengo30,
//			ModelOptions: []string{twelvelabs.ModalityVisual, twelvelabs.ModalityAudio},
//		}},
//	})
//	if err != nil {
//		return err
//	}
//
//	// Upload an asset, wait for it to be processed, then index it.
//	file, err := twelvelabs.NewFileUpload("lecture.mp4")
//	if err != nil {
//		return err
//	}
//	defer file.Close()
//
//	asset, err := client.Assets.Create(ctx, &twelvelabs.AssetCreateRequest{File: file})
//	if err != nil {
//		return err
//	}
//	if _, err := client.Assets.WaitForReady(ctx, asset.ID, nil); err != nil {
//		return err
//	}
//	if _, err := client.IndexedAssets.Create(ctx, index.ID, &twelvelabs.IndexedAssetCreateRequest{
//		AssetID: asset.ID,
//	}); err != nil {
//		return err
//	}
//
//	results, err := client.Search.Query(ctx, &twelvelabs.SearchRequest{
//		IndexID:       index.ID,
//		QueryText:     "a slide about recursion",
//		SearchOptions: []string{twelvelabs.ModalityVisual, twelvelabs.ModalityTranscription},
//	})
//
// # Authentication
//
// Every request carries the x-api-key header. If Options.APIKey is empty, the
// client falls back to the TWELVE_LABS_API_KEY environment variable.
//
// # Retries
//
// The client retries rate-limited requests (HTTP 429) and, for idempotent
// methods, transient server errors. Retry-After is honoured. Streaming
// multipart uploads are never retried because the request body cannot be
// replayed.
package twelvelabs
