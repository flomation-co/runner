package config

import (
	"encoding/json"
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
	Server           string  `json:"url"`
	RegistrationCode string  `json:"registration_code"`
	Name             *string `json:"name"`
	CheckInTimeout   int     `json:"checkin_timeout"`
}

type ExecutionConfig struct {
	MaxConcurrentExecutors   int64  `json:"max_concurrent_executors"`
	StateDirectory           string `json:"state_directory"`
	ExecutionDirectory       string `json:"execution_directory"`
	ExecutorInstallDirectory string `json:"execution_install_dir"`
	ExecutorModuleDirectory  string `json:"execution_module_dir"`
	DownloadOnStart          bool   `json:"download_on_start"`
	ExecutableName           string `json:"executable_name"`
}

type Config struct {
	RunnerConfig    RunnerConfig    `json:"runner"`
	ExecutionConfig ExecutionConfig `json:"execution"`
}

func LoadConfig(path string) (*Config, error) {
	filePath := filepath.Join(".", filepath.Clean(path))
	b, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Config

	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	if config.ExecutionConfig.StateDirectory == "" {
		config.ExecutionConfig.StateDirectory = "./"
	}

	return &config, nil
}
