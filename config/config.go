package config

import (
	"fmt"

	"github.com/alecthomas/kong"
)

var CLIConfig CLI

func Parse(version string) {
	ctx := kong.Parse(&CLIConfig,
		kong.Name("singbox-checker"),
		kong.Description("Singbox Checker: A Prometheus exporter for monitoring Singbox proxies"),
		kong.Vars{
			"version": version,
		},
	)
	_ = ctx
}

type CLI struct {
	Subscription struct {
		URL            string `name:"subscription-url" help:"URL of the subscription" required:"true" env:"SUBSCRIPTION_URL"`
		Update         bool   `name:"subscription-update" help:"Whether to recheck the subscription" default:"true" env:"SUBSCRIPTION_UPDATE"`
		UpdateInterval int    `name:"subscription-update-interval" help:"Interval for subscription updates in seconds" default:"300" env:"SUBSCRIPTION_UPDATE_INTERVAL"`
	} `embed:"" prefix:""`

	Proxy struct {
		CheckInterval   int    `name:"proxy-check-interval" help:"Interval for proxy checks in seconds" default:"300" env:"PROXY_CHECK_INTERVAL"`
		CheckMethod     string `name:"proxy-check-method" help:"Method for checking proxy, ip or status" default:"ip" env:"PROXY_CHECK_METHOD"`
		IpCheckUrl      string `name:"proxy-ip-check-url" help:"Service URL for IP checking" default:"https://api.ipify.org?format=text" env:"PROXY_IP_CHECK_URL"`
		StatusCheckUrl  string `name:"proxy-status-check-url" help:"Response status generator, used by check-method=status" default:"http://cp.cloudflare.com/generate_204" env:"PROXY_STATUS_CHECK_URL"`
		Timeout         int    `name:"proxy-timeout" help:"Timeout for IP checking in seconds" default:"30" env:"PROXY_TIMEOUT"`
		SimulateLatency bool   `name:"simulate-latency" help:"Whether to add latency to the response" default:"true" env:"SIMULATE_LATENCY"`
	} `embed:"" prefix:""`

	Singbox struct {
		StartPort int    `name:"singbox-start-port" help:"Start port for proxy configuration" default:"10000" env:"SINGBOX_START_PORT"`
		LogLevel  string `name:"singbox-log-level" help:"Singbox log level (debug|info|warning|error|none)" default:"none" env:"SINGBOX_LOG_LEVEL"`
	} `embed:"" prefix:""`

	Metrics struct {
		Host      string `name:"metrics-host" help:"Host to listen on" default:"0.0.0.0" env:"METRICS_HOST"`
		Port      string `name:"metrics-port" help:"Port to listen on" default:"2112" env:"METRICS_PORT"`
		Protected bool   `name:"metrics-protected" help:"Whether metrics are protected by basic auth" default:"false" env:"METRICS_PROTECTED"`
		Username  string `name:"metrics-username" help:"Username for metrics if protected by basic auth" default:"metricsUser" env:"METRICS_USERNAME"`
		Password  string `name:"metrics-password" help:"Password for metrics if protected by basic auth" default:"MetricsVeryHardPassword" env:"METRICS_PASSWORD"`
		Instance  string `name:"metrics-instance" help:"Instance label for metrics" default:"" env:"METRICS_INSTANCE"`
		PushURL   string `name:"metrics-push-url" help:"Prometheus pushgateway URL (e.g. https://user:pass@host:port)" default:"" env:"METRICS_PUSH_URL"`
		BasePath  string `name:"metrics-base-path" help:"URL path to metrics (e.g. /singbox/metrics)" default:"" env:"METRICS_BASE_PATH"`
	} `embed:"" prefix:""`

	Version VersionFlag `name:"version" help:"Print version information and quit"`
	RunOnce bool        `name:"run-once" help:"Run one check cycle and exit" default:"false" env:"RUN_ONCE"`
}

type VersionFlag string

func (v VersionFlag) Decode(ctx *kong.DecodeContext) error { return nil }
func (v VersionFlag) IsBool() bool                         { return true }
func (v VersionFlag) BeforeApply(app *kong.Kong, vars kong.Vars) error {
	fmt.Println("Singbox Checker: A Prometheus exporter for monitoring Singbox proxies")
	fmt.Printf("Version:\t %s\n", vars["version"])
	fmt.Printf("GitHub: https://github.com/kutovoys/singbox-checker\n")
	app.Exit(0)
	return nil
}
