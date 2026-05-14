FROM golang:1.26-alpine AS builder

WORKDIR /src

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" \
    -o /out/em-subscriptions ./cmd/api

# --- runtime image ------------------------------------------------------

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 1000 app

WORKDIR /app
COPY --from=builder /out/em-subscriptions /app/em-subscriptions
COPY config.yaml /app/config.yaml
COPY migrations /app/migrations

ENV CONFIG_PATH=/app/config.yaml \
    DB_MIGRATIONS_PATH=file:///app/migrations

USER app
EXPOSE 8080

ENTRYPOINT ["/app/em-subscriptions"]
