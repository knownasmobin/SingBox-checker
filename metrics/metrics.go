package metrics

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"xray-checker/logger"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/common/expfmt"
)

type RemoteWriteConfig struct {
	URL      string
	Username string
	Password string
	Timeout  time.Duration
}

var (
	proxyStatus   *prometheus.GaugeVec
	proxyLatency  *prometheus.GaugeVec
	defaultLabels = []string{"protocol", "address", "name"}
)

func InitMetrics(instance string) {
	labels := defaultLabels
	if instance != "" {
		labels = append(labels, "instance")
	}

	proxyStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "xray_proxy_status",
			Help: "Status of proxy connection (1: success, 0: failure)",
		},
		labels,
	)

	proxyLatency = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "xray_proxy_latency_ms",
			Help: "Latency of proxy connection in milliseconds, 0 if failed",
		},
		labels,
	)
}

func GetProxyStatusMetric() *prometheus.GaugeVec {
	return proxyStatus
}

func GetProxyLatencyMetric() *prometheus.GaugeVec {
	return proxyLatency
}

func RecordProxyStatus(protocol, address, name string, value float64, instance string) {
	if instance != "" {
		proxyStatus.WithLabelValues(protocol, address, name, instance).Set(value)
	} else {
		proxyStatus.WithLabelValues(protocol, address, name).Set(value)
	}
}

func RecordProxyLatency(protocol, address, name string, value time.Duration, instance string) {
	if instance != "" {
		proxyLatency.WithLabelValues(protocol, address, name, instance).Set(float64(value.Milliseconds()))
	} else {
		proxyLatency.WithLabelValues(protocol, address, name).Set(float64(value.Milliseconds()))
	}
}

func DeleteProxyStatus(protocol, address, name string, instance string) {
	if instance != "" {
		proxyStatus.DeleteLabelValues(protocol, address, name, instance)
	} else {
		proxyStatus.DeleteLabelValues(protocol, address, name)
	}
}

func DeleteProxyLatency(protocol, address, name string, instance string) {
	if instance != "" {
		proxyLatency.DeleteLabelValues(protocol, address, name, instance)
	} else {
		proxyLatency.DeleteLabelValues(protocol, address, name)
	}
}

func ParseURL(remoteWriteURL string) (*RemoteWriteConfig, error) {
	if remoteWriteURL == "" {
		return nil, nil
	}

	u, err := url.Parse(remoteWriteURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %v", err)
	}

	config := &RemoteWriteConfig{
		Timeout: 10 * time.Second,
	}

	if u.User != nil {
		config.Username = u.User.Username()
		if password, ok := u.User.Password(); ok {
			config.Password = password
		}
		u.User = nil
	}

	config.URL = u.String()
	return config, nil
}

func PushMetrics(config *RemoteWriteConfig, registry *prometheus.Registry) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	metricFamilies, err := registry.Gather()
	if err != nil {
		return fmt.Errorf("failed to gather metrics: %v", err)
	}

	var buf bytes.Buffer
	encoder := expfmt.NewEncoder(&buf, expfmt.FmtText)

	for _, mf := range metricFamilies {
		if err := encoder.Encode(mf); err != nil {
			return fmt.Errorf("failed to encode metrics: %v", err)
		}
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	req, err := http.NewRequest("POST", config.URL, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	if config.Username != "" && config.Password != "" {
		req.SetBasicAuth(config.Username, config.Password)
	}

	req.Header.Set("Content-Type", "text/plain; version=0.0.4")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}
	logger.Debug("Metrics pushed to %s", config.URL)

	return nil
}

func GetPushURL(url string) string {
	if url == "" {
		return ""
	}

	cfg, err := ParseURL(url)
	if err != nil || cfg == nil {
		return ""
	}

	return cfg.URL
}
