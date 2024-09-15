package argparser

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"
)

// ArgParser manages command-line arguments for configuring the caching proxy server
type ArgParser struct {
	Host         string        // Host address where the proxy server will listen
	Port         int           // Port number where the proxy server will listen
	Origin       *url.URL      // URL of the origin server to which requests will be forwarded
	UniqueByUser bool          // Whether to generate unique cache keys per user based on User-Agent and cookies
	CacheTimeout time.Duration // Duration to keep cached responses before they expire
	ClearCache   bool          // Flag to indicate if the cache should be cleared
	CacheFolder  string        // Directory to store cached data
}

// New creates a new ArgParser instance
func New() *ArgParser {
	return &ArgParser{}
}

// Parse processes command-line arguments and sets the corresponding fields in ArgParser
func (a *ArgParser) Parse() {
	// Define flags for port, origin, and help
	var origin string
	flag.IntVar(&a.Port, "port", 0, "Port on which the caching proxy server will run.")
	flag.StringVar(&origin, "origin", "", "URL of the server to which the requests will be forwarded.")

	flag.BoolVar(&a.ClearCache, "clear-cache", false, "Clear the cache of the proxy server.")

	flag.StringVar(&a.Host, "host", "0.0.0.0", "Host on which the caching proxy server will run. (default: 0.0.0.0)")
	flag.BoolVar(&a.UniqueByUser, "unique", false, "Generate unique cache per user (based on User-Agent or cookies). (default: false)")
	flag.DurationVar(&a.CacheTimeout, "cache-timeout", 0, "Duration to keep cached responses before expiration (e.g., 10s, 5m, 1h). (default: none)")

	flag.StringVar(&a.CacheFolder, "cache-folder", "./cache", "Directory to cache proxy server in. (default: \"./cache\")")

	// Define flags for displaying help
	help := flag.Bool("help", false, "Show help message.")
	h := flag.Bool("h", false, "Show help message.")

	// Parse command-line arguments
	flag.Parse()

	if a.ClearCache {
		// If --clear-cache flag is set, exit after clearing the cache
		return
	}

	// Display help message if --help or -h flag is set
	if *help || *h {
		printUsage()
		os.Exit(0)
	}

	// Validate required arguments
	if a.Port == 0 || origin == "" {
		fmt.Println("Error: Missing required arguments.")
		printUsage()
		os.Exit(1)
	}

	// Validate port number
	if !isValidPort(&a.Port) {
		fmt.Printf("Error: Invalid port number %d. Port must be between 1 and 65535.\n", a.Port)
		printUsage()
		os.Exit(1)
	}

	// Validate origin URL
	validOriginURL, ok := getValidOriginURL(&origin)
	if !ok {
		fmt.Printf("Error: Invalid origin URL '%s'. Only protocol (http, https) and domain are allowed, no path, query, or fragment.\n", origin)
		printUsage()
		os.Exit(1)
	}

	// Set the validated origin URL
	a.Origin = validOriginURL
}

// printUsage displays the usage instructions for the command-line arguments
func printUsage() {
	fmt.Println(`Usage: caching-proxy --port <number> --origin <url> [options]

Required:
  --port <number>          Port on which the caching proxy server will run.
  --origin <url>           URL of the server to which the requests will be forwarded.

Options:
  --host <string>          Host on which the caching proxy server will run. (default: 0.0.0.0)
  --unique                 Generate unique cache per user (based on User-Agent or cookies). (default: false)
  --cache-timeout <time>   Duration to keep cached responses before expiration (e.g., 10s, 5m, 1h). (default: none)
  --cache-folder <string>  Directory to cache proxy server in. (default: "./cache")
  --clear-cache            Clear the cache of the proxy server and exit.
  -h, --help               Show this help message.`)
}

// isValidPort checks if the port number is within the valid range (1 to 65535)
func isValidPort(port *int) bool {
	return *port > 0 && *port <= 65535
}

// getValidOriginURL validates that the origin URL consists only of protocol and domain, without path, query, or fragment
func getValidOriginURL(origin *string) (*url.URL, bool) {
	// Parse the origin URL
	parsedURL, err := url.ParseRequestURI(*origin)
	if err != nil {
		return nil, false
	}

	// Ensure the URL has a valid scheme (http or https), a host, and no path, query, or fragment
	if parsedURL.Scheme == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, false
	}
	if parsedURL.Host == "" || parsedURL.Path != "" || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, false
	}

	return parsedURL, true
}
