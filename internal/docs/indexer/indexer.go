package indexer

import (
	"fmt"

	"github.com/blevesearch/bleve/v2"
)

// Indexer stores an inverted index for searching using Bleve.
type Indexer struct {
	index bleve.Index
}

// NewIndexer creates a new Indexer instance.
func NewIndexer(indexPath string) (*Indexer, error) {
	// Create a new mapping
	indexMapping := bleve.NewIndexMapping()

	// Create a document mapping for the default type
	docMapping := bleve.NewDocumentMapping()

	// Add a text field mapping for content with English analyzer
	textFieldMapping := bleve.NewTextFieldMapping()
	textFieldMapping.Analyzer = "en"
	docMapping.AddFieldMappingsAt("Content", textFieldMapping)

	indexMapping.AddDocumentMapping("", docMapping) // Add to default type

	// Create a new index
	index, err := bleve.New(indexPath, indexMapping)
	if err != nil {
		if err == bleve.ErrorIndexPathExists {
			// If index already exists, open it
			index, err = bleve.Open(indexPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open existing index: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to create new index: %w", err)
		}
	}

	fmt.Printf("Bleve index opened/created at %s\n", indexPath)
	return &Indexer{
		index: index,
	}, nil
}

// AddDocument adds a document's content to the index.
func (i *Indexer) AddDocument(filePath, content string) error {
	data := struct {
		Path    string
		Content string
	}{
		Path:    filePath,
		Content: content,
	}

	err := i.index.Index(filePath, data)
	if err != nil {
		return fmt.Errorf("failed to index document %s: %w", filePath, err)
	}
	return nil
}

// Search searches the index for a given query and returns matching file paths.
func (i *Indexer) Search(query string) ([]string, error) {
	queryRequest := bleve.NewSearchRequest(bleve.NewQueryStringQuery(query))
	searchResult, err := i.index.Search(queryRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search index: %w", err)
	}

	var matchingPaths []string
	for _, hit := range searchResult.Hits {
		matchingPaths = append(matchingPaths, hit.ID)
	}

	return matchingPaths, nil
}

// SearchFuzzy performs a fuzzy search on the index.
func (i *Indexer) SearchFuzzy(query string) ([]string, error) {
	queryRequest := bleve.NewSearchRequest(bleve.NewFuzzyQuery(query))
	searchResult, err := i.index.Search(queryRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to fuzzy search index: %w", err)
	}

	var matchingPaths []string
	for _, hit := range searchResult.Hits {
		matchingPaths = append(matchingPaths, hit.ID)
	}

	return matchingPaths, nil
}

// Close closes the Bleve index.
func (i *Indexer) Close() error {
	return i.index.Close()
}