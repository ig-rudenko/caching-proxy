package argparser

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"
)

type ArgParser struct {
	Host         string
	Port         int
	Origin       *url.URL
	UniqueByUser bool
	CacheTimeout time.Duration
	ClearCache   bool
	CacheFolder  string
}

func New() *ArgParser {
	return &ArgParser{}
}

func (a *ArgParser) Parse() {
	// Определяем флаги для порта, origin и справки
	var origin string
	flag.IntVar(&a.Port, "port", 0, "Port on which the caching proxy server will run.")
	flag.StringVar(&origin, "origin", "", "URL of the server to which the requests will be forwarded.")

	flag.BoolVar(&a.ClearCache, "clear-cache", false, "Clear the cache of proxy server.")

	flag.StringVar(&a.Host, "host", "0.0.0.0", "Host on which the caching proxy server will run. (default: 0.0.0.0)")
	flag.BoolVar(&a.UniqueByUser, "unique", false, "Generate unique cache per user (based on User-Agent or cookies). (default: false)")
	flag.DurationVar(&a.CacheTimeout, "cache-timeout", 0, "Duration to keep cached responses before expiration (e.g., 10s, 5m, 1h). (default: none)")

	flag.StringVar(&a.CacheFolder, "cache-folder", "./cache", "Directory to cache proxy server in. (default: \"./cache\")")

	help := flag.Bool("help", false, "Show help message.")
	h := flag.Bool("h", false, "Show help message.")

	// Парсим аргументы
	flag.Parse()

	if a.ClearCache {
		return
	}

	// Если указаны --help или -h, выводим справку и выходим
	if *help || *h {
		printUsage()
		os.Exit(0)
	}

	// Проверка корректности аргументов
	if a.Port == 0 || origin == "" {
		fmt.Println("Error: Missing required arguments.")
		printUsage()
		os.Exit(1)
	}

	// Проверка валидности порта
	if !isValidPort(&a.Port) {
		fmt.Printf("Error: Invalid port number %d. Port must be between 1 and 65535.\n", a.Port)
		printUsage()
		os.Exit(1)
	}

	// Проверка валидности URL
	validOriginURL, ok := getValidOriginURL(&origin)
	if !ok {
		fmt.Printf("Error: Invalid origin URL '%s'. Only protocol (http, https) and domain are allowed, no path, query or fragment.\n", origin)
		printUsage()
		os.Exit(1)
	}

	a.Origin = validOriginURL
}

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
  --clear-cache			   Clear the cache of proxy server and exit.
  -h, --help               Show this help message.`)
}

// isValidPort проверяет, что порт в диапазоне от 1 до 65535
func isValidPort(port *int) bool {
	return *port > 0 && *port <= 65535
}

// getValidOriginURL проверяет корректность URL и что он состоит только из протокола и домена
func getValidOriginURL(origin *string) (*url.URL, bool) {
	parsedURL, err := url.ParseRequestURI(*origin)
	if err != nil {
		return nil, false
	}

	// Проверяем, что URL валиден, имеет правильный протокол и хост, и отсутствуют путь, параметры и фрагменты
	if parsedURL.Scheme == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return nil, false
	}
	if parsedURL.Host == "" || parsedURL.Path != "" || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, false
	}

	return parsedURL, true
}
