// Command upload uploads a video, indexes it, and waits for it to be
// searchable.
//
// Usage:
//
//	go run ./examples/upload /path/to/video.mp4
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/favourthemaster/twelvelabs-go-sdk"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %s <video-file>", os.Args[0])
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	client, err := twelvelabs.NewClient(twelvelabs.Options{})
	if err != nil {
		log.Fatal(err)
	}

	index, err := client.Indexes.Create(ctx, &twelvelabs.IndexCreateRequest{
		IndexName: "examples",
		Models: []twelvelabs.IndexModel{{
			ModelName:    twelvelabs.ModelMarengo30,
			ModelOptions: []string{twelvelabs.ModalityVisual, twelvelabs.ModalityAudio},
		}},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("index:", index.ID)

	file, err := twelvelabs.NewFileUpload(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	asset, err := client.Assets.Upload(ctx, &twelvelabs.MultipartUploadRequest{
		File: file,
		OnProgress: func(done, total int) {
			fmt.Printf("\ruploaded %d/%d chunks", done, total)
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\rasset: %s\n", asset.ID)

	ready, err := client.Assets.WaitForReady(ctx, asset.ID, &twelvelabs.WaitOptions{
		PollInterval: 5 * time.Second,
		OnPoll:       func(status string) error { fmt.Println("asset:", status); return nil },
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("duration:", ready.Duration, "filename:", ready.Filename)

	indexed, err := client.IndexedAssets.Create(ctx, index.ID, &twelvelabs.IndexedAssetCreateRequest{
		AssetID: asset.ID,
		UserMetadata: map[string]any{
			"source": "examples/upload",
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	if _, err := client.IndexedAssets.WaitForReady(ctx, index.ID, indexed.ID, &twelvelabs.WaitOptions{
		PollInterval: 5 * time.Second,
		OnPoll:       func(status string) error { fmt.Println("indexing:", status); return nil },
	}); err != nil {
		log.Fatal(err)
	}

	fmt.Println("ready to search: index_id=" + index.ID)
}
