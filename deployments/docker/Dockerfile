FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates && update-ca-certificates

COPY --from=builder /app/server /app/server
COPY --from=builder /app/configs/config.example.yaml /app/configs/config.yaml
COPY --from=builder /app/internal/repository/postgres/migrations /app/internal/repository/postgres/migrations

ENV CONFIG_PATH=/app/configs/config.yaml

EXPOSE 50051

ENTRYPOINT ["/app/server"]

