// Command analyze streams a summary of an asset and then extracts timestamped
// segments.
//
// Usage:
//
//	ASSET_ID=... go run ./examples/analyze
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/favourthemaster/twelvelabs-go-sdk"
)

func main() {
	assetID := os.Getenv("ASSET_ID")
	if assetID == "" {
		log.Fatal("set ASSET_ID to a ready asset")
	}

	client, err := twelvelabs.NewClient(twelvelabs.Options{})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	fmt.Println("--- stream ---")
	err = client.Analyze.Stream(ctx, &twelvelabs.AnalyzeRequest{
		ModelName: twelvelabs.ModelPegasus16,
		Video:     twelvelabs.VideoFromAssetID(assetID),
		Prompt:    "Summarize this video in three sentences.",
	}, func(event twelvelabs.AnalyzeStreamEvent) error {
		if event.EventType == twelvelabs.StreamEventText {
			fmt.Print(event.Text)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println()

	fmt.Println("--- segments ---")
	task, err := client.Analyze.CreateTask(ctx, &twelvelabs.AnalyzeTaskCreateRequest{
		ModelName:    twelvelabs.ModelPegasus16,
		Video:        twelvelabs.VideoFromAssetID(assetID),
		AnalysisMode: twelvelabs.AnalysisModeTimeBasedMetadata,
		ResponseFormat: &twelvelabs.ResponseFormat{
			Type: twelvelabs.ResponseFormatSegmentDefinitions,
			SegmentDefinitions: []twelvelabs.SegmentDefinition{{
				ID:          "topic",
				Description: "A change of topic",
				Fields: []twelvelabs.SegmentField{{
					Name:        "summary",
					Type:        twelvelabs.SegmentFieldString,
					Description: "One sentence describing the topic",
				}},
			}},
		},
		MaxTokens: 8192,
	})
	if err != nil {
		log.Fatal(err)
	}

	task, err = client.Analyze.WaitForTask(ctx, task.TaskID, &twelvelabs.WaitOptions{
		PollInterval: 5 * time.Second,
		OnPoll:       func(status string) error { fmt.Println("task:", status); return nil },
	})
	if err != nil {
		log.Fatal(err)
	}
	if task.Result == nil {
		log.Fatal("task finished without a result")
	}

	segments, err := twelvelabs.ParseAnalyzeSegments(task.Result.Data)
	if err != nil {
		log.Fatal(err)
	}
	for _, segment := range segments["topic"] {
		metadata, _ := json.Marshal(segment.Metadata)
		fmt.Printf("%v-%v %s\n", segment.StartTime, segment.EndTime, metadata)
	}
}
