package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	goconfig "github.com/flomation-co/go-config"
)

type RunnerState struct {
	ID string `json:"identifier"`
}

func LoadState(path string) (*RunnerState, error) {
	filePath := filepath.Join(".", filepath.Clean(path))
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil
	}

	var config RunnerState

	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

type RunnerConfig struct {
	Server              string  `json:"url" env:"FLOMATION_API" arg:"api-url"`
	RegistrationCode    string  `json:"registration_code" env:"FLOMATION_REGISTRATION_CODE" arg:"registration-code"`
	Name                *string `json:"name" env:"FLOMATION_RUNNER_NAME" arg:"runner-name"`
	CheckInTimeout      int     `json:"checkin_timeout" env:"FLOMATION_RUNNER_CHECKIN_TIMEOUT" arg:"checkin-timeout"`
	CertificateFilename string  `json:"certificate" env:"FLOMATION_RUNNER_CERTIFICATE_PATH" arg:"certificate-filename"`
}

type ExecutionConfig struct {
	MaxConcurrentExecutors int64  `json:"max_concurrent_executors" env:"FLOMATION_RUNNER_MAX_EXECUTORS" arg:"max_concurrent_executors"`
	StateDirectory         string `json:"state_directory" env:"FLOMATION_RUNNER_STATE_DIRECTORY" arg:"state_directory"`
	ExecutionDirectory     string `json:"execution_directory" env:"FLOMATION_RUNNER_EXECUTION_DIRECTORY" arg:"execution_directory"`
	ExecutableName         string `json:"executable_name" env:"FLOMATION_RUNNER_EXECUTABLE_NAME" arg:"executable_name"`
}

type TLSConfig struct {
	Enabled    bool   `json:"enabled" env:"MTLS_ENABLED" arg:"mtls-enabled"`
	CACertFile string `json:"ca_cert" env:"MTLS_CA_CERT" arg:"mtls-ca-cert"`
	CertFile   string `json:"cert" env:"MTLS_CERT" arg:"mtls-cert"`
	KeyFile    string `json:"key" env:"MTLS_KEY" arg:"mtls-key"`
	APIURL     string `json:"api_url" env:"MTLS_API_URL" arg:"mtls-api-url"`
}

type Config struct {
	RunnerConfig    RunnerConfig    `json:"runner"`
	ExecutionConfig ExecutionConfig `json:"execution"`
	TLS             *TLSConfig      `json:"tls,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	var c Config

	filePath := filepath.Join(".", filepath.Clean(path))
	if err := goconfig.Load(&c, goconfig.String(filePath)); err != nil {
		return &c, nil
	}

	if c.ExecutionConfig.StateDirectory == "" {
		c.ExecutionConfig.StateDirectory = "./"
	}

	if c.RunnerConfig.CertificateFilename == "" {
		c.RunnerConfig.CertificateFilename = "flomation-runner.pem"
	}

	return &c, nil
}
