# Build stage
FROM golang:1.22-alpine AS builder

# Install git and ca-certificates (needed for private repos and HTTPS)
RUN apk add --no-cache git ca-certificates tzdata

# Create appuser for security
RUN adduser -D -g '' appuser

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Install SQLC
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Generate SQLC code
RUN sqlc generate

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o charity-server .

# Final stage
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

# Copy the binary
COPY --from=builder /build/charity-server /charity-server

# Copy migration files
COPY --from=builder /build/db/migration /db/migration

# Copy app.env
COPY --from=builder /build/app.env /app.env

# Use an unprivileged user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/charity-server", "--health-check"] || exit 1

# Run the binary
ENTRYPOINT ["/charity-server"]
