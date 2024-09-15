package filecache

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Cache struct {
	timeout    time.Duration // Duration before cache entries expire
	folderPath string        // Directory where cache files are stored
}

// New creates a new Cache instance with the specified timeout and folder path
func New(timeout time.Duration, folderPath string) *Cache {
	c := &Cache{timeout, folderPath}
	c.createCacheDir()
	return c
}

// Has checks if a cache entry exists for the given key
func (c *Cache) Has(key string) bool {
	c.deleteCacheByExpiration(key)
	filePath := c.getFilePath(key)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

// GetInt retrieves an integer value from the cache for the given key
func (c *Cache) GetInt(key string) (int, bool) {
	data, ok := c.Get(key)
	if !ok {
		return 0, false
	}

	// Convert []byte to string and then to an integer
	intValue, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, false
	}

	return intValue, true
}

// GetHeaders retrieves HTTP headers from the cache for the given key
func (c *Cache) GetHeaders(key string) (*http.Header, bool) {
	data, ok := c.Get(key)
	if !ok {
		return nil, false
	}

	headers := make(http.Header)
	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue // Skip empty lines
		}
		// Split the line into header name and value
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			return nil, false
		}
		name, value := parts[0], parts[1]
		headers.Add(name, value)
	}

	if err := scanner.Err(); err != nil {
		return nil, false
	}

	return &headers, true
}

// Get retrieves raw data from the cache for the given key
func (c *Cache) Get(key string) ([]byte, bool) {
	c.deleteCacheByExpiration(key)

	// Check if the file exists
	filePath := c.getFilePath(key)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// If the file does not exist, return empty []byte and false
		return []byte{}, false
	}

	// Read the file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		// If there is an error reading the file, return empty []byte and false
		return []byte{}, false
	}

	// Return file content and true
	return data, true
}

// SetInt stores an integer value in the cache with the given key
func (c *Cache) SetInt(key string, value int) error {
	return c.Set(key, []byte(strconv.Itoa(value)))
}

// SetHeaders stores HTTP headers in the cache with the given key
func (c *Cache) SetHeaders(key string, headers *http.Header) error {
	var buf bytes.Buffer

	// Iterate over all headers and add them to the buffer
	for name, values := range *headers {
		for _, value := range values {
			buf.WriteString(fmt.Sprintf("%s: %s\n", name, value))
		}
	}
	return c.Set(key, buf.Bytes())
}

// Set stores raw data in the cache with the given key
func (c *Cache) Set(key string, value []byte) error {
	filePath := c.getFilePath(key)

	// Create a file with read and write permissions (rw-r--r--)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error adding to cache")
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// Write data to the file
	_, err = file.Write(value)
	if err != nil {
		return err
	}

	return nil
}

// RunCleanUp starts a goroutine for periodic cleanup of old cache files
func (c *Cache) RunCleanUp() {
	go c.cleanUpOldFiles()
}

// cleanUpOldFiles checks files in the directory and removes those older than the timeout
func (c *Cache) cleanUpOldFiles() {
	if c.timeout <= 0 {
		return
	}

	for {
		// Iterate over all files in the directory
		err := filepath.Walk(c.folderPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Check if it is a file (not a directory)
			if !info.IsDir() {
				// If the file was modified longer than timeout ago, remove it
				if time.Since(info.ModTime()) > c.timeout {
					log.Printf("Removing old file: %s\n", path)
					if err := os.Remove(path); err != nil {
						log.Printf("Error removing file: %s\n", err)
					}
				}
			}
			return nil
		})

		if err != nil {
			log.Printf("Error walking through directory: %s\n", err)
		}

		// Wait before the next cleanup run
		time.Sleep(c.timeout)
	}
}

// deleteCacheByExpiration removes cache entries that are older than the timeout
func (c *Cache) deleteCacheByExpiration(key string) {
	if c.timeout <= 0 {
		return
	}

	for _, cacheKey := range []string{key, key + "-status", key + "-headers"} {
		filePath := c.getFilePath(cacheKey)
		stats, err := os.Stat(filePath)
		if err != nil {
			return
		}

		if time.Since(stats.ModTime()) > c.timeout {
			_ = os.Remove(filePath)
		}
	}
}

// ClearAll removes all files and directories in the cache folder
func (c *Cache) ClearAll() {
	// Get a list of all files and directories in the folder
	files, err := os.ReadDir(c.folderPath)
	if err != nil {
		log.Fatalf("failed to read directory: %w", err)
	}

	// Iterate over each item and remove it
	for _, file := range files {
		filePath := filepath.Join(c.folderPath, file.Name())
		err := os.RemoveAll(filePath) // Remove file or directory recursively
		if err != nil {
			log.Printf("failed to remove %s: %s", filePath, err)
		}
	}
}

// getFilePath generates the file path for the given cache key
func (c *Cache) getFilePath(key string) string {
	return c.folderPath + "/" + key
}

// createCacheDir creates the cache directory with permissions 0755 (read/write for owner, read for group and others)
func (c *Cache) createCacheDir() {
	err := os.MkdirAll(c.folderPath, 0755)
	if err != nil {
		log.Fatalf("failed to create cache directory: %s\n", err)
	}
}
