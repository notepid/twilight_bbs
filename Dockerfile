# Build stage
FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bbs ./cmd/bbs/

# Runtime stage
FROM debian:bookworm-slim

# Install dosemu2 for DOS door support (optional)
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create BBS user
RUN useradd -m -s /bin/bash bbs

# Copy binary
COPY --from=builder /bbs /usr/local/bin/bbs

# Copy assets
COPY assets/ /opt/bbs/assets/
COPY config.yaml /opt/bbs/config.yaml

# Create data directories
RUN mkdir -p /opt/bbs/data /opt/bbs/assets/menus /opt/bbs/assets/text && \
    chown -R bbs:bbs /opt/bbs

# Expose ports
EXPOSE 2323 2222

# Run as BBS user
USER bbs
WORKDIR /opt/bbs

# Health check
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD timeout 2 bash -c 'echo | nc -w 1 localhost 2323' || exit 1

ENTRYPOINT ["bbs"]
CMD ["-config", "/opt/bbs/config.yaml"]
