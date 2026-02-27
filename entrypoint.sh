#!/bin/sh
set -e

CONFIG_FILE="/home/flomation/config.json"

echo "=== Flomation Runner Startup ==="

# Check if config.json already exists (mounted by K8s ConfigMap)
if [ -f "$CONFIG_FILE" ]; then
    echo "✓ Config file found at $CONFIG_FILE"
    echo "  Assuming Kubernetes deployment - using existing config"
else
    echo "✗ Config file not found at $CONFIG_FILE"
    echo "  Assuming Docker deployment - checking for environment variables..."

    # Check if required environment variables are set
    if [ -z "$RUNNER_NAME" ] || [ -z "$RUNNER_URL" ] || [ -z "$RUNNER_REGISTRATION_CODE" ]; then
        echo ""
        echo "ERROR: Required environment variables are missing!"
        echo ""
        echo "For Docker deployments, you must provide:"
        echo "  RUNNER_NAME                 - Name of this runner"
        echo "  RUNNER_URL                  - API URL (e.g., https://api.dev.flomation.app)"
        echo "  RUNNER_REGISTRATION_CODE    - Registration code for this runner"
        echo ""
        echo "Optional environment variables:"
        echo "  RUNNER_CHECKIN_TIMEOUT      - Check-in timeout in seconds (default: 5)"
        echo "  EXECUTOR_MAX_CONCURRENT     - Max concurrent executors (default: 5)"
        echo "  EXECUTOR_DIRECTORY          - Execution directory (default: /home/flomation/workspace/)"
        echo "  EXECUTOR_INSTALL_DIR        - Executor install directory (default: /home/flomation/executor/lib)"
        echo "  EXECUTOR_MODULE_DIR         - Executor module directory (default: /home/flomation/executor/lib/modules)"
        echo "  EXECUTOR_DOWNLOAD_ON_START  - Download on start (default: true)"
        echo ""
        echo "Example Docker run command:"
        echo "  docker run -e RUNNER_NAME='My Runner' \\"
        echo "             -e RUNNER_URL='https://api.dev.flomation.app' \\"
        echo "             -e RUNNER_REGISTRATION_CODE='your-code' \\"
        echo "             flomation-runner:latest"
        echo ""
        exit 1
    fi

    # Set defaults for optional variables
    RUNNER_CHECKIN_TIMEOUT=${RUNNER_CHECKIN_TIMEOUT:-5}
    EXECUTOR_MAX_CONCURRENT=${EXECUTOR_MAX_CONCURRENT:-5}
    EXECUTOR_DIRECTORY=${EXECUTOR_DIRECTORY:-/home/flomation/workspace/}
    EXECUTOR_INSTALL_DIR=${EXECUTOR_INSTALL_DIR:-/home/flomation/executor/lib}
    EXECUTOR_MODULE_DIR=${EXECUTOR_MODULE_DIR:-/home/flomation/executor/lib/modules}
    EXECUTOR_DOWNLOAD_ON_START=${EXECUTOR_DOWNLOAD_ON_START:-true}

    echo "✓ Environment variables found - generating config.json"

    # Generate config.json from environment variables
    cat > "$CONFIG_FILE" <<EOF
{
  "runner": {
    "name": "$RUNNER_NAME",
    "url": "$RUNNER_URL",
    "registration_code": "$RUNNER_REGISTRATION_CODE",
    "checkin_timeout": $RUNNER_CHECKIN_TIMEOUT
  },
  "execution": {
    "max_concurrent_executors": $EXECUTOR_MAX_CONCURRENT,
    "execution_directory": "$EXECUTOR_DIRECTORY",
    "execution_install_dir": "$EXECUTOR_INSTALL_DIR",
    "execution_module_dir": "$EXECUTOR_MODULE_DIR",
    "download_on_start": $EXECUTOR_DOWNLOAD_ON_START
  }
}
EOF

    echo "✓ Config file generated successfully"

    # Show generated config (mask sensitive data)
    echo ""
    echo "Generated configuration:"
    cat "$CONFIG_FILE" | sed "s/\"registration_code\": \".*\"/\"registration_code\": \"***REDACTED***/g"
    echo ""
fi

# Verify config file is valid JSON
if ! command -v jq &> /dev/null; then
    echo "⚠ jq not available, skipping JSON validation"
else
    if jq empty "$CONFIG_FILE" 2>/dev/null; then
        echo "✓ Config JSON is valid"
    else
        echo "✗ ERROR: Invalid JSON in config file!"
        exit 1
    fi
fi

echo "=== Starting Flomation Runner ==="
echo ""

# Execute the application
exec /home/flomation/runner