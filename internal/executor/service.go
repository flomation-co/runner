package executor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"flomation.app/automate/runner/internal/config"
	log "github.com/sirupsen/logrus"
)

// LogCallback is called with batches of log lines as they arrive from the executor.
type LogCallback func(lines []string)

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

	b, err := os.ReadFile(filepath.Join(s.config.ExecutionConfig.ExecutionDirectory, filename))
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

func (s *Service) Execute(ctx context.Context, id string, flow string, path string, entry string, environment *string, triggerData string, contextFile string, onLog LogCallback) (*string, bool, error) {
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

	if triggerData != "" {
		args = append(args, "--trigger-data")
		args = append(args, triggerData)
	}

	if contextFile != "" {
		args = append(args, "--context")
		args = append(args, contextFile)
	}

	if s.config.RunnerConfig.CertificateFilename != "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, false, err
		}

		certificatePath := filepath.Join(wd, s.config.RunnerConfig.CertificateFilename)
		args = append(args, "--key")
		args = append(args, certificatePath)
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
	cmd := exec.CommandContext(ctx, executionParts[0], args...)
	cmd.Dir = executionDirectory
	// Send SIGTERM on context cancellation (not SIGKILL) so the executor can clean up
	cmd.Cancel = func() error {
		log.WithFields(log.Fields{
			"id": id,
		}).Info("Sending SIGTERM to executor process")
		return cmd.Process.Signal(syscall.SIGTERM)
	}

	// Pipe stdout and stderr for real-time streaming
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false, err
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, false, err
	}

	if err := cmd.Start(); err != nil {
		return nil, false, err
	}

	// Read both stdout and stderr concurrently, collecting all output
	var allLines []string
	var mu sync.Mutex

	readPipe := func(pipe io.ReadCloser) {
		scanner := bufio.NewScanner(pipe)
		// Increase buffer to 1MB to handle large node events (e.g. with
		// base64 audio data). Default 64KB causes "token too long" errors
		// which silently stop reading the pipe and deadlock the executor.
		scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
		for scanner.Scan() {
			line := scanner.Text()
			mu.Lock()
			allLines = append(allLines, line)
			mu.Unlock()

			if onLog != nil {
				onLog([]string{line})
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); readPipe(stdoutPipe) }()
	go func() { defer wg.Done(); readPipe(stderrPipe) }()
	wg.Wait()

	err = cmd.Wait()
	output := strings.Join(allLines, "\n")

	if ctx.Err() != nil {
		log.WithFields(log.Fields{
			"id": id,
		}).Info("execution cancelled")
		return &output, false, fmt.Errorf("execution cancelled")
	}

	if err != nil {
		log.WithFields(log.Fields{
			"id":    id,
			"path":  path,
			"entry": entry,
		}).Info("execution failed")
		return &output, cmd.ProcessState.Success(), err
	}

	return &output, cmd.ProcessState.Success(), nil
}
