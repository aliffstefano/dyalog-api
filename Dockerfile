FROM golang:1.26-alpine AS builder

RUN apk add --no-cache build-base

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Ajuste no whatsmeow para a lista com envelope <biz> proprio (modo lista_biz).
# Ver tools/whatsmeow-overlay. Falha o build se o whatsmeow mudar o trecho.
RUN go run ./tools/whatsmeow-overlay /tmp/whatsmeow \
    && go mod edit -replace go.mau.fi/whatsmeow=/tmp/whatsmeow
# Versao vem do arquivo VERSION; commit e data podem vir do build:
#   docker build --build-arg COMMIT=$(git rev-parse --short HEAD) --build-arg BUILD_DATE=... .
ARG COMMIT=local
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X dyalog-api-go/internal/buildinfo.Version=$(cat VERSION | tr -d '[:space:]') -X dyalog-api-go/internal/buildinfo.Commit=${COMMIT} -X dyalog-api-go/internal/buildinfo.BuildDate=${BUILD_DATE}" \
    -o /bin/dyalog-api-go ./cmd/api

FROM alpine:3.21

# ffmpeg: converte audio para o formato de nota de voz (Ogg/Opus) que o
# iPhone toca; sem ele, MP3/Vorbis rotulados como ogg aparecem como "audio
# nao disponivel".
RUN apk add --no-cache ca-certificates libstdc++ sqlite-libs tzdata ffmpeg

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
