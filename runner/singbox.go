package runner

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

type SingboxRunner struct {
	cmd        *exec.Cmd
	configFile string
}

func NewSingboxRunner(configFile string) *SingboxRunner {
	return &SingboxRunner{configFile: configFile}
}

func (r *SingboxRunner) Start() error {
	r.cmd = exec.Command("sing-box", "run", "-c", r.configFile)
	r.cmd.Stdout = os.Stdout
	r.cmd.Stderr = os.Stderr

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("error starting sing-box: %v", err)
	}

	go func() {
		_ = r.cmd.Wait()
		r.cmd = nil
	}()

	log.Println("sing-box started successfully")
	return nil
}

func (r *SingboxRunner) Stop() error {
	if r.cmd != nil && r.cmd.Process != nil {
		if err := r.cmd.Process.Signal(os.Interrupt); err != nil {
			return fmt.Errorf("error stopping sing-box: %v", err)
		}
		_, _ = r.cmd.Process.Wait()
		r.cmd = nil
		log.Println("sing-box stopped successfully")
	}
	return nil
}

func (r *SingboxRunner) IsRunning() bool {
	return r.cmd != nil && r.cmd.ProcessState == nil
}
