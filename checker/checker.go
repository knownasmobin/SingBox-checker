package checker

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/knownasmobin/singbox-checker/metrics"
	"github.com/knownasmobin/singbox-checker/models"
)

// ProxyChecker checks the status and latency of proxies, manages metrics, and fetches current IP.
// currentMetrics and latencyMetrics use sync.Map for concurrent access from checker goroutines and web handlers.
type ProxyChecker struct {
	proxies        []*models.ProxyConfig
	startPort      int
	ipCheck        string
	currentIP      string
	httpClient     *http.Client
	currentMetrics sync.Map // map[string]bool
	latencyMetrics sync.Map // map[string]time.Duration
	ipInitialized  bool
	ipCheckTimeout int
	genMethodURL   string
	checkMethod    string
	instance       string
}

// NewProxyChecker creates a new ProxyChecker instance.
func NewProxyChecker(proxies []*models.ProxyConfig, startPort int, ipCheckURL string, ipCheckTimeout int, genMethodURL string, checkMethod string, instance string) *ProxyChecker {
	return &ProxyChecker{
		proxies:   proxies,
		startPort: startPort,
		ipCheck:   ipCheckURL,
		httpClient: &http.Client{
			Timeout: time.Second * time.Duration(ipCheckTimeout),
		},
		ipCheckTimeout: ipCheckTimeout,
		genMethodURL:   genMethodURL,
		checkMethod:    checkMethod,
		instance:       instance,
	}
}

// GetCurrentIP fetches and caches the current public IP address.
func (pc *ProxyChecker) GetCurrentIP() (string, error) {
	if pc.ipInitialized && pc.currentIP != "" {
		return pc.currentIP, nil
	}

	resp, err := pc.httpClient.Get(pc.ipCheck)
	if err != nil {
		return "", fmt.Errorf("get current IP from %s: %w", pc.ipCheck, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read IP response: %w", err)
	}

	pc.currentIP = string(body)
	pc.ipInitialized = true
	return pc.currentIP, nil
}

func (pc *ProxyChecker) CheckProxy(proxy *models.ProxyConfig) {
	if proxy.StableID == "" {
		proxy.StableID = proxy.GenerateStableID()
	}

	metricKey := fmt.Sprintf("%s|%s:%d|%s|%s",
		proxy.Protocol,
		proxy.Server,
		proxy.Port,
		proxy.Name,
		proxy.StableID,
	)

	setFailedStatus := func() {
		metrics.RecordProxyStatus(
			proxy.Protocol,
			fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
			proxy.Name,
			0,
			pc.instance,
		)
		pc.currentMetrics.Store(metricKey, false)
	}

	setFailedLatency := func() {
		metrics.RecordProxyLatency(
			proxy.Protocol,
			fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
			proxy.Name,
			time.Duration(0),
			pc.instance,
		)
		pc.latencyMetrics.Store(metricKey, time.Duration(0))
	}

	proxyURL := fmt.Sprintf("socks5://127.0.0.1:%d", pc.startPort+proxy.Index)
	proxyURLParsed, err := url.Parse(proxyURL)
	if err != nil {
		log.Printf("Error parsing proxy URL %s: %v", proxyURL, err)
		setFailedStatus()
		setFailedLatency()

		return
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyURL(proxyURLParsed),
			DisableKeepAlives: true,
		},
		Timeout: time.Second * time.Duration(pc.ipCheckTimeout),
	}

	var checkSuccess bool
	var checkErr error
	var logMessage string

	start := time.Now()

	if pc.checkMethod == "ip" {
		checkSuccess, logMessage, checkErr = pc.checkByIP(client)
	} else if pc.checkMethod == "status" {
		checkSuccess, logMessage, checkErr = pc.checkByGen(client)
	} else {
		log.Printf("Invalid check method: %s", pc.checkMethod)
		return
	}

	latency := time.Since(start)

	if checkErr != nil {
		log.Printf("%s | Error | %v", proxy.Name, checkErr)
		setFailedStatus()
		setFailedLatency()

		return
	}

	if !checkSuccess {
		log.Printf("%s | Failed | %s | Latency: %s", proxy.Name, logMessage, latency)
		setFailedStatus()
		setFailedLatency()
	} else {
		log.Printf("%s | Success | %s | Latency: %s", proxy.Name, logMessage, latency)
		metrics.RecordProxyStatus(
			proxy.Protocol,
			fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
			proxy.Name,
			1,
			pc.instance,
		)
		metrics.RecordProxyLatency(
			proxy.Protocol,
			fmt.Sprintf("%s:%d", proxy.Server, proxy.Port),
			proxy.Name,
			latency,
			pc.instance,
		)

		pc.latencyMetrics.Store(metricKey, latency)
		pc.currentMetrics.Store(metricKey, true)
	}
}

