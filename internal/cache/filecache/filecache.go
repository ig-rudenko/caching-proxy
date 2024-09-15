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
	timeout    time.Duration
	folderPath string
}

func New(timeout time.Duration, folderPath string) *Cache {
	c := &Cache{timeout, folderPath}
	c.createCacheDir()
	return c
}

func (c *Cache) Has(key string) bool {
	c.deleteCacheByExpiration(key)
	filePath := c.getFilePath(key)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func (c *Cache) GetInt(key string) (int, bool) {
	data, ok := c.Get(key)
	if !ok {
		return 0, false
	}

	// Преобразуем []byte в строку, а затем в число
	intValue, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, false
	}

	return intValue, true
}

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
			continue // Пропускаем пустые строки
		}
		// Разделяем строку на имя и значение заголовка
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

func (c *Cache) Get(key string) ([]byte, bool) {
	c.deleteCacheByExpiration(key)

	// Проверяем, существует ли файл
	filePath := c.getFilePath(key)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Если файл не существует, возвращаем пустой []byte и статус false
		return []byte{}, false
	}

	// Читаем содержимое файла
	data, err := os.ReadFile(filePath)
	if err != nil {
		// Если при чтении произошла ошибка, возвращаем пустой []byte и статус false
		return []byte{}, false
	}

	// Возвращаем содержимое файла и статус true
	return data, true
}

func (c *Cache) SetInt(key string, value int) error {
	return c.Set(key, []byte(strconv.Itoa(value)))
}

func (c *Cache) SetHeaders(key string, headers *http.Header) error {
	var buf bytes.Buffer

	// Проходим по всем заголовкам и добавляем их в буфер
	for name, values := range *headers {
		for _, value := range values {
			buf.WriteString(fmt.Sprintf("%s: %s\n", name, value))
		}
	}
	return c.Set(key, buf.Bytes())
}

func (c *Cache) Set(key string, value []byte) error {
	filePath := c.getFilePath(key)

	// Создаем файл с правами на запись (rw-r--r--)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error add to cache")
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	// Записываем данные в файл
	_, err = file.Write(value)
	if err != nil {
		return err
	}

	return nil
}

// RunCleanUp запускает функцию в горутине для периодической очистки
func (c *Cache) RunCleanUp() {
	go c.cleanUpOldFiles()
}

// CleanUpOldFiles проверяет файлы в директории и удаляет те, которые старше таймаута
func (c *Cache) cleanUpOldFiles() {
	if c.timeout <= 0 {
		return
	}

	for {
		// Проходим по всем файлам в директории
		err := filepath.Walk(c.folderPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Проверяем, что это файл (а не папка)
			if !info.IsDir() {
				// Если файл был изменён больше, чем на timeout назад, удаляем его
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

		// Ждем перед следующим запуском
		time.Sleep(c.timeout)
	}
}

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

func (c *Cache) ClearAll() {
	// Получаем список всех файлов и директорий в папке
	files, err := os.ReadDir(c.folderPath)
	if err != nil {
		log.Fatalf("failed to read directory: %w", err)
	}

	// Проходим по каждому элементу и удаляем его
	for _, file := range files {
		filePath := filepath.Join(c.folderPath, file.Name())
		err := os.RemoveAll(filePath) // Удаляем файл или директорию рекурсивно
		if err != nil {
			log.Printf("failed to remove %s: %s", filePath, err)
		}
	}
}

func (c *Cache) getFilePath(key string) string {
	return c.folderPath + "/" + key
}

func (c *Cache) createCacheDir() {
	// Создаем папку с правами 0755 (чтение/запись для владельца и чтение для группы и других пользователей)
	err := os.MkdirAll(c.folderPath, 0755)
	if err != nil {
		log.Fatalf("failed to create cache directory: %s\n", err)
	}
}
