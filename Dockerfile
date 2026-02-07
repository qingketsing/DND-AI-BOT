FROM golang:alpine AS builder

WORKDIR /app

# Set GOPROXY for faster/reliable download
ENV GOPROXY=https://goproxy.cn,direct

# Copy module files first
COPY go.mod ./
# Only copy go.sum if it exists
COPY go.sum* ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN go build -o dndbot main.go

# Runtime Environment
FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/dndbot .

# Optional: Copy .env if exists, though Env Vars are preferred in Docker
# COPY .env .

CMD ["./dndbot"]
