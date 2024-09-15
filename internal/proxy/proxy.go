package proxy

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

type Cache interface {
	Has(string) bool
	Get(string) ([]byte, bool)
	GetInt(string) (int, bool)
	GetHeaders(string) (*http.Header, bool)
	Set(string, []byte) error
	SetInt(string, int) error
	SetHeaders(string, *http.Header) error
}

type Proxy struct {
	cache        Cache    // The cache implementation used by the proxy
	origin       *url.URL // The origin server to which requests are forwarded
	uniqueByUser bool     // Determines whether to create unique cache keys per user
}

// New creates a new Proxy instance with the specified cache and origin server URL
func New(cache Cache, origin *url.URL) *Proxy {
	return &Proxy{cache, origin, false}
}

// SetUniqueByUser sets whether cache keys should be unique per user based on User-Agent and cookies
func (p *Proxy) SetUniqueByUser(is bool) {
	p.uniqueByUser = is
}

// Start starts the proxy server on the specified host and port
func (p *Proxy) Start(host string, port int) {
	http.HandleFunc("/", p.handleRequest)
	log.Printf("Starting caching proxy server on %s:%d, forwarding requests to %s\n", host, port, p.origin.String())

	if err := http.ListenAndServe(host+":"+strconv.Itoa(port), nil); err != nil {
		log.Fatalln("Error starting server:", err)
	}
}

// handleRequest processes incoming HTTP requests
func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	if isNotSafeMethod(r.Method) {
		// For non-safe methods, always bypass cache
		w.Header().Set("X-Cache", "MISS")
		p.proxyRequest(w, r, false, "")
		return
	}

	// Generate a cache key based on the request
	cacheKey := p.getRequestCacheKey(r)
	isCached := p.hasRequestInCache(cacheKey)

	var headerXCacheValue string

	if !isCached {
		// If the request is not in cache, forward it and cache the response
		headerXCacheValue = "MISS"
		w.Header().Set("X-Cache", headerXCacheValue)
		p.proxyRequest(w, r, true, cacheKey)
	} else {
		// If the request is in cache, serve the cached response
		headerXCacheValue = "HIT"
		w.Header().Set("X-Cache", headerXCacheValue)
		p.responseFromCache(w, cacheKey)
	}

	log.Printf("Cache %s for URL: %s", headerXCacheValue, r.URL.String())
}

// getRequestCacheKey generates a cache key based on the request URL, method, and optionally User-Agent and cookies
func (p *Proxy) getRequestCacheKey(r *http.Request) string {
	// Assemble the cache key from URL, method, headers (User-Agent and Cookie)
	var keyParts []string

	// Add URL to the key parts
	keyParts = append(keyParts, r.URL.String())

	if p.uniqueByUser {
		// If unique per user, include User-Agent in the key
		userAgent := r.Header.Get("User-Agent")
		if userAgent != "" {
			keyParts = append(keyParts, userAgent)
		}

		// Include cookies in the key if present
		if cookies := r.Header.Get("Cookie"); cookies != "" {
			keyParts = append(keyParts, cookies)
		}
	}

	// Join all parts to form the raw key
	rawKey := strings.Join(keyParts, "|")

	// Hash the raw key using MD5 and return it as a hexadecimal string
	hash := md5.Sum([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}

// hasRequestInCache checks if the cache contains entries for the given key and associated metadata
func (p *Proxy) hasRequestInCache(key string) bool {
	return p.cache.Has(key) && p.cache.Has(key+"-status") && p.cache.Has(key+"-headers")
}

// responseFromCache serves the cached response for the given cache key
func (p *Proxy) responseFromCache(w http.ResponseWriter, cacheKey string) {
	// Retrieve cached data
	data, _ := p.cache.Get(cacheKey)

	// Retrieve cached headers and set them in the response
	headers, ok := p.cache.GetHeaders(cacheKey + "-headers")
	if ok {
		for name := range *headers {
			w.Header().Set(name, headers.Get(name))
		}
	}

	// Retrieve cached status and set it in the response
	status, ok := p.cache.GetInt(cacheKey + "-status")
	if ok {
		w.WriteHeader(status)
	}

	// Write cached data to the response
	if data != nil {
		_, _ = w.Write(data)
	}
}

// proxyRequest forwards the request to the origin server, handles caching if required, and writes the response
func (p *Proxy) proxyRequest(w http.ResponseWriter, r *http.Request, caching bool, cacheKey string) {
	// Get response from the origin server
	resp, err := p.getResponseFromOrigin(r)
	if err != nil {
		http.Error(w, "Failed to fetch data from origin", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read the response body into a buffer
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %s", err)
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	if caching {
		// Cache the response data, status, and headers asynchronously
		go p.cache.Set(cacheKey, respBody)
		go p.cache.SetInt(cacheKey+"-status", resp.StatusCode)
		go p.cache.SetHeaders(cacheKey+"-headers", &resp.Header)
	}

	// Set response headers and status
	for name := range resp.Header {
		w.Header().Set(name, resp.Header.Get(name))
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

// getResponseFromOrigin sends a request to the origin server and returns the response
func (p *Proxy) getResponseFromOrigin(r *http.Request) (*http.Response, error) {
	// Construct the new URL for the origin server
	newURL := *p.origin
	newURL.Path = r.URL.Path
	newURL.RawQuery = r.URL.RawQuery

	// Create a new request with the same method, URL, and headers as the original request
	newReq, err := http.NewRequest(r.Method, newURL.String(), r.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = r.Header.Clone()

	// Create an HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(newReq)
	if err != nil {
		log.Printf("Error reading response body: %s for URL %s", err, r.URL.String())
		return nil, err
	}

	return resp, nil
}

// isNotSafeMethod checks if the HTTP method is not one of the safe methods (GET, HEAD, OPTIONS)
func isNotSafeMethod(method string) bool {
	method = strings.ToUpper(method)
	safeMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	return !slices.Contains(safeMethods, method)
}
