package singbox

import (
	"context"
	"fmt"
	"os"

	"xray-checker/logger"

	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

type Runner struct {
	instance   *box.Box
	configFile string
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewRunner(configFile string) *Runner {
	return &Runner{
		configFile: configFile,
	}
}

func (r *Runner) Start() error {
	configBytes, err := os.ReadFile(r.configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	r.ctx, r.cancel = context.WithCancel(context.Background())

	options, err := json.UnmarshalExtendedContext[option.Options](r.ctx, configBytes)
	if err != nil {
		r.cancel()
		return fmt.Errorf("error decoding config: %v", err)
	}

	instance, err := box.New(box.Options{
		Context: r.ctx,
		Options: options,
	})
	if err != nil {
		r.cancel()
		return fmt.Errorf("error creating sing-box instance: %v", err)
	}

	if err := instance.Start(); err != nil {
		r.cancel()
		return fmt.Errorf("error starting sing-box: %v", err)
	}

	r.instance = instance
	logger.Debug("sing-box instance started")

	return nil
}

func (r *Runner) Stop() error {
	if r.instance != nil {
		err := r.instance.Close()
		r.instance = nil
		if r.cancel != nil {
			r.cancel()
		}
		if err != nil {
			return fmt.Errorf("error stopping sing-box: %v", err)
		}
		logger.Debug("sing-box instance stopped")
	}
	return nil
}
