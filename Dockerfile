# Build stage
FROM golang:1.24.5-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with SQLite support
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o oar .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache docker-cli-compose su-exec

WORKDIR /app

# Copy binary from build stage
COPY --from=builder /build/oar .

# Copy UI assets
COPY --from=builder /build/ui ./ui

CMD ["./oar"]
