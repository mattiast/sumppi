package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: sumppi <series-guid>")
		os.Exit(1)
	}

	guid := os.Args[1]

	fmt.Printf("Fetching series data for GUID: %s\n", guid)
	seriesData, err := fetchSeriesData(guid)
	if err != nil {
		log.Fatalf("Error fetching series data: %v", err)
	}

	fmt.Printf("Series: %s by %s\n", seriesData.Title, seriesData.Author)
	fmt.Printf("Episodes: %d\n", len(seriesData.Episodes))

	rssXML, err := generateRSSFeed(seriesData)
	if err != nil {
		log.Fatalf("Error generating RSS feed: %v", err)
	}

	filename := fmt.Sprintf("%s.rss", guid)
	err = os.WriteFile(filename, []byte(rssXML), 0644)
	if err != nil {
		log.Fatalf("Error writing RSS file: %v", err)
	}

	fmt.Printf("RSS feed written to %s\n", filename)
}
