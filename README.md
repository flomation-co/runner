# Flomation Runner

Long-running agent that polls the Flomation API for pending workflow executions and delegates them to the executor binary.

## Overview

The Flomation Runner is the remote execution agent for the Flomation Automate platform. It registers itself with the Flomation API, generates RSA keys for request signing, then continuously polls for pending workflow executions. When work is available, it writes the flow definition to disk, invokes the executor binary as a subprocess, and reports results back to the API. It supports both Kubernetes (ConfigMap) and Docker (environment variable) deployment modes.

## Prerequisites

- Go 1.26.1+
- Access to the Flomation API
- The `flomation-executor` binary available on the runner's path
- Docker (for containerised deployment)

## Installation

**Build from source:**

```sh
make build
```

Produces binaries under `dist/` for linux/amd64, linux/arm64, linux/arm, darwin/amd64, darwin/arm64, and windows/amd64.

**Docker:**

```sh
docker build \
  --build-arg BINARY_FILE=dist/flomation-executor-amd64-linux-1.0.local \
  --build-arg BINARY_FILE_2=dist/flomation-runner-amd64-linux-1.0.local \
  -t flomation-runner .
```

Base image: `dhi.io/alpine-base:3.23-alpine3.23-dev`. Runs as non-root `flomation` user. Includes a health check via `pgrep`.

## Configuration

The runner loads configuration from `config.json`. In Docker mode, the entrypoint script generates this file from environment variables.

**Config file structure (`config.json`):**

```json
{
  "runner": {
    "name": "My Runner",
    "url": "https://api.flomation.app",
    "registration_code": "your-registration-code",
    "checkin_timeout": 5
  },
  "execution": {
    "max_concurrent_executors": 5,
    "execution_directory": "/home/flomation/workspace/",
    "executable_name": "flomation-executor"
  }
}
```

**Runner settings:**

| Field | Env Variable | Description | Required | Default |
|-------|-------------|-------------|----------|---------|
| `url` | `FLOMATION_API` | Flomation API URL | Yes | — |
| `registration_code` | `FLOMATION_REGISTRATION_CODE` | Runner registration code | Yes | — |
| `name` | `FLOMATION_RUNNER_NAME` | Display name for this runner | No | `Flo Runner` |
| `checkin_timeout` | `FLOMATION_RUNNER_CHECKIN_TIMEOUT` | Seconds between API polls | No | `5` |
| `certificate` | `FLOMATION_RUNNER_CERTIFICATE_PATH` | Path to RSA private key PEM file | No | `flomation-runner.pem` |

**Execution settings:**

| Field | Env Variable | Description | Required | Default |
|-------|-------------|-------------|----------|---------|
| `max_concurrent_executors` | `FLOMATION_RUNNER_MAX_EXECUTORS` | Max parallel executions | No | — |
| `state_directory` | `FLOMATION_RUNNER_STATE_DIRECTORY` | Directory for runner state file | No | `./` |
| `execution_directory` | `FLOMATION_RUNNER_EXECUTION_DIRECTORY` | Working directory for executions | No | — |
| `executable_name` | `FLOMATION_RUNNER_EXECUTABLE_NAME` | Executor binary name or command | No | — |

**Docker environment variables (used by entrypoint when no `config.json` is mounted):**

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `RUNNER_NAME` | Runner display name | Yes | — |
| `RUNNER_URL` | Flomation API URL | Yes | — |
| `RUNNER_REGISTRATION_CODE` | Registration code | Yes | — |
| `RUNNER_CHECKIN_TIMEOUT` | Poll interval in seconds | No | `5` |
| `EXECUTOR_MAX_CONCURRENT` | Max concurrent executors | No | `5` |
| `EXECUTOR_DIRECTORY` | Execution working directory | No | `/home/flomation/workspace/` |
| `EXECUTOR_INSTALL_DIR` | Executor install directory | No | `/home/flomation/executor/lib` |
| `EXECUTOR_MODULE_DIR` | Executor module directory | No | `/home/flomation/executor/lib/modules` |
| `EXECUTOR_DOWNLOAD_ON_START` | Download executor on start | No | `true` |

## Usage

**Run with Docker (environment variables):**

```sh
docker run \
  -e RUNNER_NAME="My Runner" \
  -e RUNNER_URL="https://api.flomation.app" \
  -e RUNNER_REGISTRATION_CODE="your-code" \
  flomation-runner:latest
```

**Run with Kubernetes (ConfigMap):**

Mount `config.json` at `/usr/local/bin/config.json` via a ConfigMap. The entrypoint detects the file and uses it directly.

**Run the binary directly:**

```sh
./flomation-runner
```

The runner reads `config.json` from the current directory, registers with the API, and begins polling for executions.

## Development

**Run tests:**

```sh
make test
```

**Lint:**

```sh
make lint
```

Runs `goimports`, `golangci-lint`, `go vet`, `gosec`, and `govulncheck`.

## Project Structure

```
.
├── cmd/
│   └── main.go                     # Entry point — starts the runner service
├── types.go                        # Shared types (Flo, Execution, PendingExecution)
├── internal/
│   ├── config/                     # Config loading from JSON / env vars
│   ├── executor/                   # Subprocess wrapper for the executor binary
│   ├── runner/                     # Core runner service (registration, polling, signing)
│   ├── utils/                      # Random ID generation
│   └── version/                    # Build version info
├── entrypoint.sh                   # Docker entrypoint (K8s vs Docker mode)
├── Dockerfile
├── Makefile
└── go.mod
```

## Licence

MIT — Flomation LTD. See [LICENCE.md](LICENCE.md).
