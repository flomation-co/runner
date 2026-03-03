package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"flomation.app/automate/runner/internal/config"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	config *config.Config
}

func NewService(config *config.Config) *Service {
	return &Service{
		config: config,
	}
}

func (s *Service) Manifest() (interface{}, error) {
	filename := "manifest.json"
	args := []string{
		"--manifest",
		filename,
	}

	executionParts := strings.Split(s.config.ExecutionConfig.ExecutableName, " ")
	if len(executionParts) > 1 {
		args = append(executionParts[1:], args...)
	}

	cmd := exec.Command(executionParts[0], args...)
	cmd.Dir = s.config.ExecutionConfig.ExecutionDirectory

	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(path.Join(s.config.ExecutionConfig.ExecutionDirectory, filename))
	if err != nil {
		return nil, err
	}

	var manifest interface{}
	if err := json.Unmarshal(b, &manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}

func (s *Service) Version() (*string, error) {
	args := []string{
		"--version",
	}

	executionParts := strings.Split(s.config.ExecutionConfig.ExecutableName, " ")
	if len(executionParts) > 1 {
		args = append(executionParts[1:], args...)
	}

	cmd := exec.Command(executionParts[0], args...)
	cmd.Dir = s.config.ExecutionConfig.ExecutionDirectory

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	output := string(out)

	return &output, nil
}

func (s *Service) Execute(id string, flow string, path string, entry string, environment *string) (*string, bool, error) {
	args := []string{
		"--output",
		"json",
		"--path",
		path,
		"--entry",
		entry,
		"--id",
		id,
		"--flow",
		flow,
		"--api",
		s.config.RunnerConfig.Server,
		"--runner",
		*s.config.RunnerConfig.Name,
	}

	if environment != nil {
		args = append(args, "--environment")
		args = append(args, *environment)
	}

	executionParts := strings.Split(s.config.ExecutionConfig.ExecutableName, " ")
	if len(executionParts) > 1 {
		args = append(executionParts[1:], args...)
	}

	log.WithFields(log.Fields{
		"args": strings.Join(args, " "),
	}).Info("invoking executor")

	executionDirectory := fmt.Sprintf("%v/%v/%v", s.config.ExecutionConfig.ExecutionDirectory, flow, id)
	_, err := os.Stat(executionDirectory)
	if err != nil && !os.IsNotExist(err) {
		return nil, false, err
	}

	if err := os.MkdirAll(executionDirectory, 0750); err != nil {
		return nil, false, err
	}

	// #nosec G204 -- Parameters for Executor are intentional and controlled
	fmt.Printf("%v %v %v\n", executionDirectory, executionParts[0], args)
	cmd := exec.Command(executionParts[0], args...)
	cmd.Dir = executionDirectory

	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil {
		log.WithFields(log.Fields{
			"id":     id,
			"path":   path,
			"entry":  entry,
			"output": string(out),
		}).Info("execution failed")
		return &output, cmd.ProcessState.Success(), err
	}

	return &output, cmd.ProcessState.Success(), nil
}
