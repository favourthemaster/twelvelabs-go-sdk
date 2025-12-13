package main

import (
	"context"
	"fmt"
	"log"

	"github.com/favourthemaster/twelvelabs-go-sdk"
	"github.com/favourthemaster/twelvelabs-go-sdk/pkg/models"
)

func main() {
	// Initialize client using placeholder API key
	client, err := twelvelabs.NewTwelveLabs(&twelvelabs.Options{
		APIKey: "your-api-key-here", // Replace with your actual API key
	})
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	fmt.Println("🔍 TwelveLabs Go SDK - Search Examples")
	fmt.Println("=====================================")

	indexID := "your-index-id-here" // Replace with your actual index ID

	// 1. Basic text search
	fmt.Println("\n📝 Basic text search...")
	textResult, err := client.Search.SearchByText(context.Background(),
		indexID,
		"your search query here",
		[]string{"visual", "audio"},
	)
	if err != nil {
		log.Printf("Error in text search: %v", err)
	} else {
		fmt.Printf("✅ Text search found %d results\n", len(textResult.Data))
		displaySearchResults(textResult.Data, 3)
	}

	// 2. Image-based search
	fmt.Println("\n🖼️ Image-based search...")
	imageResults, err := client.Search.SearchByImage(context.Background(),
		indexID,
		"https://example.com/your-image-url.jpg",
		[]string{"visual"},
	)
	if err != nil {
		log.Printf("Error in image search: %v", err)
	} else {
		fmt.Printf("✅ Image search found %d results\n", len(imageResults.Data))
		displaySearchResults(imageResults.Data, 3)
	}

	// 4. Advanced search with custom parameters
	fmt.Println("\n⚙️ Advanced search with custom parameters...")
	advancedResults, err := client.Search.Query(context.Background(), &models.SearchQueryRequest{
		IndexID:       indexID,
		QueryText:     "your advanced search query",
		SearchOptions: []string{"visual", "audio"},
	})
	if err != nil {
		log.Printf("Error in advanced search: %v", err)
	} else {
		fmt.Printf("✅ Advanced search found %d results\n", len(advancedResults.Data))
		displaySearchResults(advancedResults.Data, 5)
	}

	//// 5. Search with local media file
	//fmt.Println("\n📁 Search with local image file...")
	//localFileResults, err := client.Search.Query(context.Background(), &models.SearchQueryRequest{
	//	IndexID:        indexID,
	//	QueryMediaType: "image",
	//	QueryMediaFile: "./assets/search_sample.png",
	//	SearchOptions:  []string{"visual"},
	//})
	//if err != nil {
	//	log.Printf("Error in local file search: %v", err)
	//} else {
	//	fmt.Printf("✅ Local file search found %d results\n", len(localFileResults.Data))
	//	displaySearchResults(localFileResults.Data, 3)
	//}

	// 7. Search with pagination and filtering
	fmt.Println("\n📄 Search with pagination...")
	paginatedResults, err := client.Search.Query(context.Background(), &models.SearchQueryRequest{
		IndexID:       indexID,
		QueryText:     "your paginated search query",
		SearchOptions: []string{"visual"},
	})
	if err != nil {
		log.Printf("Error in paginated search: %v", err)
	} else {
		fmt.Printf("✅ Paginated search found %d results\n", len(paginatedResults.Data))

		if paginatedResults.PageInfo.NextPageToken != "" {
			fmt.Println("   📄 Getting next page...")
			nextPageResults, err := client.Search.Retrieve(context.Background(),
				paginatedResults.PageInfo.NextPageToken,
			)
			if err != nil {
				log.Printf("Error getting next page: %v", err)
			} else {
				fmt.Printf("   ✅ Next page has %d results\n", len(nextPageResults.Data))
			}
		}
	}

	// 8. Search result analysis
	fmt.Println("\n📊 Search result analysis...")
	if len(textResult.Data) > 0 {
		fmt.Println("   Analyzing confidence scores...")

		highConfidence := 0
		mediumConfidence := 0
		lowConfidence := 0

		for _, result := range textResult.Data {
			switch result.Confidence {
			case "high":
				highConfidence++
			case "medium":
				mediumConfidence++
			case "low":
				lowConfidence++
			}
		}

		fmt.Printf("   📈 Confidence distribution:\n")
		fmt.Printf("      High: %d results\n", highConfidence)
		fmt.Printf("      Medium: %d results\n", mediumConfidence)
		fmt.Printf("      Low: %d results\n", lowConfidence)
	}

	fmt.Println("\n🎉 Search examples completed!")
	fmt.Println("\nSearch types demonstrated:")
	fmt.Println("- ✅ Text-based search")
	fmt.Println("- ✅ Image-based search")
	fmt.Println("- ✅ Video-based search")
	fmt.Println("- ✅ Multi-modal search")
	fmt.Println("- ✅ Local file search")
	fmt.Println("- ✅ Advanced search with custom parameters")
	fmt.Println("- ✅ Paginated search results")
	fmt.Println("- ✅ Search result analysis")
}

func displaySearchResults(results []models.SearchResult, limit int) {
	if len(results) == 0 {
		fmt.Println("   No results found")
		return
	}

	displayCount := limit
	if len(results) < limit {
		displayCount = len(results)
	}

	fmt.Printf("   Top %d results:\n", displayCount)
	for i := 0; i < displayCount; i++ {
		result := results[i]
		fmt.Printf("   %d. Video: %s | Score: %.4f | Time: %.1f-%.1fs | Confidence: %s\n",
			i+1, result.VideoID, result.Score, result.Start, result.End, result.Confidence)
	}
}
