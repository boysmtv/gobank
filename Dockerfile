# ----- Build stage -----
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Download dependencies first (cache layer)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s -X main.Version=$(git describe --tags --always --dirty)" \
    -o /bin/gobank ./cmd/server

# ----- Runtime stage -----
FROM scratch

# Import ca-certs and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary
COPY --from=builder /bin/gobank /gobank

# Copy config (can be overridden by ConfigMap)
COPY --from=builder /app/configs/config.yaml /configs/config.yaml

EXPOSE 8080 9090

USER 65534:65534

ENTRYPOINT ["/gobank"]