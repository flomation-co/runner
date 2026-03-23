# Use minimal Alpine Linux image
FROM dhi.io/alpine-base:3.23-alpine3.23-dev

# Install pre-requisite tools
RUN apk add --no-cache \
     ca-certificates \
     jq \
     procps\
     clamav

# Create flomation user and group
RUN addgroup -S flomation && adduser -S flomation -G flomation &&\
    mkdir -p /home/flomation/executor/lib/modules && \
    mkdir -p /home/flomation/workspace && \
    chown -R flomation:flomation /home/flomation

# Copy the binary into the container
ARG BINARY_FILE
COPY ${BINARY_FILE} /usr/local/bin/flomation-executor

ARG BINARY_FILE_2
COPY ${BINARY_FILE_2} /usr/local/bin/flomation-runner

# Copy the entrypoint script
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/* && \
    chown flomation:flomation /usr/local/bin/*

# Switch to flomation user
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