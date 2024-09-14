package filecache

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Cache struct {
	folderPath string
}

func New(folderPath string) *Cache {
	c := &Cache{folderPath}
	c.createCacheDir()
	return c
}

func (c *Cache) Has(key string) bool {
	log.Printf("CACHE HAS %s\n", key)
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
	log.Printf("CACHE GET %s\n", key)
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
	log.Printf("CACHE SET %s\n", key)

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
