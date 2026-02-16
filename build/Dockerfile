# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS backend-builder
WORKDIR /src
RUN apk add --no-cache build-base git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -trimpath -tags sqlite_fts5 -ldflags='-s -w' -o /out/kenaz ./cmd/app

FROM node:22-alpine AS frontend-builder
WORKDIR /src/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

FROM alpine:3.22 AS runtime
WORKDIR /app
RUN apk add --no-cache ca-certificates && update-ca-certificates

COPY --from=backend-builder /out/kenaz /app/kenaz
COPY --from=frontend-builder /src/frontend/dist /app/frontend/dist
COPY config/config.yaml /app/config/config.yaml

ENV APP_CONFIG_FILE=/app/config/config.yaml \
    VAULT_PATH=/app/vault \
    SQLITE_PATH=/app/kenaz.db \
    AUTH_MODE=disabled \
    FRONTEND_ENABLED=true \
    FRONTEND_DIST_PATH=/app/frontend/dist

EXPOSE 8080

VOLUME ["/app/vault", "/app/config"]

CMD ["/app/kenaz", "serve", "--config", "/app/config/config.yaml"]
