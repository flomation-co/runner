package config

import (
	"encoding/json"
	goconfig "github.com/flomation-co/go-config"
	"os"
	"path/filepath"
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
	Server              string  `json:"url"`
	RegistrationCode    string  `json:"registration_code"`
	Name                *string `json:"name"`
	CheckInTimeout      int     `json:"checkin_timeout"`
	CertificateFilename string  `json:"certificate"`
}

type ExecutionConfig struct {
	MaxConcurrentExecutors int64  `json:"max_concurrent_executors"`
	StateDirectory         string `json:"state_directory"`
	ExecutionDirectory     string `json:"execution_directory"`
	ExecutableName         string `json:"executable_name"`
}

type Config struct {
	RunnerConfig    RunnerConfig    `json:"runner"`
	ExecutionConfig ExecutionConfig `json:"execution"`
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
