package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/sirupsen/logrus"
)

const (
	docsBaseURL = "https://documents.devdocs.io/"
)

// DocEntry represents a single entry within a documentation set
type DocEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Doc represents a documentation index (from index.json)
type Doc struct {
	Name    string     `json:"name"`
	Version string     `json:"version"`
	Entries []DocEntry `json:"entries"`
}

var allowedLanguages map[string]bool

func main() {
	// Define subcommands
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	searchLang := searchCmd.String("lang", "", "Language slug to search within (e.g., html, angularjs~1.8)")
	searchQuery := searchCmd.String("query", "", "Search query")

	readCmd := flag.NewFlagSet("read", flag.ExitOnError)
	readLang := readCmd.String("lang", "", "Language slug to read from")
	readPath := readCmd.String("path", "", "Path to the documentation entry (e.g., reference/elements/a)")
	
	serverCmd := flag.NewFlagSet("server", flag.ExitOnError)
	serverPort := serverCmd.String("port", "8080", "Port for the HTTP server to listen on")
	serverLangs := serverCmd.String("lang", "", "Comma-separated list of language slugs to serve (e.g., html,css)")

	allowedLangsCmd := flag.NewFlagSet("allowed-langs", flag.ExitOnError)

	// Parse the main command-line arguments
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "search":
		searchCmd.Parse(os.Args[2:])
		if *searchLang == "" || *searchQuery == "" {
			log.Fatal("Error: -lang and -query are required for search command.")
		}
		searchResults, err := SearchDoc(*searchLang, *searchQuery)
		if err != nil {
			log.Printf("Error searching docs: %v\n", err)
		} else if len(searchResults) == 0 {
			fmt.Println("No results found.")
		} else {
			fmt.Printf("Search results for '%s' in %s:\n", *searchQuery, *searchLang)
			for _, entry := range searchResults {
				fmt.Printf("  - %s (Path: %s)\n", entry.Name, entry.Path)
			}
		}
	case "read":
		readCmd.Parse(os.Args[2:])
		if *readLang == "" || *readPath == "" {
			log.Fatal("Error: -lang and -path are required for read command.")
		}
		content, err := ReadDocContent(*readLang, *readPath)
		if err != nil {
			log.Printf("Error reading doc content: %v\n", err)
		} else {
			fmt.Printf("Content for %s/%s:\n", *readLang, *readPath)
			// Print only a snippet to avoid flooding the console
			fmt.Printf("\n--- Content Snippet ---\n%s\n...\n", content[:500])
		}
	case "server":
		serverCmd.Parse(os.Args[2:])
		if *serverLangs == "" {
			log.Fatal("Error: -lang is required for the server command. Please specify a comma-separated list of languages.")
		}
		initAllowedLanguages(*serverLangs)
		startMcpServer(*serverPort)
	case "allowed-langs":
		allowedLangsCmd.Parse(os.Args[2:])
		// This command is meant to be run after the server has been configured with --lang
		// However, for a standalone command, we need to re-initialize allowedLanguages
		// based on a potential flag, or just print the current state if run without server
		// For simplicity, we'll assume it's run in a context where allowedLanguages is set
		// or we'll print a message if it's not.
		if allowedLanguages == nil {
			fmt.Println("No specific languages configured. All languages are allowed.")
		} else if len(allowedLanguages) == 0 {
			fmt.Println("No languages are explicitly allowed.")
		} else {
			fmt.Println("Allowed Languages:")
			for lang := range allowedLanguages {
				fmt.Printf("- %s\n", lang)
			}
		}
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage: devdocsmcp <command> [arguments]")
	fmt.Println("Commands:")
	fmt.Println("  search   -lang <language_slug> -query <search_query>")
	fmt.Println("  read     -lang <language_slug> -path <entry_path>")
	fmt.Println("  server   [-port <port_number>] -lang <comma_separated_languages> (starts HTTP server)")
	fmt.Println("  allowed-langs (displays languages allowed by the server configuration)")
}

