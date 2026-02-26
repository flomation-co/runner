# Flomation Runner Dockerfile
# This builds a containerized version of the Flomation runner application
#
# Supports two deployment modes:
#   1. Kubernetes: Mount config.json via ConfigMap
#   2. Docker: Pass environment variables, config auto-generated
#
# Prerequisites:
#   - runner-binary and executor-binary must be present in build context
#   - These are extracted from S3 artifacts in the CI pipeline (docker-prepare stage)
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

FROM alpine:3.22.2

# Metadata
LABEL maintainer="build@flomation.co"
LABEL description="Flomation Runner - Containerized workflow execution engine"
LABEL version="1.1"

# Install runtime dependencies
# - net-tools: Network utilities (netstat, etc.)
# - curl: HTTP client for API communication
# - ca-certificates: SSL/TLS support
# - jq: JSON processor for config validation
# - procps: Process utilities (for healthcheck)
RUN apk add --no-cache \
    net-tools \
    curl \
    ca-certificates \
    jq \
    procps

# Create flomation user (non-root for security)
# UID/GID 5000 to match RPM/DEB package specifications
RUN addgroup -g 5000 flomation && \
    adduser -D -u 5000 -G flomation -s /sbin/nologin -c "Flomation Service Account" flomation && \
    mkdir -p /home/flomation/executor/lib/modules && \
    mkdir -p /home/flomation/workspace && \
    chown -R flomation:flomation /home/flomation \


# Set working directory
WORKDIR /home/flomation

# Copy pre-extracted binaries (extracted in CI pipeline)
# These are provided as artifacts from the docker-prepare stage
COPY runner-binary /home/flomation/runner
COPY executor-binary /home/flomation/executor

# Set permissions and ownership
RUN chmod +x /home/flomation/runner /home/flomation/executor && \
    chown flomation:flomation /home/flomation/runner /home/flomation/executor

# Copy the entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# Switch to non-root user
USER flomation

# Health check - verify the application process is running
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep -f runner || exit 1

# Runtime configuration:
# Two modes supported:
#
# 1. Kubernetes mode (config.json mounted):
#    - config.json will be mounted at /home/flomation/config.json via ConfigMap
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
#      - EXECUTOR_DIRECTORY (default: /home/flomation/workspace/)
#      - EXECUTOR_INSTALL_DIR (default: /home/flomation/executor/lib)
#      - EXECUTOR_MODULE_DIR (default: /home/flomation/executor/lib/modules)
#      - EXECUTOR_DOWNLOAD_ON_START (default: true)

# Use entrypoint script to handle both deployment modes
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]