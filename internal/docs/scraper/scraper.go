package scraper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/net/html"
	"devdocsmcp/internal/docs/indexer"
)

// Doc represents a single documentation entry.
type Doc struct {
	Name    string
	Version string
	URL     string
	// Add more fields as needed, e.g., local path, metadata
}

// Scraper is responsible for downloading and processing documentation.
type Scraper struct {
	DownloadPath string
	visitedURLs  map[string]bool
	mu           sync.Mutex
	wg           sync.WaitGroup
	Indexer      *indexer.Indexer // Add Indexer to Scraper
}

// NewScraper creates a new Scraper instance.
func NewScraper(downloadPath string, idx *indexer.Indexer) *Scraper {
	return &Scraper{
		DownloadPath: downloadPath,
		visitedURLs:  make(map[string]bool),
		Indexer:      idx,
	}
}

// DownloadDoc downloads a single documentation and its linked pages recursively.
func (s *Scraper) DownloadDoc(doc Doc, maxDepth int) error {
	fmt.Printf("Starting download for %s %s from %s (max depth: %d)\n", doc.Name, doc.Version, doc.URL, maxDepth)

	initialURL, err := url.Parse(doc.URL)
	if err != nil {
		return fmt.Errorf("invalid initial URL: %w", err)
	}
	initialHost := initialURL.Host

	queue := make(chan struct {
		url   string
		depth int
	}, 100) // Buffered channel for queue

	s.wg.Add(1) // Add for the initial URL
	go func() {
		queue <- struct {
			url   string
			depth int
		}{
			url: doc.URL, depth: 0,
		}
		s.wg.Wait()
		close(queue)
	}()

	for current := range queue {
		s.mu.Lock()
		if s.visitedURLs[current.url] {
			s.mu.Unlock()
			s.wg.Done() // Decrement for already visited URL
			continue
		}
		s.visitedURLs[current.url] = true
		s.mu.Unlock()

		if current.depth > maxDepth {
			s.wg.Done() // Decrement for exceeding max depth
			continue
		}

		go s.fetchAndProcess(current.url, current.depth, initialHost, doc.Name, doc.Version, queue)
	}

	return nil
}

func (s *Scraper) fetchAndProcess(currentURL string, currentDepth int, initialHost, docName, docVersion string, queue chan<- struct { url string; depth int }) {
	defer s.wg.Done()

	fmt.Printf("Downloading (depth %d): %s\n", currentDepth, currentURL)

	resp, err := http.Get(currentURL)
	if err != nil {
		fmt.Printf("Error downloading %s: %v\n", currentURL, err)
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close() // Close body immediately after reading
	if err != nil {
		fmt.Printf("Error reading response body for %s: %v\n", currentURL, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error downloading %s: status code %d\n", currentURL, resp.StatusCode)
		return
	}

	// Determine the local file path based on the URL
	parsedURL, err := url.Parse(currentURL)
	if err != nil {
		fmt.Printf("Error parsing URL %s: %v\n", currentURL, err)
		return
	}

	relativePath := parsedURL.Host + parsedURL.Path
	// Clean up path for file system, e.g., remove leading slashes, replace invalid chars
	relativePath = strings.TrimPrefix(relativePath, initialHost)
	relativePath = strings.TrimPrefix(relativePath, "/")
	relativePath = strings.ReplaceAll(relativePath, ":", "_") // Replace colon for Windows compatibility

	if strings.HasSuffix(relativePath, "/") || relativePath == "" {
		relativePath += "index.html"
	} else if filepath.Ext(relativePath) == "" {
		relativePath += ".html"
	}

	filePath := filepath.Join(s.DownloadPath, docName, docVersion, relativePath)
	dir := filepath.Dir(filePath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Printf("Error creating directory %s: %v\n", dir, err)
		return
	}

	err = ioutil.WriteFile(filePath, body, 0644)
	if err != nil {
		fmt.Printf("Error saving file %s: %v\n", filePath, err)
		return
	}

	fmt.Printf("Saved: %s\n", filePath)

	// Process the downloaded document (extract links, etc.)
	docReader := strings.NewReader(string(body))
	htmlDoc, err := html.Parse(docReader)
	if err != nil {
		fmt.Printf("Error parsing HTML from %s: %v\n", currentURL, err)
		return
	}

	// Extract text and add to index
	plainText := extractText(htmlDoc)
	// fmt.Printf("Extracted text for %s: %s\n", filePath, plainText[:min(len(plainText), 100)]) // Removed for brevity
	s.Indexer.AddDocument(filePath, plainText)

	links := extractLinks(htmlDoc, currentURL)
	for _, link := range links {
		parsedLink, err := url.Parse(link)
		if err != nil {
			continue // Skip invalid URLs
		}
		// Only follow links within the same domain
		if parsedLink.Host == initialHost {
			s.mu.Lock()
			if !s.visitedURLs[link] {
				s.wg.Add(1) // Increment for new URL to be processed
				queue <- struct {
					url   string
					depth int
				}{
					url:   link,
					depth: currentDepth + 1,
				}
			}
			s.mu.Unlock()
		}
	}
}

// extractLinks recursively extracts all 'href' attributes from 'a' tags and 'src' attributes from 'img' and 'script' tags.
func extractLinks(n *html.Node, baseURL string) []string {
	var links []string

	if n.Type == html.ElementNode {
		if n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					resolvedURL := resolveURL(baseURL, a.Val)
					if resolvedURL != "" {
						links = append(links, resolvedURL)
					}
				}
			}
		} else if n.Data == "img" || n.Data == "script" || n.Data == "link" {
			for _, a := range n.Attr {
				// For link tags, we are interested in href for stylesheets
				if (n.Data == "link" && a.Key == "href") || (n.Data != "link" && a.Key == "src") {
					resolvedURL := resolveURL(baseURL, a.Val)
					if resolvedURL != "" {
						links = append(links, resolvedURL)
					}
				}
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		links = append(links, extractLinks(c, baseURL)...)
	}

	return links
}

// extractText recursively extracts text content from HTML nodes.
func extractText(n *html.Node) string {
	var b strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		} 
		// Always recurse for element nodes, but skip script and style content
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return // Skip content of script and style tags
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return b.String()
}

// resolveURL resolves a relative URL against a base URL.
func resolveURL(baseURL, relativeURL string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	rel, err := url.Parse(relativeURL)
	if err != nil {
		return ""
	}
	return base.ResolveReference(rel).String()
}

// ListAvailableDocs would typically fetch a list of available documentations from a remote source.
// For now, it's a placeholder.
func (s *Scraper) ListAvailableDocs() ([]Doc, error) {
	// In a real scenario, this would parse a manifest from devdocs.io or a similar source.
	// For demonstration, return a dummy list.
	return []Doc{
		{Name: "html", Version: "5", URL: "https://devdocs.io/html/"},
		{Name: "css", Version: "3", URL: "https://devdocs.io/css/"},
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}