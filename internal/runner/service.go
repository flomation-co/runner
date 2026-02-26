package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"flomation.app/automate/runner/internal/version"

	r "flomation.app/automate/runner"
	"flomation.app/automate/runner/internal/executor"

	"flomation.app/automate/runner/internal/config"
	"flomation.app/automate/runner/internal/utils"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	state       *config.RunnerState
	config      *config.Config
	executor    *executor.Service
	running     bool
	lastCheckIn *time.Time
}

type Runner struct {
	ID               string     `json:"id"`
	Identifier       string     `json:"identifier"`
	Name             string     `json:"name"`
	RegistrationCode string     `json:"registration_code"`
	EnrolledAt       time.Time  `json:"enrolled_at"`
	LastContactAt    *time.Time `json:"last_contact_at"`
	IPAddress        *string    `json:"ip_address"`
	Version          *string    `json:"version"`
}

const FloStateFilename = "flo.state"

func NewService(cfg *config.Config) (*Service, error) {
	s := Service{
		config:   cfg,
		running:  true,
		executor: executor.NewService(cfg),
	}

	rs, err := config.LoadState(cfg.ExecutionConfig.StateDirectory + FloStateFilename)
	if err != nil {
		return nil, err
	}

	if rs == nil {
		rs = &config.RunnerState{
			ID: utils.GenerateRandomStringID(16),
		}

		b, err := json.Marshal(rs)
		if err != nil {
			return nil, err
		}

		if err := os.WriteFile(cfg.ExecutionConfig.StateDirectory+FloStateFilename, b, 0600); err != nil {
			return nil, err
		}
	}

	s.state = rs

	if err := s.registerRunner(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"api":   s.config.RunnerConfig.Server,
		}).Error("error initialising runner contact")
	}

	go s.monitor()

	return &s, nil
}

func (s *Service) registerRunner() error {
	client := http.Client{
		Timeout: time.Second * 15,
	}

	name := "Flo Runner"
	if s.config.RunnerConfig.Name != nil {
		name = *s.config.RunnerConfig.Name
	}

	runner := Runner{
		Identifier:       s.state.ID,
		RegistrationCode: s.config.RunnerConfig.RegistrationCode,
		Name:             name,
		Version:          &version.Version,
	}

	b, err := json.Marshal(runner)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v/api/v1/runner", s.config.RunnerConfig.Server), bytes.NewReader(b))
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("invalid status (%v) when registering runner", res.StatusCode)
	}

	return nil
}

func (s *Service) checkForExecutions() error {
	client := http.Client{
		Timeout: time.Second * 15,
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v/api/v1/runner/%v/execution", s.config.RunnerConfig.Server, s.state.ID), nil)
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("invalid status (%v) when registering runner", res.StatusCode)
	}

	if res.StatusCode == 204 {
		return nil
	}

	var response r.PendingExecution
	j, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(j, &response); err != nil {
		return err
	}

	f, err := json.Marshal(response.Data)
	if err != nil {
		return err
	}

	dir := fmt.Sprintf("%v/%v/%v", s.config.ExecutionConfig.ExecutionDirectory, response.Execution.FloID, response.Execution.ID)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
	}

	path := fmt.Sprintf("%v/%v/execution.flow", response.Execution.FloID, response.Execution.ID)
	filename := fmt.Sprintf("%v/%v", s.config.ExecutionConfig.ExecutionDirectory, path)
	if err := os.WriteFile(filename, f, 0600); err != nil {
		log.WithFields(log.Fields{
			"path":     path,
			"filename": filename,
			"error":    err,
		}).Error("unable to write flow to disk")
		return err
	}

	type executionStateType struct {
		State string `json:"state"`
	}

	executionState := executionStateType{
		State: "running",
	}

	b, err := json.Marshal(executionState)
	if err != nil {
		return err
	}

	req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("%v/api/v1/execution/%v/state", s.config.RunnerConfig.Server, response.Execution.ID), bytes.NewReader(b))
	if err != nil {
		return err
	}

	_, err = client.Do(req)
	if err != nil {
		return err
	}

	hasErrored := false
	output, success, err := s.executor.Execute(response.Execution.ID, response.Execution.FloID, "execution.flow", "", response.Flow.EnvironmentID)
	if err != nil || !success {
		hasErrored = true
	}

	// TODO: Give time for files to be written to disk
	time.Sleep(time.Second * 5)

	var state map[string]interface{}
	stateFilePath := fmt.Sprintf("%v%v/%v/state.json", s.config.ExecutionConfig.ExecutionDirectory, response.Execution.FloID, response.Execution.ID)
	filePath := filepath.Join(".", filepath.Clean(stateFilePath))
	sb, err := os.ReadFile(filePath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("error reading state file")

		state = map[string]interface{}{
			"error":  err.Error(),
			"output": output,
		}
	} else {
		if err := json.Unmarshal(sb, &state); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("error parsing state file")

			state = map[string]interface{}{
				"error":  err.Error(),
				"output": output,
			}
		}
	}

	state["logs"] = output

	url := fmt.Sprintf("%v/api/v1/execution/%v", s.config.RunnerConfig.Server, response.Execution.ID)
	executionResult := r.ExecutionResult{
		HasErrored: hasErrored,
		State:      state,
	}

	b, err = json.Marshal(executionResult)
	if err != nil {
		return err
	}

	req, err = http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}

	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) monitor() {
	// TODO: Run multiple monitors up to max parallel executors

	s.running = true
	for {
		if s.lastCheckIn == nil || time.Since(*s.lastCheckIn) > time.Duration(s.config.RunnerConfig.CheckInTimeout)*time.Second {
			now := time.Now()
			s.lastCheckIn = &now

			if err := s.checkForExecutions(); err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Error("unable to update runner contact")
			}
		}

		time.Sleep(time.Second * 1)
	}
}

func (s *Service) LoadState() (*config.RunnerState, error) {
	return nil, nil
}

func (s *Service) IsRunning() bool {
	return s.running
}
