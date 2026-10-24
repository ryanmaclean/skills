# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download
RUN go mod verify

# Copy source code
COPY . .

# Build arguments for version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
    -o dd \
    ./cmd

# Final stage - minimal image from scratch
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /build/dd /dd

# Set the entrypoint
ENTRYPOINT ["/dd"]

# Default command (show help)
CMD ["help"]

# Metadata
LABEL org.opencontainers.image.title="Datadog CLI"
LABEL org.opencontainers.image.description="Command-line interface for Datadog operations and queries"
LABEL org.opencontainers.image.source="https://github.com/datadog/skill"
LABEL org.opencontainers.image.licenses="MIT"
