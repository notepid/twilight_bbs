# Build stage
FROM golang:1.24-trixie AS builder

WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bbs ./cmd/bbs/

# Runtime stage
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

# Install dosemu2 (from official PPA) and runtime deps
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    ca-certificates \
    gnupg \
    netcat-openbsd \
    perl \
    software-properties-common \
    && rm -rf /var/lib/apt/lists/*

RUN add-apt-repository -y ppa:dosemu2/ppa && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
    dosemu2 \
    && rm -rf /var/lib/apt/lists/*

# Create BBS user
RUN useradd -m -s /bin/bash bbs

# Copy binary
COPY --from=builder /bbs /usr/local/bin/bbs

# Copy assets
COPY assets/ /opt/bbs/assets/
COPY config.yaml /opt/bbs/config.yaml

# Copy doors (DOSEMU drive C tree) from the repo.
# Expected layout: doors/drive_c/<DOORNAME>/...
COPY doors/drive_c/ /opt/bbs/doors/drive_c/


# Normalize DOS text files to CRLF so batch files work reliably regardless of
# the build host OS / git newline settings. Do not touch binaries.
RUN find /opt/bbs/doors/drive_c -type f \( -iname '*.bat' -o -iname '*.cmd' -o -iname '*.txt' \) -print0 | \
    xargs -0 -r perl -pi -e 's/\r?\n/\r\n/g'

# Create data directories
RUN mkdir -p /opt/bbs/data /opt/bbs/assets/menus /opt/bbs/assets/text /opt/bbs/doors/drive_c && \
    chown -R bbs:bbs /opt/bbs

# Expose ports
EXPOSE 2323 2222

# Run as BBS user
USER bbs
WORKDIR /opt/bbs

# Health check
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD timeout 2 bash -c 'printf "GET /healthz HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n" | nc -w 1 localhost 2223 | grep -q "200 OK"' || exit 1

ENTRYPOINT ["bbs"]
CMD ["-config", "/opt/bbs/config.yaml"]
