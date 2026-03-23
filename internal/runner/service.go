package runner

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
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

type SigningConfig struct {
	PrivateKeyBytes *rsa.PrivateKey
	PublicKeyBytes  []byte
}

type Service struct {
	state       *config.RunnerState
	config      *config.Config
	executor    *executor.Service
	signing     *SigningConfig
	running     bool
	lastCheckIn *time.Time
}

type Runner struct {
	ID               string      `json:"id"`
	Identifier       string      `json:"identifier"`
	Name             string      `json:"name"`
	RegistrationCode string      `json:"registration_code"`
	EnrolledAt       time.Time   `json:"enrolled_at"`
	LastContactAt    *time.Time  `json:"last_contact_at"`
	IPAddress        *string     `json:"ip_address"`
	Version          *string     `json:"version"`
	ExecutorVersion  *string     `json:"executor_version"`
	Manifest         interface{} `json:"manifest"`
	PublicKey        *string     `json:"public_key" `
}

type RunnerRequest struct {
	RequestedAt time.Time `json:"requested_at"`
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

	if _, err := os.Stat(s.config.ExecutionConfig.ExecutionDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(s.config.ExecutionConfig.ExecutionDirectory, 0750); err != nil {
			return nil, err
		}
	}

	s.state = rs

	if err := s.generateKeys(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to generate runner private key")
	}

	if err := s.registerRunner(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"api":   s.config.RunnerConfig.Server,
		}).Error("error initialising runner contact")
	}

	go s.monitor()

	return &s, nil
}

func (s *Service) generateKeys() error {
	if s.config.RunnerConfig.CertificateFilename == "" {
		return nil
	}

	if b, err := os.ReadFile(s.config.RunnerConfig.CertificateFilename); err == nil {
		block, _ := pem.Decode(b)
		if block == nil {
			return errors.New("invalid pem file")
		}

		switch block.Type {
		case "RSA PRIVATE KEY":
			key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return err
			}

			publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
			if err != nil {
				return err
			}

			publicPem := pem.EncodeToMemory(&pem.Block{
				Type:  "PUBLIC KEY",
				Bytes: publicBytes,
			})

			s.signing = &SigningConfig{
				PrivateKeyBytes: key,
				PublicKeyBytes:  publicPem,
			}

		case "PRIVATE KEY":
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return err
			}
			rsaKey, ok := key.(*rsa.PrivateKey)
			if !ok {
				return errors.New("invalid pem format")
			}

			publicBytes, err := x509.MarshalPKIXPublicKey(&rsaKey.PublicKey)
			if err != nil {
				return err
			}

			publicPem := pem.EncodeToMemory(&pem.Block{
				Type:  "PUBLIC KEY",
				Bytes: publicBytes,
			})

			s.signing.PrivateKeyBytes = rsaKey
			s.signing.PublicKeyBytes = publicPem
		default:
			return errors.New("invalid private key type")
		}

		return nil
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	p := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	publicBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return err
	}

	publicPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicBytes,
	})

	s.signing = &SigningConfig{
		PrivateKeyBytes: key,
		PublicKeyBytes:  publicPem,
	}

	if err := os.WriteFile(s.config.RunnerConfig.CertificateFilename, p, 0600); err != nil {
		return err
	}

	return nil
}

