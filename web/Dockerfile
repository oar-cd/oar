# Build stage
FROM golang:1.24.5-alpine AS builder

# Accept version as build argument
ARG VERSION=dev

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with SQLite support and version
RUN CGO_ENABLED=1 go build -ldflags="-s -w -X github.com/oar-cd/oar/web/handlers.ServerVersion=${VERSION}" -o oar-web ./web

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache docker-cli-compose netcat-openbsd

WORKDIR /app

# Copy binary from build stage
COPY --from=builder /build/oar-web .

# Copy UI assets
COPY --from=builder /build/web/assets ./web/assets

CMD ["./oar-web"]
