package main

import (
	"caching-proxy/internal/argparser"
	"caching-proxy/internal/cache/filecache"
	"caching-proxy/internal/proxy"
)

func main() {
	cache := filecache.New("./cache")

	arg := argparser.New()
	arg.Parse()

	p := proxy.New(cache, arg.Origin)
	p.Start(arg.Host, arg.Port)

}
