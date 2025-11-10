package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"wget/internal/crawl"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <URL> [depth] [concurrency] [outputDir]")
		return
	}
	startURL := os.Args[1]
	maxDepth := 3
	if len(os.Args) > 2 {
		var err error
		maxDepth, err = strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("Invalid depth: %v", err)
		}
	}
	concurrency := 5
	if len(os.Args) > 3 {
		var err error
		concurrency, err = strconv.Atoi(os.Args[3])
		if err != nil {
			log.Fatalf("Invalid concurrency: %v", err)
		}
	}
	outputDir := "./mirror"
	if len(os.Args) > 4 {
		outputDir = os.Args[4]
	}
	crawler, err := crawl.NewCrawler(startURL, maxDepth, concurrency, outputDir)
	if err != nil {
		log.Fatal(err)
	}
	crawler.Wg.Add(1)
	go func() {
		crawler.Sem <- struct{}{}
		defer func() { <-crawler.Sem }()
		crawler.Crawl(startURL, 0)
	}()
	crawler.Wg.Wait()
	log.Println("Mirroring completed.")
}
