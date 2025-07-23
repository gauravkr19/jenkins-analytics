FROM golang:1.24 AS builder

WORKDIR /app

# Cache modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/

# ---------- Stage 2: Run ----------
FROM alpine:3.19

# Install minimal deps (optional: ca-certificates)
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy binary and templates/static files from builder
COPY --from=builder /app/server ./
COPY --from=builder /app/web ./web
COPY --from=builder /app/internal/web/static ./internal/web/static

# Expose port
EXPOSE 8092

# Run the server
ENTRYPOINT ["./server"]

