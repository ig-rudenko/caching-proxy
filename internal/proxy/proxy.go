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
	cache  Cache
	origin *url.URL
}

func New(cache Cache, origin *url.URL) *Proxy {
	return &Proxy{cache, origin}
}

func (p *Proxy) Start(host string, port int) {
	http.HandleFunc("/", p.handleRequest)
	log.Printf("Starting caching proxy server on %s:%d, forwarding requests to %s\n", host, port, p.origin.String())

	if err := http.ListenAndServe(host+":"+strconv.Itoa(port), nil); err != nil {
		log.Fatalln("Error starting server:", err)
	}

}

func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	if isNotSafeMethod(r.Method) {
		w.Header().Set("X-Cache", "MISS")
		p.proxyRequest(w, r, false, "")
		return
	}

	requestURL := r.URL.String()
	cacheKey := getMD5Hash(requestURL)

	isCached := p.cache.Has(cacheKey)

	var headerXCacheValue string

	if !isCached {
		headerXCacheValue = "MISS"
		log.Printf("Cache miss for %s\n", cacheKey)
		w.Header().Set("x-cache", headerXCacheValue)
		p.proxyRequest(w, r, true, cacheKey)

	} else {
		headerXCacheValue = "HIT"
		w.Header().Set("x-cache", headerXCacheValue)
		p.responseFromCache(w, cacheKey)
	}
	log.Printf("Cache %s for URL: %s", headerXCacheValue, requestURL)
}

func (p *Proxy) hasRequest(urlHash string) bool {
	return p.cache.Has(urlHash) && p.cache.Has(urlHash+"-status") && p.cache.Has(urlHash+"-headers")
}

func (p *Proxy) responseFromCache(w http.ResponseWriter, cacheKey string) {
	data, _ := p.cache.Get(cacheKey)

	headers, ok := p.cache.GetHeaders(cacheKey + "-headers")
	if ok {
		for name := range *headers {
			w.Header().Set(name, headers.Get(name))
		}
	}
	status, ok := p.cache.GetInt(cacheKey + "-status")
	if ok {
		w.WriteHeader(status)
	}
	if data != nil {
		_, _ = w.Write(data)
	}
}

func (p *Proxy) proxyRequest(w http.ResponseWriter, r *http.Request, caching bool, cacheKey string) {
	resp, err := p.getRequestDataFromOrigin(r)
	if err != nil {
		http.Error(w, "Failed to fetch data from origin", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Закрываем тело ответа после завершения работы с ним
	// Читаем тело ответа в буфер
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %s", err)
		http.Error(w, "Failed to read response body", http.StatusInternalServerError)
		return
	}

	if caching {
		go p.cache.Set(cacheKey, respBody)
		go p.cache.SetInt(cacheKey+"-status", resp.StatusCode)
		go p.cache.SetHeaders(cacheKey+"-headers", &resp.Header)
	}

	// Устанавливаем заголовки ответа
	for name := range resp.Header {
		w.Header().Set(name, resp.Header.Get(name))
	}
	// Устанавливаем статус ответа
	w.WriteHeader(resp.StatusCode)
	// Устанавливаем тело ответа
	w.Write(respBody)

}

// getRequestDataFromOrigin отправляет запрос к оригинальному серверу и возвращает тело ответа
func (p *Proxy) getRequestDataFromOrigin(r *http.Request) (*http.Response, error) {

	//newUrl, _ := url.Parse(r.URL.String())
	//newUrl.Scheme = p.origin.Scheme
	//newUrl.Host = p.origin.Host

	newURL := *p.origin
	newURL.Path = r.URL.Path
	newURL.RawQuery = r.URL.RawQuery

	newReq, err := http.NewRequest(r.Method, newURL.String(), r.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = r.Header.Clone()

	client := &http.Client{}
	// Отправляем запрос
	resp, err := client.Do(newReq)
	if err != nil {
		log.Printf("Error reading response body: %s for URL %s", err, r.URL.String())
		return nil, err
	}

	return resp, nil

}

func isNotSafeMethod(method string) bool {
	method = strings.ToUpper(method)
	safeMethods := []string{http.MethodGet, http.MethodHead, http.MethodOptions}
	return !slices.Contains(safeMethods, method)
}

func getMD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
