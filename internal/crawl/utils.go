package crawl

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func (c *Crawler) allowedByRobots(u *url.URL) bool {
	robotsURL := c.BaseURL.Scheme + "://" + c.BaseURL.Host + "/robots.txt"
	resp, err := c.Client.Get(robotsURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Error fetching robots.txt or non-OK status: %v", err)
		return true
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(body), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Disallow:") {
			disallow := strings.TrimSpace(strings.TrimPrefix(line, "Disallow:"))
			if strings.HasPrefix(u.Path, disallow) {
				return false
			}
		}
	}
	return true
}
