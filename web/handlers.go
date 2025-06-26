package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/knownasmobin/singbox-checker/checker"
	"github.com/knownasmobin/singbox-checker/config"
	"github.com/knownasmobin/singbox-checker/metrics"
	"github.com/knownasmobin/singbox-checker/models"
)

var registeredEndpoints []EndpointInfo

// EndpointInfo contains metadata for a proxy endpoint exposed in the web UI.
type EndpointInfo struct {
	Name      string
	URL       string
	ProxyPort int
	Status    bool
	Latency   time.Duration
}

// IndexHandler returns an HTTP handler for the root dashboard page.
func IndexHandler(version string, proxyChecker *checker.ProxyChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		RegisterConfigEndpoints(proxyChecker.GetProxies(), proxyChecker, config.CLIConfig.SingBox.StartPort)

		data := PageData{
			Version:                    version,
			Host:                       config.CLIConfig.Metrics.Host,
			Port:                       config.CLIConfig.Metrics.Port,
			CheckInterval:              config.CLIConfig.Proxy.CheckInterval,
			IPCheckUrl:                 config.CLIConfig.Proxy.IpCheckUrl,
			CheckMethod:                config.CLIConfig.Proxy.CheckMethod,
			StatusCheckUrl:             config.CLIConfig.Proxy.StatusCheckUrl,
			SimulateLatency:            config.CLIConfig.Proxy.SimulateLatency,
			Timeout:                    config.CLIConfig.Proxy.Timeout,
			SubscriptionUpdate:         config.CLIConfig.Subscription.Update,
			SubscriptionUpdateInterval: config.CLIConfig.Subscription.UpdateInterval,
			StartPort:                  config.CLIConfig.SingBox.StartPort,
			ConfigDir:                  config.CLIConfig.SingBox.ConfigDir,
			Instance:                   config.CLIConfig.Metrics.Instance,
			PushUrl:                    metrics.GetPushURL(config.CLIConfig.Metrics.PushURL),
			Endpoints:                  registeredEndpoints,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := RenderIndex(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

func BasicAuthMiddleware(username, password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || user != username || pass != password {
				w.Header().Set("WWW-Authenticate", `Basic realm="metrics"`)
				http.Error(w, "Unauthorized.", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ConfigStatusHandler(proxyChecker *checker.ProxyChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[len("/config/"):]
		if path == "" {
			http.Error(w, "Config path is required", http.StatusBadRequest)
			return
		}

		found, exists := proxyChecker.GetProxyByStableID(path)
		if !exists {
			http.Error(w, "Config not found", http.StatusNotFound)
			return
		}

		status, latency, err := proxyChecker.GetProxyStatus(found.Name)
		if err != nil {
			http.Error(w, "Status not available", http.StatusNotFound)
			return
		}

		if config.CLIConfig.Proxy.SimulateLatency {
			time.Sleep(time.Duration(latency))
		}

		if status {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Failed"))
		}
	}
}

func RegisterConfigEndpoints(proxies []*models.ProxyConfig, proxyChecker *checker.ProxyChecker, startPort int) {
	registeredEndpoints = make([]EndpointInfo, 0, len(proxies))

	for i, proxy := range proxies {
		if proxy.StableID == "" {
			proxy.StableID = proxy.GenerateStableID()
		}

		// Ensure the proxy has an index
		proxy.Index = i

		endpoint := fmt.Sprintf("./config/%s", proxy.StableID)

		status, latency, _ := proxyChecker.GetProxyStatus(proxy.Name)

		registeredEndpoints = append(registeredEndpoints, EndpointInfo{
			Name:      fmt.Sprintf("%s (%s:%d)", proxy.Name, proxy.Server, proxy.Port),
			URL:       endpoint,
			ProxyPort: startPort + i,
			Status:    status,
			Latency:   latency,
		})
	}
}

type PrefixServeMux struct {
	prefix string
	mux    *http.ServeMux
}

func NewPrefixServeMux(prefix string) (*PrefixServeMux, error) {
	if strings.HasSuffix(prefix, "/") {
		return nil, fmt.Errorf("served url path prefix '%s' should not ends with a '/'", prefix)
	}
	return &PrefixServeMux{
		prefix: prefix,
		mux:    http.NewServeMux(),
	}, nil
}

func (pm *PrefixServeMux) Handle(pattern string, handler http.Handler) {
	pm.mux.Handle(pm.prefix+pattern, http.StripPrefix(pm.prefix, handler))
}

func (pm *PrefixServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == pm.prefix || strings.HasPrefix(r.URL.Path, pm.prefix+"/") {
		pm.mux.ServeHTTP(w, r)
	} else {
		http.NotFound(w, r)
	}
}
