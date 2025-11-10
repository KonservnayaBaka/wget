package crawl

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Crawler struct {
	Client    *http.Client
	BaseURL   *url.URL
	OutputDir string
	Visited   sync.Map
	MaxDepth  int
	Sem       chan struct{}
	Wg        sync.WaitGroup
}

func NewCrawler(startURL string, maxDepth int, concurrency int, outputDir string) (*Crawler, error) {
	u, err := url.Parse(startURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}
	return &Crawler{
		Client:    client,
		BaseURL:   u,
		OutputDir: outputDir,
		MaxDepth:  maxDepth,
		Sem:       make(chan struct{}, concurrency),
	}, nil
}

func (c *Crawler) Crawl(currentURL string, depth int) {
	defer c.Wg.Done()
	if depth > c.MaxDepth {
		return
	}
	u, err := url.Parse(currentURL)
	if err != nil {
		log.Printf("Error parsing URL %s: %v", currentURL, err)
		return
	}
	u = c.BaseURL.ResolveReference(u)
	if u.Host != c.BaseURL.Host {
		return
	}
	if !c.allowedByRobots(u) {
		log.Printf("URL %s disallowed by robots.txt", u.String())
		return
	}
	key := u.String()
	if _, loaded := c.Visited.LoadOrStore(key, true); loaded {
		return
	}
	resp, err := c.Client.Get(key)
	if err != nil {
		log.Printf("Error fetching %s: %v", key, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Non-OK status %d for %s", resp.StatusCode, key)
		return
	}
	localPath := c.getLocalPath(u)
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Error creating directory %s: %v", dir, err)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading body for %s: %v", key, err)
		return
	}
	contentType := resp.Header.Get("Content-Type")
	var resources []string
	if strings.Contains(contentType, "text/html") {
		modifiedBody, res := c.parseAndRewriteHTML(body, u, depth)
		body = modifiedBody
		resources = res
	}
	if err := os.WriteFile(localPath, body, 0644); err != nil {
		log.Printf("Error saving %s: %v", localPath, err)
	}
	log.Printf("Downloaded and saved: %s -> %s", key, localPath)
	for _, resURL := range resources {
		c.Wg.Add(1)
		go func(res string) {
			c.Sem <- struct{}{}
			defer func() { <-c.Sem }()
			c.Crawl(res, depth+1)
		}(resURL)
	}
}

func (c *Crawler) getLocalPath(u *url.URL) string {
	p := u.Path
	if p == "" || p == "/" {
		p = "/index.html"
	} else if !strings.Contains(path.Base(p), ".") {
		p = path.Join(p, "index.html")
	}
	return filepath.Join(c.OutputDir, u.Host, p)
}