func (pc *ProxyChecker) checkByIP(client *http.Client) (bool, string, error) {
	resp, err := client.Get(pc.ipCheck)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", err
	}

	proxyIP := string(body)
	logMessage := fmt.Sprintf("Source IP: %s | Proxy IP: %s", pc.currentIP, proxyIP)
	return proxyIP != pc.currentIP, logMessage, nil
}

func (pc *ProxyChecker) checkByGen(client *http.Client) (bool, string, error) {
	resp, err := client.Get(pc.genMethodURL)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	logMessage := fmt.Sprintf("Status: %d", resp.StatusCode)
	return resp.StatusCode >= 200 && resp.StatusCode < 300, logMessage, nil
}

func (pc *ProxyChecker) ClearMetrics() {
	pc.currentMetrics.Range(func(key, _ interface{}) bool {
		metricKey := key.(string)
		parts := strings.Split(metricKey, "|")
		if len(parts) >= 3 {
			metrics.DeleteProxyStatus(parts[0], parts[1], parts[2], pc.instance)
			metrics.DeleteProxyLatency(parts[0], parts[1], parts[2], pc.instance)
		}
		pc.currentMetrics.Delete(key)
		return true
	})

	pc.latencyMetrics.Range(func(key, _ interface{}) bool {
		pc.latencyMetrics.Delete(key)
		return true
	})
}

func (pc *ProxyChecker) UpdateProxies(newProxies []*models.ProxyConfig) {
	pc.ClearMetrics()
	pc.proxies = newProxies
}

func (pc *ProxyChecker) CheckAllProxies() {
	if _, err := pc.GetCurrentIP(); err != nil {
		log.Printf("Error getting current IP: %v", err)
		return
	}

	var wg sync.WaitGroup
	for _, proxy := range pc.proxies {
		wg.Add(1)
		go func(p *models.ProxyConfig) {
			defer wg.Done()
			pc.CheckProxy(p)
		}(proxy)
	}
	wg.Wait()
}

func (pc *ProxyChecker) GetProxyStatus(name string) (bool, time.Duration, error) {
	var metricKey string
	for _, proxy := range pc.proxies {
		if proxy.Name == name {
			if proxy.StableID == "" {
				proxy.StableID = proxy.GenerateStableID()
			}

			metricKey = fmt.Sprintf("%s|%s:%d|%s|%s",
				proxy.Protocol,
				proxy.Server,
				proxy.Port,
				proxy.Name,
				proxy.StableID,
			)
			break
		}
	}

	if metricKey == "" {
		return false, 0, fmt.Errorf("proxy not found")
	}

	status, ok := pc.currentMetrics.Load(metricKey)
	if !ok {
		return false, 0, fmt.Errorf("metric not found")
	}

	latency, _ := pc.latencyMetrics.Load(metricKey)
	if latency == nil {
		latency = time.Duration(0)
	}

	return status.(bool), latency.(time.Duration), nil
}

func (pc *ProxyChecker) GetProxyByStableID(stableID string) (*models.ProxyConfig, bool) {
	for _, proxy := range pc.proxies {
		if proxy.StableID == "" {
			proxy.StableID = proxy.GenerateStableID()
		}

		if proxy.StableID == stableID {
			return proxy, true
		}
	}
	return nil, false
}

func (pc *ProxyChecker) GetProxies() []*models.ProxyConfig {
	return pc.proxies
}
