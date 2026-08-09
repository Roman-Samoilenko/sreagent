FROM golang:1.26.5-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG APP_CMD=updater
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/${APP_CMD} ./cmd/${APP_CMD}

# Финальный образ
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/bin/updater ./

COPY config/ ./config/
COPY data/ ./data/

ENTRYPOINT ["./updater"]
CMD ["--config", "/app/config/config.docker.yaml"]