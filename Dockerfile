# ====== BUILDER ======
FROM golang:1.23-alpine AS builder

WORKDIR /app

# install git (penting untuk beberapa dependency)
RUN apk add --no-cache git

# copy go mod dulu (biar cache optimal)
COPY go.mod go.sum ./
RUN go mod download

# copy source code
COPY . .

# build binary (static biar ringan)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gobank ./cmd/server

# ====== RUNNER ======
FROM alpine:3.20

WORKDIR /app

# install cert (penting untuk HTTPS, JWT, dll)
RUN apk add --no-cache ca-certificates

# copy binary
COPY --from=builder /app/gobank /app/gobank

# copy config
COPY --from=builder /app/configs/config.yaml /app/configs/config.yaml

# expose port
EXPOSE 8080

# run app
CMD ["/app/gobank"]