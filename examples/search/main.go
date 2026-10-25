// Command search runs text and image queries against an index.
//
// Usage:
//
//	INDEX_ID=... go run ./examples/search "the pricing slide"
//	INDEX_ID=... IMAGE=./slide.png go run ./examples/search
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
	indexID := os.Getenv("INDEX_ID")
	if indexID == "" {
		log.Fatal("set INDEX_ID to the index you want to search")
	}

	client, err := twelvelabs.NewClient(twelvelabs.Options{})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	request := &twelvelabs.SearchRequest{
		IndexID:       indexID,
		SearchOptions: []string{twelvelabs.ModalityVisual, twelvelabs.ModalityTranscription},
		GroupBy:       twelvelabs.GroupByClip,
		PageLimit:     10,
	}

	if query := os.Args[1:]; len(query) > 0 {
		request.QueryText = query[0]
	}
	if path := os.Getenv("IMAGE"); path != "" {
		file, err := twelvelabs.NewFileUpload(path)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		request.QueryMediaType = twelvelabs.SearchMediaTypeImage
		request.QueryMediaFiles = []*twelvelabs.FileUpload{file}
	}

	response, err := client.Search.Query(ctx, request)
	if err != nil {
		log.Fatal(err)
	}

	printResults(response)

	for response.PageInfo != nil && response.PageInfo.NextPageToken != "" {
		response, err = client.Search.GetPage(ctx, response.PageInfo.NextPageToken, nil)
		if err != nil {
			log.Fatal(err)
		}
		printResults(response)
	}
}

func printResults(response *twelvelabs.SearchResponse) {
	for _, result := range response.Data {
		fmt.Printf("%.1fs-%.1fs rank=%d %s\n", result.Start, result.End, result.Rank, result.VideoID)
		if result.Transcription != "" {
			fmt.Println("  ", result.Transcription)
		}
		for _, clip := range result.Clips {
			fmt.Printf("   clip %.1fs-%.1fs\n", clip.Start, clip.End)
		}
	}
}
