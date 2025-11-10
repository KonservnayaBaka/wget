package crawl

import (
	"bytes"
	"log"
	"net/url"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

func (c *Crawler) parseAndRewriteHTML(body []byte, base *url.URL, depth int) ([]byte, []string) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		log.Printf("Error parsing HTML: %v", err)
		return body, nil
	}
	var resources []string
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "a", "link":
				for i, attr := range n.Attr {
					if attr.Key == "href" {
						resURL := base.ResolveReference(&url.URL{Path: attr.Val}).String()
						if strings.HasPrefix(resURL, "http") {
							resources = append(resources, resURL)
							local, err := filepath.Rel(filepath.Dir(c.getLocalPath(base)), c.getLocalPath(base.ResolveReference(&url.URL{Path: attr.Val})))
							if err != nil {
								log.Printf("Error calculating relative path: %v", err)
							}
							n.Attr[i].Val = local
						}
					}
				}
			case "img", "script", "iframe":
				for i, attr := range n.Attr {
					if attr.Key == "src" {
						resURL := base.ResolveReference(&url.URL{Path: attr.Val}).String()
						if strings.HasPrefix(resURL, "http") {
							resources = append(resources, resURL)
							local, err := filepath.Rel(filepath.Dir(c.getLocalPath(base)), c.getLocalPath(base.ResolveReference(&url.URL{Path: attr.Val})))
							if err != nil {
								log.Printf("Error calculating relative path: %v", err)
							}
							n.Attr[i].Val = local
						}
					}
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}
	traverse(doc)
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		log.Printf("Error rendering HTML: %v", err)
		return body, resources
	}
	return buf.Bytes(), resources
}
