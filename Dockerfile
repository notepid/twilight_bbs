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

# Add a simple example DOS door for smoke testing.
# This lives under the configured `doors.drive_c` path (./doors/drive_c).
RUN mkdir -p /opt/bbs/doors/drive_c/drive_c/HELLO && \
    printf '%s\r\n' \
    '@echo off' \
    'cls' \
    'echo.' \
    'echo Twilight BBS DOSEMU2 test door' \
    'echo If you can see this, DOSEMU2 launched successfully.' \
    'echo A dropfile should have been created in the working directory (e.g. DOOR.SYS).' \
    'echo ran > C:\\HELLO\\RAN.TXT' \
    'echo.' \
    'echo Returning to the BBS...' \
    'exit' \
    > /opt/bbs/doors/drive_c/drive_c/HELLO/HELLO.BAT

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
    CMD timeout 2 bash -c 'echo | nc -w 1 localhost 2323' || exit 1

ENTRYPOINT ["bbs"]
CMD ["-config", "/opt/bbs/config.yaml"]
