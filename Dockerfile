# Use minimal Alpine Linux image
FROM dhi.io/alpine-base:3.23-alpine3.23-dev

# Install pre-requisite tools
RUN apk add --no-cache \
     ca-certificates \
     jq \
     procps\
     clamav

# Create the flomation user and group, pinned to an explicit uid/gid, and set up
# the runner's home.
#
# The -u/-g pinning is the whole point. `adduser -S` with no -u takes the first
# free system id, so this image previously landed on uid 101 rather than the 100
# every other Flomation image used — `apk add clamav` above claims 100 first.
# That made the runner's uid an accident of package ordering, and it would shift
# again if the package list changed. This is the one component with a persistent
# volume, so a silent uid shift makes an existing identity volume unreadable —
# which the runner cannot distinguish from a first boot, so it enrols a
# duplicate rather than failing.
RUN addgroup -g 10001 -S flomation && \
    adduser  -u 10001 -S flomation -G flomation && \
    mkdir -p /home/flomation/executor/lib/modules && \
    mkdir -p /home/flomation/workspace && \
    chown -R flomation:flomation /home/flomation

# Copy the binaries and entrypoint into the container.
# Owned by root and mode 0555: the application cannot rewrite its own
# executables, and one COPY replaces COPY + chmod + chown, which previously
# rewrote every byte of both binaries into a second image layer.
ARG BINARY_FILE
COPY --chown=root:root --chmod=0555 ${BINARY_FILE} /usr/local/bin/flomation-executor

ARG BINARY_FILE_2
COPY --chown=root:root --chmod=0555 ${BINARY_FILE_2} /usr/local/bin/flomation-runner

COPY --chown=root:root --chmod=0555 entrypoint.sh /usr/local/bin/entrypoint.sh

# Numeric rather than a name: with `runAsNonRoot: true` the kubelet refuses an
# image whose USER is a name, because it cannot verify the name is not root.
# The account is still called `flomation`, so `ps` and `ls -l` stay readable.
USER 10001:10001

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