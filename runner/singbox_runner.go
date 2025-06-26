package runner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/knownasmobin/singbox-checker/models"
)

type SingBoxRunner struct {
	configFile string
	cmd       *exec.Cmd
	mu        sync.Mutex
	running   bool
}

func NewSingBoxRunner(configFile string) *SingBoxRunner {
	return &SingBoxRunner{
		configFile: configFile,
	}
}

func (r *SingBoxRunner) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("singbox is already running")
	}

	// Ensure config file exists
	if _, err := os.Stat(r.configFile); os.IsNotExist(err) {
		// Create default config if it doesn't exist
		defaultConfig := models.NewDefaultSingBoxConfig()
		configData, err := json.MarshalIndent(defaultConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal default config: %v", err)
		}

		if err := os.MkdirAll(filepath.Dir(r.configFile), 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %v", err)
		}

		if err := os.WriteFile(r.configFile, configData, 0644); err != nil {
			return fmt.Errorf("failed to write default config: %w", err)
		}
	}

	// Start singbox process
	r.cmd = exec.Command("sing-box", "run", "-c", r.configFile)
	
	// Redirect output to stdout/stderr
	r.cmd.Stdout = os.Stdout
	r.cmd.Stderr = os.Stderr

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start singbox: %v", err)
	}

	r.running = true
	go r.waitForExit()

	// Wait a bit to ensure the process is running
	time.Sleep(1 * time.Second)
	if r.cmd.ProcessState != nil && r.cmd.ProcessState.Exited() {
		return fmt.Errorf("singbox process exited immediately")
	}

	return nil
}

func (r *SingBoxRunner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running || r.cmd == nil || r.cmd.Process == nil {
		return nil
	}

	// Send SIGTERM to the process
	if err := r.cmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("failed to send interrupt signal: %v", err)
	}

	// Wait for the process to exit, but don't wait forever
	done := make(chan error, 1)
	go func() {
		done <- r.cmd.Wait()
	}()

	select {
	case <-time.After(5 * time.Second):
		// Process didn't exit in time, kill it
		if err := r.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill singbox process: %v", err)
		}
		return fmt.Errorf("singbox process did not exit gracefully and was killed")
	case err := <-done:
		if err != nil {
			return fmt.Errorf("singbox process exited with error: %v", err)
		}
	}

	r.running = false
	return nil
}

func (r *SingBoxRunner) Restart() error {
	if err := r.Stop(); err != nil {
		return fmt.Errorf("failed to stop singbox: %v", err)
	}
	return r.Start()
}

func (r *SingBoxRunner) waitForExit() {
	err := r.cmd.Wait()
	
	r.mu.Lock()
	r.running = false
	r.mu.Unlock()

	// Only print error if not interrupted by signal
	if err != nil && err.Error() != "signal: interrupt" {
		fmt.Printf("SingBox process exited with error: %v\n", err)
	}
}

func (r *SingBoxRunner) GetConfig() (*models.SingBoxConfig, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Read the config file
	configData, err := os.ReadFile(r.configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the config
	var config models.SingBoxConfig
	if err := json.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	return &config, nil
}

// UpdateConfig updates the SingBox config.
func (r *SingBoxRunner) UpdateConfig(config *models.SingBoxConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Marshal the config to JSON
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// Write the config to a temporary file
	tempFile := r.configFile + ".tmp"
	if err := os.WriteFile(tempFile, configData, 0644); err != nil {
		return fmt.Errorf("failed to write temp config: %v", err)
	}

	// Replace the old config with the new one
	if err := os.Rename(tempFile, r.configFile); err != nil {
		return fmt.Errorf("failed to update config: %v", err)
	}

	// If SingBox is running, send SIGHUP to reload the config
	if r.running && r.cmd != nil && r.cmd.Process != nil {
		return r.cmd.Process.Signal(os.Interrupt)
	}

	return nil
}

func (r *SingBoxRunner) GetAPIClient() (*SingBoxAPIClient, error) {
	// TODO: Implement API client for controlling singbox
	return nil, fmt.Errorf("not implemented")
}

type SingBoxAPIClient struct {
	baseURL string
	client  *http.Client
}

func NewSingBoxAPIClient(baseURL string) *SingBoxAPIClient {
	return &SingBoxAPIClient{
		baseURL: trimSuffix(baseURL, "/"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// trimSuffix is a minimal replacement for strings.TrimSuffix to avoid importing the whole package for one use.
func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}


func (c *SingBoxAPIClient) GetOutbounds() ([]interface{}, error) {
	// TODO: Implement API call to get outbounds
	return nil, fmt.Errorf("not implemented")
}

func (c *SingBoxAPIClient) AddOutbound(outbound interface{}) error {
	// TODO: Implement API call to add outbound
	return fmt.Errorf("not implemented")
}

func (c *SingBoxAPIClient) DeleteOutbound(tag string) error {
	// TODO: Implement API call to delete outbound
	return fmt.Errorf("not implemented")
}

func (c *SingBoxAPIClient) doRequest(method, path string, body io.Reader, result interface{}) error {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}
