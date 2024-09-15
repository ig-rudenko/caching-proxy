FROM golang:1.23.1-alpine AS builder

LABEL authors="ig-rudenko"

# Устанавливаем рабочую директорию
WORKDIR /app

COPY go.* /app/

RUN go mod download

# Копируем исходный код приложения
COPY . /app/

# Собираем бинарный файл приложения с отключением CGO
RUN CGO_ENABLED=0 go build -o caching-proxy ./cmd/main.go

# Стадия запуска
FROM alpine

# Копируем бинарный файл из стадии сборки
COPY --from=builder /app/caching-proxy /app/caching-proxy

WORKDIR /app

# Запускаем приложение
ENTRYPOINT ["./caching-proxy"]
