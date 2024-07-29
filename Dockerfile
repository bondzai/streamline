# Stage 1: Build the Go application
FROM golang:1.21 as builder

WORKDIR /app

# Copy go.mod and go.sum files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the Go application
RUN CGO_ENABLED=0 GOOS=linux go build -o /streamline ./cmd/main.go

# Stage 2: Create the final image
FROM alpine:latest

WORKDIR /root/

# Copy the built application from the builder stage
COPY --from=builder /streamline .

# Copy configuration files if needed
COPY ./config /config
COPY ./env.yaml /config/env.yaml

# Expose ports for Fiber and net/http servers
EXPOSE 3000 3001

# Command to run the application
CMD ["./streamline"]
