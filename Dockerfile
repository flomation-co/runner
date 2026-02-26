# Flomation Runner Dockerfile
# This builds a containerized version of the Flomation runner application
#
# Supports two deployment modes:
#   1. Kubernetes: Mount config.json via ConfigMap
#   2. Docker: Pass environment variables, config auto-generated
#
# Prerequisites:
#   - Place latest.zip in the project root before building
#   - Or use the build-and-push.sh script which handles downloading from S3
#
# Build: docker build -t flomation-runner:latest .
#
# Run (K8s):
#   kubectl apply -f kubernetes-manifests.yaml
#
# Run (Docker with env vars):
#   docker run -e RUNNER_NAME="My Runner" \
#              -e RUNNER_URL="https://api.dev.flomation.app" \
#              -e RUNNER_REGISTRATION_CODE="your-code" \
#              flomation-runner:latest

FROM node:20-slim

# Metadata
LABEL maintainer="dave@flomation.co"
LABEL description="Flomation Runner - Containerized workflow execution engine"
LABEL version="1.0"

# Install runtime dependencies
# - net-tools: Network utilities (netstat, etc.)
# - zip/unzip: Archive handling for workflows and application extraction
# - curl: HTTP client for API communication
# - ca-certificates: SSL/TLS support
# - jq: JSON processor for config validation
RUN apt-get update && apt-get install -y \
    net-tools \
    zip \
    unzip \
    curl \
    ca-certificates \
    jq \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create platform user (non-root for security)
# UID/GID 1000 for compatibility with most systems
RUN useradd -m -u 1001 -s /bin/bash platform && \
    mkdir -p /home/platform/executor/lib/modules && \
    mkdir -p /home/platform/workspace && \
    chown -R platform:platform /home/platform

# Set working directory
WORKDIR /home/platform

# Copy the application zip file and extract it
# The zip contains the application binary
COPY latest.zip /tmp/latest.zip

# Extract the application binary and clean up
RUN unzip -o /tmp/latest.zip -d /home/platform && \
    mv /home/platform/*amd64-linux* /home/platform/application && \
    chmod +x /home/platform/application && \
    chown platform:platform /home/platform/application && \
    rm -f /tmp/latest.zip

# Copy the entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Switch to non-root user
USER platform

# Health check - verify the application process is running
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep -f application || exit 1

# Runtime configuration:
# Two modes supported:
#
# 1. Kubernetes mode (config.json mounted):
#    - config.json will be mounted at /home/platform/config.json via ConfigMap
#    - executor/ will be mounted for persistent execution libraries
#    - workspace/ will be mounted for workflow execution space
#    - flo.state will be in mounted state directory
#
# 2. Docker mode (environment variables):
#    Required:
#      - RUNNER_NAME
#      - RUNNER_URL
#      - RUNNER_REGISTRATION_CODE
#    Optional (with defaults):
#      - RUNNER_CHECKIN_TIMEOUT (default: 5)
#      - EXECUTOR_MAX_CONCURRENT (default: 5)
#      - EXECUTOR_DIRECTORY (default: /home/platform/workspace/)
#      - EXECUTOR_INSTALL_DIR (default: /home/platform/executor/lib)
#      - EXECUTOR_MODULE_DIR (default: /home/platform/executor/lib/modules)
#      - EXECUTOR_DOWNLOAD_ON_START (default: true)

# Use entrypoint script to handle both deployment modes
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]