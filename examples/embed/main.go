// Command embed creates text and video embeddings with Embed API v2.
//
// Usage:
//
//	go run ./examples/embed
//	ASSET_ID=... go run ./examples/embed
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
	client, err := twelvelabs.NewClient(twelvelabs.Options{})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	text, err := client.Embed.Embed(ctx, &twelvelabs.EmbedRequest{
		InputType: twelvelabs.EmbedInputText,
		ModelName: twelvelabs.ModelMarengo35,
		Text:      &twelvelabs.EmbedTextInput{InputText: "a dog catching a frisbee"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("text embedding: %d dimensions\n", len(text.Data[0].Embedding))

	assetID := os.Getenv("ASSET_ID")
	if assetID == "" {
		return
	}

	video, err := client.Embed.Embed(ctx, &twelvelabs.EmbedRequest{
		InputType: twelvelabs.EmbedInputVideo,
		ModelName: twelvelabs.ModelMarengo30,
		Video: &twelvelabs.EmbedAudioVideoInput{
			MediaSource:     twelvelabs.MediaSourceFromAssetID(assetID),
			EmbeddingOption: []string{twelvelabs.EmbeddingOptionVisual},
			EmbeddingScope:  []string{twelvelabs.EmbeddingScopeClip},
			Segmentation:    twelvelabs.FixedSegments(6),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, segment := range video.Data {
		if segment.StartSec == nil || segment.EndSec == nil {
			continue
		}
		fmt.Printf("segment %.1fs-%.1fs: %d dimensions\n", *segment.StartSec, *segment.EndSec, len(segment.Embedding))
	}
}
