FROM golang:1.25.6 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /dnd-api ./cmd/api

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /dnd-api /usr/local/bin/dnd-api
COPY migrations ./migrations

CMD ["dnd-api"]
