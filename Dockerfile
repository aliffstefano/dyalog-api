FROM golang:1.25-alpine AS builder

RUN apk add --no-cache build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /bin/dyalog-api-go ./cmd/api

FROM alpine:3.21

RUN apk add --no-cache ca-certificates libstdc++ sqlite-libs

WORKDIR /app

COPY --from=builder /bin/dyalog-api-go /app/dyalog-api-go
COPY --from=builder /app/static /app/static
COPY --from=builder /app/docs /app/docs
COPY --from=builder /app/migrations /app/migrations

EXPOSE 8080

ENV APP_PORT=8080
ENV APP_ENV=production
ENV SESSION_STORAGE_DIR=/app/data/sessoes
ENV TEMP_FILES_DIR=/app/data/temp
ENV UPDATE_ARTIFACTS_DIR=/app/data/updates

VOLUME ["/app/data"]

CMD ["/app/dyalog-api-go"]
