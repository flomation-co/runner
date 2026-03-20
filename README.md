# Flomation Runner

Runner service that executes automation workflows on behalf of the Flomation Automate platform.

## Overview

The Flomation Runner is a long-running Go service that registers with the Flomation API, periodically polls for pending workflow executions, and delegates them to a local executor binary. Requests are authenticated using RSA-PSS signatures. It supports both Kubernetes and standalone Docker deployments.

## Prerequisites

- Go 1.26.1+
- `golangci-lint`, `goimports`, `gosec`, and `govulncheck` (for linting)
- Docker (for container builds)

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd runner

# Install dependencies
go mod download

# Build for all supported platforms
make build
```

Binaries are written to `dist/` and bundled into `build.zip`. Supported targets: `linux/amd64`, `linux/arm64`, `linux/arm`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`.

## Configuration

The runner loads configuration from a `config.json` file and/or environment variables.

### Config file (`config.json`)

```json
{
  "runner": {
    "name": "My Runner",
    "url": "https://api.dev.flomation.app",
    "registration_code": "your-registration-code",
    "checkin_timeout": 5,
    "certificate": "flomation-runner.pem"
  },
  "execution": {
    "max_concurrent_executors": 5,
    "state_directory": "./",
    "execution_directory": "/home/flomation/workspace/",
    "executable_name": "flomation-executor"
  }
}
```

### Environment variables

The Go binary reads these environment variables directly (they override config file values):

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `FLOMATION_API` | Flomation API URL | Yes | — |
| `FLOMATION_REGISTRATION_CODE` | Runner registration code | Yes | — |
| `FLOMATION_RUNNER_NAME` | Display name for this runner | No | `Flo Runner` |
| `FLOMATION_RUNNER_CHECKIN_TIMEOUT` | Seconds between check-in polls | No | — |
| `FLOMATION_RUNNER_CERTIFICATE_PATH` | Path to RSA private key PEM file | No | `flomation-runner.pem` |
| `FLOMATION_RUNNER_MAX_EXECUTORS` | Maximum concurrent executors | No | — |
| `FLOMATION_RUNNER_STATE_DIRECTORY` | Directory for runner state files | No | `./` |
| `FLOMATION_RUNNER_EXECUTION_DIRECTORY` | Working directory for executions | No | — |
| `FLOMATION_RUNNER_EXECUTABLE_NAME` | Executor binary name/path | No | — |

### Docker environment variables

When running via Docker without a mounted `config.json`, the entrypoint script accepts a simplified set of variables and generates the config file automatically:

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `RUNNER_NAME` | Name of this runner | Yes | — |
| `RUNNER_URL` | Flomation API URL | Yes | — |
| `RUNNER_REGISTRATION_CODE` | Registration code | Yes | — |
| `RUNNER_CHECKIN_TIMEOUT` | Check-in timeout in seconds | No | `5` |
| `EXECUTOR_MAX_CONCURRENT` | Max concurrent executors | No | `5` |
| `EXECUTOR_DIRECTORY` | Execution working directory | No | `/home/flomation/workspace/` |
| `EXECUTOR_INSTALL_DIR` | Executor install directory | No | `/home/flomation/executor/lib` |
| `EXECUTOR_MODULE_DIR` | Executor module directory | No | `/home/flomation/executor/lib/modules` |
| `EXECUTOR_DOWNLOAD_ON_START` | Download modules on startup | No | `true` |

## Usage

### Run with Docker

```bash
docker run \
  -e RUNNER_NAME='Production Runner' \
  -e RUNNER_URL='https://api.dev.flomation.app' \
  -e RUNNER_REGISTRATION_CODE='abc123' \
  flomation-runner:latest
```

### Run with Kubernetes

Mount a `config.json` via ConfigMap at `/usr/local/bin/config.json`. The entrypoint detects the file and skips environment variable generation.

### Run locally

```bash
# Create a config.json in the working directory
flomation-runner
```

## Development

```bash
# Run tests with coverage
make test

# Lint (runs go mod tidy, goimports, golangci-lint, go vet, gosec, govulncheck)
make lint

# Build all platforms
make build
```

Version, git hash, and build date are injected at build time via `-ldflags`.

## Project Structure

```
.
├── cmd/
│   └── main.go                  # Application entrypoint
├── internal/
│   ├── config/
│   │   ├── config.go            # Configuration loading and structs
│   │   └── config_test.go
│   ├── executor/
│   │   └── service.go           # Executor binary invocation
│   ├── runner/
│   │   └── service.go           # Runner registration, polling, and execution orchestration
│   ├── utils/
│   │   └── util.go              # Random ID generation
│   └── version/
│       ├── version.go           # Build version metadata
│       └── version_test.go
├── types.go                     # Shared domain types (Flo, Execution, PendingExecution)
├── Dockerfile                   # Container image definition
├── entrypoint.sh                # Docker/K8s entrypoint script
├── Makefile                     # Build, lint, and test targets
├── project-metadata.json        # Package metadata
├── go.mod
└── go.sum
```

## Licence

MIT — see [LICENCE.md](LICENCE.md).
