# ─── Stage 1: Build the admin UI ─────────────────────────────────────────────
FROM node:22-alpine AS ui-builder
WORKDIR /build/ui
COPY ui/package.json ui/package-lock.json ./
RUN npm ci
COPY ui/ ./
RUN npm run build

# ─── Stage 2: Build Go binaries ───────────────────────────────────────────────
FROM golang:1.26-alpine AS go-builder
WORKDIR /build
# Download dependencies first (layer-cached separately from source).
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Embed the compiled admin UI.
COPY --from=ui-builder /build/internal/adminui/dist ./internal/adminui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/relay  ./cmd/relay

# ─── Stage 3: Runtime image ───────────────────────────────────────────────────
FROM alpine:3.21
# ffmpeg: required by bin/broadcast.sh when "Start Ingest" is used from the UI.
# curl + bash: required by bin/broadcast.sh.
# ca-certificates: required for HTTPS outbound requests (relay, inbox delivery).
RUN apk add --no-cache ffmpeg curl bash ca-certificates

WORKDIR /app
COPY --from=go-builder /out/server /out/relay ./
COPY bin/broadcast.sh ./bin/broadcast.sh
RUN chmod +x bin/broadcast.sh

# Keys are generated on first run and must survive container restarts.
VOLUME /app/keys

ENV CONFIG_PATH=/app/config.yaml

EXPOSE 8080

# Default: run the source server. Override with ["./relay"] for relay nodes.
CMD ["./server"]
