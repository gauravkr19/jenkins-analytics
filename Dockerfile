# syntax=docker/dockerfile:1

############################
# Build stage
############################
FROM golang:1.24.1 AS builder

WORKDIR /src

# Force vendor usage; avoid network downloads
ENV GO111MODULE=on \
    GOPROXY=off \
    GOSUMDB=off

COPY go.mod go.sum ./
COPY vendor/ ./vendor/
COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -mod=vendor -trimpath -ldflags="-s -w" \
    -o /out/server ./cmd

############################
# Runtime stage (matches your working v4 inspect)
############################
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

# binary name matches existing running image expectation
COPY --from=builder /out/server ./server

# IMPORTANT: copy the whole web/ directory (not just templates/)
COPY --from=builder /src/web ./web

# Static assets path exactly as in the working image
COPY --from=builder /src/internal/web/static ./internal/web/static

EXPOSE 8091

ENTRYPOINT ["./server"]