func initAllowedLanguages(langs string) {
	allowedLanguages = make(map[string]bool)
	if langs == "" {
		// This case should now be caught by the flag parsing in main, but as a safeguard
		log.Fatal("Error: initAllowedLanguages called with empty language list.")
	}
	for _, lang := range strings.Split(langs, ",") {
		allowedLanguages[strings.TrimSpace(lang)] = true
	}
	log.Printf("Server will serve documentation for languages: %v\n", strings.Split(langs, ","))
}

func isLanguageAllowed(lang string) bool {
	if allowedLanguages == nil { // This case should ideally not be reached if initAllowedLanguages enforces non-empty
		return true // Fallback: if somehow not initialized, allow all
	}
	return allowedLanguages[lang]
}

func startMcpServer(port string) {
	log.Printf("Starting DevDocsMCP server on port %s...\n", port)

	s := server.NewMCPServer(
		"DevDocs MCP",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// Define and add the search_doc tool
	searchDocTool := mcp.NewTool("search_doc",
		mcp.WithDescription("Searches for a query within the documentation entries of a specific language."),
		mcp.WithString("lang",
			mcp.Required(),
			mcp.Description("The language slug (e.g., html, angularjs~1.8)."),
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query."),
		),
	)
	s.AddTool(searchDocTool, handleSearchDoc)

	// Define and add the read_doc_content tool
	readDocContentTool := mcp.NewTool("read_doc_content",
		mcp.WithDescription("Reads the content of a specific documentation HTML file."),
		mcp.WithString("lang",
			mcp.Required(),
			mcp.Description("The language slug (e.g., html, angularjs~1.8)."),
		),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("The path to the documentation entry (e.g., reference/elements/a)."),
		),
	)
	s.AddTool(readDocContentTool, handleReadDocContent)

	// Start the server in Stdio mode (as per MCP server configuration)
	if err := server.ServeStdio(s); err != nil {
		logrus.Printf("Server error: %v", err)
	}
}

func handleSearchDoc(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lang, err := request.RequireString("lang")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if !isLanguageAllowed(lang) {
		return mcp.NewToolResultError(fmt.Sprintf("Language '%s' is not allowed by this server configuration.", lang)), nil
	}

	results, err := SearchDoc(lang, query)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	jsonResults, err := json.Marshal(results)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(string(jsonResults)), nil
}

func handleReadDocContent(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	lang, err := request.RequireString("lang")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if !isLanguageAllowed(lang) {
		return mcp.NewToolResultError(fmt.Sprintf("Language '%s' is not allowed by this server configuration.", lang)), nil
	}

	content, err := ReadDocContent(lang, path)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(content), nil
}

// fetchIndex fetches the index.json for a given language slug.
func fetchIndex(langSlug string) (*Doc, error) {
	indexURL := fmt.Sprintf("%s%s/index.json", docsBaseURL, langSlug)
	log.Printf("Fetching index.json from: %s\n", indexURL)
	resp, err := http.Get(indexURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch index.json for %s: %w", langSlug, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch index.json for %s: status code %d - %s", langSlug, resp.StatusCode, resp.Status)
	}

	var doc Doc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to decode index.json for %s: %w", langSlug, err)
	}
	return &doc, nil
}

// SearchDoc searches for a query within the documentation entries of a specific language.
func SearchDoc(langSlug, query string) ([]DocEntry, error) {
	var results []DocEntry

	doc, err := fetchIndex(langSlug)
	if err != nil {
		return nil, err
	}

	lowerQuery := strings.ToLower(query)

	for _, entry := range doc.Entries {
		if strings.Contains(strings.ToLower(entry.Name), lowerQuery) || strings.Contains(strings.ToLower(entry.Path), lowerQuery) {
			results = append(results, entry)
		}
	}

	return results, nil
}

// ReadDocContent reads the content of a specific documentation HTML file.
func ReadDocContent(langSlug, entryPath string) (string, error) {
	contentURL := fmt.Sprintf("%s%s/%s.html", docsBaseURL, langSlug, entryPath)
	log.Printf("Fetching content from: %s\n", contentURL)
	resp, err := http.Get(contentURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch doc content from %s: %w", contentURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch doc content from %s: status code %d - %s", contentURL, resp.StatusCode, resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from %s: %w", contentURL, err)
	}

	return string(data), nil
}
