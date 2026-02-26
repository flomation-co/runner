package executor

import (
	"fmt"
	"os"
	"os/exec"
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

func (s *Service) Execute(id string, flow string, path string, entry string, environment *string) (*string, bool, error) {
	args := []string{
		s.config.ExecutionConfig.ExecutableName,
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
	cmd := exec.Command("node", args...)
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
