package config

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
)

func Test_LoadConfig(t *testing.T) {
	RegisterTestingT(t)

	// goconfig restricts file access to prevent path traversal,
	// so config files must be relative to the working directory.
	configJSON := `{
		"runner": {
			"url": "https://api.example.com",
			"registration_code": "test-code",
			"checkin_timeout": 30
		},
		"execution": {
			"max_concurrent_executors": 2,
			"state_directory": "/tmp/test-state/",
			"execution_directory": "/tmp/test-exec/",
			"executable_name": "executor"
		}
	}`

	err := os.WriteFile("test-config.json", []byte(configJSON), 0600)
	Expect(err).To(BeNil())
	defer os.Remove("test-config.json")

	cfg, err := LoadConfig("test-config.json")
	Expect(err).To(BeNil())
	Expect(cfg).To(Not(BeNil()))
	Expect(cfg.RunnerConfig.Server).To(Equal("https://api.example.com"))
	Expect(cfg.RunnerConfig.RegistrationCode).To(Equal("test-code"))
	Expect(cfg.ExecutionConfig.MaxConcurrentExecutors).To(Equal(int64(2)))
}

func Test_LoadConfigBadPath(t *testing.T) {
	RegisterTestingT(t)

	cfg, err := LoadConfig("some-bad-path-that-does-not-exist.json")
	Expect(err).To(Not(BeNil()))
	Expect(cfg).To(BeNil())
}

func Test_LoadConfigNotJSON(t *testing.T) {
	RegisterTestingT(t)

	err := os.WriteFile("not-json-config.json", []byte("this is not json"), 0600)
	Expect(err).To(BeNil())
	defer os.Remove("not-json-config.json")

	cfg, err := LoadConfig("not-json-config.json")
	Expect(err).To(Not(BeNil()))
	Expect(cfg).To(BeNil())
}

func Test_LoadConfigDefaults(t *testing.T) {
	RegisterTestingT(t)

	configJSON := `{
		"runner": {
			"url": "https://api.example.com",
			"registration_code": "test-code"
		},
		"execution": {}
	}`

	err := os.WriteFile("test-config-defaults.json", []byte(configJSON), 0600)
	Expect(err).To(BeNil())
	defer os.Remove("test-config-defaults.json")

	cfg, err := LoadConfig("test-config-defaults.json")
	Expect(err).To(BeNil())
	Expect(cfg).To(Not(BeNil()))
	Expect(cfg.ExecutionConfig.StateDirectory).To(Equal("./"))
	Expect(cfg.RunnerConfig.CertificateFilename).To(Equal("flomation-runner.pem"))
}

func Test_LoadState(t *testing.T) {
	RegisterTestingT(t)

	stateJSON := `{"identifier": "test-runner-id"}`

	tmpFile, err := os.CreateTemp("", "test-state-*.json")
	Expect(err).To(BeNil())
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(stateJSON)
	Expect(err).To(BeNil())
	tmpFile.Close()

	state, err := LoadState(tmpFile.Name())
	Expect(err).To(BeNil())
	Expect(state).To(Not(BeNil()))
	Expect(state.ID).To(Equal("test-runner-id"))
}

func Test_LoadStateBadPath(t *testing.T) {
	RegisterTestingT(t)

	state, err := LoadState("/nonexistent/path/flo.state")
	Expect(err).To(BeNil())
	Expect(state).To(BeNil())
}

func Test_LoadStateAbsolutePath(t *testing.T) {
	RegisterTestingT(t)

	// Regression test: absolute paths must work correctly.
	// Previously filepath.Join(".", filepath.Clean(path)) converted
	// absolute paths to relative, causing state to never be found.
	stateJSON := `{"identifier": "abs-path-runner"}`

	tmpDir, err := os.MkdirTemp("", "test-state-dir-*")
	Expect(err).To(BeNil())
	defer os.RemoveAll(tmpDir)

	absPath := tmpDir + "/flo.state"
	err = os.WriteFile(absPath, []byte(stateJSON), 0600)
	Expect(err).To(BeNil())

	state, err := LoadState(absPath)
	Expect(err).To(BeNil())
	Expect(state).To(Not(BeNil()))
	Expect(state.ID).To(Equal("abs-path-runner"))
}
