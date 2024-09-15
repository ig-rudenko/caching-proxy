# Use the official Golang 1.23.1 base image with Alpine Linux for building the application
FROM golang:1.23.1-alpine AS builder

# Set a label to specify the author of this Dockerfile
LABEL authors="ig-rudenko"

# Set the working directory inside the container to /app
WORKDIR /app

# Copy go.mod and go.sum files to the /app directory in the container
COPY go.* /app/

# Download Go module dependencies defined in go.mod
RUN go mod download

# Copy the entire application source code to the /app directory in the container
COPY . /app/

# Build the Go application with CGO_ENABLED=0 to ensure a statically linked binary
# The binary will be named caching-proxy and located in /app
RUN CGO_ENABLED=0 go build -o caching-proxy ./cmd/main.go

# Start a new stage for the final runtime image
FROM alpine

# Copy the built binary from the builder stage to the /app directory in the new image
COPY --from=builder /app/caching-proxy /app/caching-proxy

# Set the working directory inside the container to /app
WORKDIR /app

# Set the default command to run the caching-proxy binary
ENTRYPOINT ["./caching-proxy"]
