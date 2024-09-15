package main

import (
	"caching-proxy/internal/argparser"
	"caching-proxy/internal/cache/filecache"
	"caching-proxy/internal/proxy"
	"os"
)

func main() {
	arg := argparser.New()
	arg.Parse()

	cache := filecache.New(arg.CacheTimeout, arg.CacheFolder)

	if arg.ClearCache {
		cache.ClearAll()
		os.Exit(0)
	}

	cache.RunCleanUp()

	p := proxy.New(cache, arg.Origin)
	p.SetUniqueByUser(arg.UniqueByUser)

	p.Start(arg.Host, arg.Port)

}