func (s *Service) registerRunner() error {
	client := http.Client{
		Timeout: time.Second * 15,
	}

	name := "Flo Runner"
	if s.config.RunnerConfig.Name != nil {
		name = *s.config.RunnerConfig.Name
	}

	v, err := s.executor.Version()
	if err != nil {
		return err
	}

	manifest, err := s.executor.Manifest()
	if err != nil {
		return err
	}

	runner := Runner{
		Identifier:       s.state.ID,
		RegistrationCode: s.config.RunnerConfig.RegistrationCode,
		Name:             name,
		Version:          &version.Version,
		ExecutorVersion:  v,
		Manifest:         manifest,
	}

	if s.signing != nil && len(s.signing.PublicKeyBytes) > 0 {
		k := string(s.signing.PublicKeyBytes)
		runner.PublicKey = &k
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

	update := RunnerRequest{
		RequestedAt: time.Now().UTC(),
	}

	k, err := json.Marshal(update)
	if err != nil {
		return err
	}

	hash := sha256.Sum256(k)
	signature, err := rsa.SignPSS(rand.Reader, s.signing.PrivateKeyBytes, crypto.SHA256, hash[:], nil)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%v/api/v1/runner/%v/execution", s.config.RunnerConfig.Server, s.state.ID), bytes.NewBuffer(k))
	if err != nil {
		return err
	}

	req.Header.Set("X-Flomation-Runner-Signature", hex.EncodeToString(signature))

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

	hash = sha256.Sum256(b)
	signature, err = rsa.SignPSS(rand.Reader, s.signing.PrivateKeyBytes, crypto.SHA256, hash[:], nil)
	if err != nil {
		return err
	}

	req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("%v/api/v1/execution/%v/state", s.config.RunnerConfig.Server, response.Execution.ID), bytes.NewReader(b))
	if err != nil {
		return err
	}

	req.Header.Set("X-Flomation-Runner-Signature", hex.EncodeToString(signature))

	_, err = client.Do(req)
	if err != nil {
		return err
	}

	// Write trigger invocation data if present
	var triggerDataPath string
	if response.Execution.Data != nil {
		var tdBytes []byte

		// Data may already be a map, string, or raw JSON — normalise to a JSON object
		switch v := response.Execution.Data.(type) {
		case string:
			// Could be a JSON string containing an object — try to use it directly
			if len(v) > 2 && v[0] == '{' {
				tdBytes = []byte(v)
			} else {
				// Try to unmarshal the string as JSON
				var inner interface{}
				if err := json.Unmarshal([]byte(v), &inner); err == nil {
					tdBytes, _ = json.Marshal(inner)
				}
			}
		default:
			tdBytes, _ = json.Marshal(v)
		}

		if len(tdBytes) > 2 {
			tdFile := fmt.Sprintf("%v/%v/%v/trigger-data.json", s.config.ExecutionConfig.ExecutionDirectory, response.Execution.FloID, response.Execution.ID)
			if err := os.WriteFile(tdFile, tdBytes, 0600); err != nil {
				log.WithFields(log.Fields{"error": err}).Warn("unable to write trigger data file")
			} else {
				triggerDataPath = "trigger-data.json"
			}
		}
	}

	hasErrored := false
	logCallback := s.createLogCallback(response.Execution.ID)
	output, success, err := s.executor.Execute(response.Execution.ID, response.Execution.FloID, "execution.flow", "", response.Flow.EnvironmentID, triggerDataPath, logCallback)
	if err != nil || !success {
		hasErrored = true
	}

	// TODO: Give time for files to be written to disk
	time.Sleep(time.Second * 5)

	var state map[string]interface{}
	stateFilePath := fmt.Sprintf("%v/%v/%v/state.json", s.config.ExecutionConfig.ExecutionDirectory, response.Execution.FloID, response.Execution.ID)
	filePath := filepath.Join(".", filepath.Clean(stateFilePath))
	sb, err := os.ReadFile(filePath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("error reading state file")

		hasErrored = true
		state = map[string]interface{}{
			"error":  err.Error(),
			"output": output,
		}
	} else {
		if err := json.Unmarshal(sb, &state); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("error parsing state file")

			hasErrored = true
			state = map[string]interface{}{
				"error":  err.Error(),
				"output": output,
			}
		} else if status, ok := state["status"]; ok {
			if s, ok := status.(float64); ok && s != 0 {
				hasErrored = true
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

	hash = sha256.Sum256(b)
	signature, err = rsa.SignPSS(rand.Reader, s.signing.PrivateKeyBytes, crypto.SHA256, hash[:], nil)
	if err != nil {
		return err
	}

	req, err = http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}

	req.Header.Set("X-Flomation-Runner-Signature", hex.EncodeToString(signature))

	_, err = client.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) createLogCallback(executionID string) executor.LogCallback {
	return func(lines []string) {
		payload := struct {
			Lines []string `json:"lines"`
		}{
			Lines: lines,
		}

		b, err := json.Marshal(payload)
		if err != nil {
			return
		}

		hash := sha256.Sum256(b)
		signature, err := rsa.SignPSS(rand.Reader, s.signing.PrivateKeyBytes, crypto.SHA256, hash[:], nil)
		if err != nil {
			return
		}

		url := fmt.Sprintf("%v/api/v1/execution/%v/logs", s.config.RunnerConfig.Server, executionID)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
		if err != nil {
			return
		}

		req.Header.Set("X-Flomation-Runner-Signature", hex.EncodeToString(signature))

		client := http.Client{Timeout: time.Second * 5}
		if _, err := client.Do(req); err != nil {
			log.WithFields(log.Fields{
				"error":        err,
				"execution_id": executionID,
			}).Warn("unable to stream log to API")
		}
	}
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
