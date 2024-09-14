package argparser

import (
	"flag"
	"fmt"
	"net/url"
	"os"
)

type ArgParser struct {
	Host   string
	Port   int
	Origin *url.URL
}

func New() *ArgParser {
	return &ArgParser{
		Host: "0.0.0.0",
	}
}

func (a *ArgParser) Parse() {
	// Определяем флаги для порта, origin и справки
	var origin string
	flag.IntVar(&a.Port, "port", 0, "Port on which the caching proxy server will run.")
	flag.StringVar(&origin, "origin", "", "URL of the server to which the requests will be forwarded.")
	help := flag.Bool("help", false, "Show help message.")
	h := flag.Bool("h", false, "Show help message.")

	// Парсим аргументы
	flag.Parse()

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
	fmt.Println(`Usage: caching-proxy --port <number> --origin <url>

Options:
  --port <number>   Port on which the caching proxy server will run.
  --origin <url>    URL of the server to which the requests will be forwarded.
  -h, --help        Show this help message.`)
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
