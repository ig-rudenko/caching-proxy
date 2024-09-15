package main

import (
	"caching-proxy/internal/argparser"
	"caching-proxy/internal/cache/filecache"
	"caching-proxy/internal/proxy"
	"os"
)

func main() {
	// Create a new ArgParser instance to handle command-line arguments
	arg := argparser.New()
	// Parse command-line arguments and set the corresponding fields in ArgParser
	arg.Parse()

	// Create a new Cache instance with the specified timeout and cache folder from ArgParser
	cache := filecache.New(arg.CacheTimeout, arg.CacheFolder)

	// If the --clear-cache flag was set, clear all cached data and exit the program
	if arg.ClearCache {
		cache.ClearAll()
		os.Exit(0)
	}

	// Start the cache cleanup process in a separate goroutine
	cache.RunCleanUp()

	// Create a new Proxy instance with the cache and origin URL from ArgParser
	p := proxy.New(cache, arg.Origin)
	// Set whether to generate unique cache per user based on User-Agent and cookies
	p.SetUniqueByUser(arg.UniqueByUser)

	// Start the proxy server on the specified host and port
	p.Start(arg.Host, arg.Port)
}
