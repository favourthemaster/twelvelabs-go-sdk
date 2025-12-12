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

	fmt.Println("🎬 TwelveLabs Go SDK - Video Management Examples")
	fmt.Println("================================================")

	// Use placeholder index ID
	indexID := "your-index-id-here" // Replace with your actual index ID

	// 1. List all videos in an index
	fmt.Println("\n📋 Listing all videos in index...")
	allVideos, err := client.Indexes.Videos.List(context.Background(), indexID, map[string]string{})
	if err != nil {
		log.Printf("Error listing videos: %v", err)
		return
	}
	fmt.Printf("✅ Found %d videos in index\n", len(allVideos))

	if len(allVideos) == 0 {
		fmt.Println("   No videos found. Please upload some videos first.")
		return
	}

	// Display first few videos
	displayCount := min(3, len(allVideos))
	fmt.Printf("   First %d videos:\n", displayCount)
	for i := 0; i < displayCount; i++ {
		video := allVideos[i]
		//fmt.Println(video)
		fmt.Printf("   %d. ID: %s | File: %s | Duration: %.1fs\n",
			i+1, video.ID, video.Metadata.FileName, video.Metadata.Duration)
	}

	// 2. List videos with pagination
	fmt.Println("\n📄 Listing videos with pagination...")
	paginatedVideos, err := client.Indexes.Videos.List(context.Background(), indexID, map[string]string{
		"page_limit": "5",
		"sort_by":    "created_at",
		"sort_order": "desc",
	})
	if err != nil {
		log.Printf("Error listing paginated videos: %v", err)
	} else {
		fmt.Printf("✅ Retrieved %d videos (limited to 5)\n", len(paginatedVideos))
	}

	// 3. Get detailed information for specific video
	if len(allVideos) > 0 {
		firstVideo := allVideos[0]
		fmt.Printf("\n🔍 Getting detailed info for video %s...\n", firstVideo.ID)

		videoDetails, err := client.Indexes.Videos.Retrieve(context.Background(), indexID, firstVideo.ID)
		if err != nil {
			log.Printf("Error retrieving video details: %v", err)
		} else {
			fmt.Printf("✅ Video Details:\n")
			fmt.Printf("   ID: %s\n", videoDetails.ID)
			fmt.Printf("   File Name: %s\n", videoDetails.Metadata.FileName)
			fmt.Printf("   Duration: %.2f seconds\n", videoDetails.Metadata.Duration)
			fmt.Printf("   Index ID: %s\n", indexID)
			fmt.Printf("   Created: %s\n", videoDetails.CreatedAt)
		}
	}

	// 4. Update video metadata
	if len(allVideos) > 0 {
		firstVideo := allVideos[0]
		fmt.Printf("\n📝 Updating metadata for video %s...\n", firstVideo.ID)

		updatedVideo, err := client.Indexes.Videos.Update(context.Background(), indexID, firstVideo.ID, &models.VideoUpdateRequest{
			UserMetadata: map[string]string{
				"title":       "Updated Video Title",
				"description": "This video has been updated with new metadata",
				"category":    "demo",
				"tags":        "nature,outdoor,beautiful",
				"location":    "Mountain Park",
				"weather":     "sunny",
				//"duration":    fmt.Sprintf("%.1f", firstVideo.Metadata.Duration),
				"updated_by": "go_sdk_example",
			},
		})
		if err != nil {
			log.Printf("Error updating video metadata: %v", err)
		} else {
			fmt.Printf("✅ Video metadata updated successfully\n")
			fmt.Printf("   Updated video ID: %s\n", updatedVideo.ID)
		}
	}

	// 5. Bulk metadata update
	fmt.Println("\n📦 Bulk updating metadata for multiple videos...")
	updateCount := min(3, len(allVideos))

	for i := 0; i < updateCount; i++ {
		video := allVideos[i]

		metadata := map[string]string{
			"batch_update": "true",
			"batch_id":     fmt.Sprintf("batch_%d", i+1),
			"processed_at": "2024-01-01T00:00:00Z",
			"category":     "batch_processed",
		}

		// Add specific metadata based on video characteristics
		if video.Metadata.Duration > 60 {
			metadata["length_category"] = "long"
		} else {
			metadata["length_category"] = "short"
		}

		updatedVideo, err := client.Indexes.Videos.Update(context.Background(), indexID, video.ID, &models.VideoUpdateRequest{
			UserMetadata: metadata,
		})
		if err != nil {
			log.Printf("   ❌ Failed to update video %d: %v", i+1, err)
			continue
		}

		fmt.Printf("   ✅ Updated video %d: %s\n", i+1, updatedVideo.ID)
	}

	// 6. Search and filter videos by metadata
	fmt.Println("\n🔍 Filtering videos by duration...")

	longVideos := []models.Video{}
	shortVideos := []models.Video{}

	for _, video := range allVideos {
		if video.Metadata.Duration > 60 {
			longVideos = append(longVideos, video)
		} else {
			shortVideos = append(shortVideos, video)
		}
	}

	fmt.Printf("✅ Video duration analysis:\n")
	fmt.Printf("   Long videos (>60s): %d\n", len(longVideos))
	fmt.Printf("   Short videos (≤60s): %d\n", len(shortVideos))

	// 7. Video statistics and analysis
	fmt.Println("\n📊 Video collection statistics...")

	totalDuration := 0.0
	var longestVideo, shortestVideo models.Video

	if len(allVideos) > 0 {
		longestVideo = allVideos[0]
		shortestVideo = allVideos[0]

		for _, video := range allVideos {
			totalDuration += video.Metadata.Duration

			if video.Metadata.Duration > longestVideo.Metadata.Duration {
				longestVideo = video
			}
			if video.Metadata.Duration < shortestVideo.Metadata.Duration {
				shortestVideo = video
			}
		}

		averageDuration := totalDuration / float64(len(allVideos))

		fmt.Printf("✅ Collection Statistics:\n")
		fmt.Printf("   Total videos: %d\n", len(allVideos))
		fmt.Printf("   Total duration: %.2f seconds (%.2f minutes)\n", totalDuration, totalDuration/60)
		fmt.Printf("   Average duration: %.2f seconds\n", averageDuration)
		fmt.Printf("   Longest video: %s (%.2fs)\n", longestVideo.Metadata.FileName, longestVideo.Metadata.Duration)
		fmt.Printf("   Shortest video: %s (%.2fs)\n", shortestVideo.Metadata.FileName, shortestVideo.Metadata.Duration)
	}

	// 8. Video deletion (commented out for safety)
	fmt.Println("\n⚠️ Video deletion example (disabled for safety)...")
	fmt.Println("   To delete a video, uncomment the following code:")
	fmt.Println("   // err := client.Indexes.Videos.Delete(context.Background(), indexID, videoID)")
	fmt.Println("   // if err != nil {")
	fmt.Println("   //     log.Printf(\"Error deleting video: %v\", err)")
	fmt.Println("   // } else {")
	fmt.Println("   //     fmt.Printf(\"✅ Video deleted successfully\\n\")")
	fmt.Println("   // }")

	// Uncomment below to actually delete (BE CAREFUL!)
	/*
		if len(allVideos) > 0 {
			// Only delete if you're sure!
			videoToDelete := allVideos[len(allVideos)-1] // Delete the last video
			err := client.Indexes.Videos.Delete(context.Background(), indexID, videoToDelete.ID)
			if err != nil {
				log.Printf("Error deleting video: %v", err)
			} else {
				fmt.Printf("✅ Video %s deleted successfully\n", videoToDelete.ID)
			}
		}
	*/

	// 9. Video organization recommendations
	fmt.Println("\n💡 Video organization recommendations...")

	if len(allVideos) > 10 {
		fmt.Println("   📁 Consider organizing videos into categories using metadata")
		fmt.Println("   🏷️  Add consistent tagging for better searchability")
	}

	if totalDuration > 3600 { // More than 1 hour of content
		fmt.Println("   ⏱️  Large video collection detected - consider using time-based filters")
	}

	fmt.Println("   📝 Best practices:")
	fmt.Println("      - Use descriptive file names")
	fmt.Println("      - Add meaningful metadata tags")
	fmt.Println("      - Regularly review and update video information")
	fmt.Println("      - Use consistent naming conventions")

	fmt.Println("\n🎉 Video management examples completed!")
	fmt.Println("\nVideo operations demonstrated:")
	fmt.Println("- ✅ Listing videos with pagination")
	fmt.Println("- ✅ Retrieving detailed video information")
	fmt.Println("- ✅ Updating video metadata (single and bulk)")
	fmt.Println("- ✅ Filtering and analyzing video collections")
	fmt.Println("- ✅ Video statistics and insights")
	fmt.Println("- ✅ Safe deletion practices")
	fmt.Println("- ✅ Organization recommendations")
}
