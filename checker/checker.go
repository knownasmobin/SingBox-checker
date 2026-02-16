package checker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"xray-checker/logger"
	"xray-checker/metrics"
	"xray-checker/models"
)

// downloadBufPool reuses 8KB buffers for download checks to reduce GC pressure.
var downloadBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 8192)
	},
}

type ProxyChecker struct {
	proxies         []*models.ProxyConfig
	nameIndex       map[string]string              // name → stableID (protected by mu)
	idIndex         map[string]*models.ProxyConfig // stableID → proxy (protected by mu)
	startPort       int
	ipCheck         string
	currentIP       string
	httpClient      *http.Client
	geoClient       *http.Client // reusable client for direct GeoIP lookups
	currentMetrics  sync.Map     // stableID -> bool
	latencyMetrics  sync.Map     // stableID -> time.Duration
	previousStatus  sync.Map     // stableID -> bool
	geoCache        sync.Map     // stableID -> *GeoIPInfo
	latencyHistory  sync.Map     // stableID -> []int64
	proxyTransports sync.Map     // proxy.Index -> *http.Transport
	ipInitialized   bool
	ipCheckTimeout  int
	genMethodURL    string
	downloadURL     string
	downloadTimeout int
	downloadMinSize int64
	checkMethod     string
	mu              sync.RWMutex
	generation      uint64
}

func NewProxyChecker(proxies []*models.ProxyConfig, startPort int, ipCheckURL string, ipCheckTimeout int, genMethodURL string, downloadURL string, downloadTimeout int, downloadMinSize int64, checkMethod string) *ProxyChecker {
	ensureStableIDs(proxies)
	pc := &ProxyChecker{
		proxies:   proxies,
		startPort: startPort,
		ipCheck:   ipCheckURL,
		httpClient: &http.Client{
			Timeout: time.Second * time.Duration(ipCheckTimeout),
		},
		geoClient:       &http.Client{Timeout: 5 * time.Second},
		ipCheckTimeout:  ipCheckTimeout,
		genMethodURL:    genMethodURL,
		downloadURL:     downloadURL,
		downloadTimeout: downloadTimeout,
		downloadMinSize: downloadMinSize,
		checkMethod:     checkMethod,
	}
	pc.rebuildIndexLocked()
	return pc
}

// rebuildIndexLocked rebuilds the name→stableID and stableID→proxy index maps.
// Must be called under write lock (or during init before concurrent access).
func (pc *ProxyChecker) rebuildIndexLocked() {
	pc.nameIndex = make(map[string]string, len(pc.proxies))
	pc.idIndex = make(map[string]*models.ProxyConfig, len(pc.proxies))
	for _, p := range pc.proxies {
		pc.nameIndex[p.Name] = p.StableID
		pc.idIndex[p.StableID] = p
	}
}

// getOrCreateTransport returns a cached or newly created Transport for the proxy.
func (pc *ProxyChecker) getOrCreateTransport(proxy *models.ProxyConfig) *http.Transport {
	if val, ok := pc.proxyTransports.Load(proxy.Index); ok {
		return val.(*http.Transport)
	}
	proxyURL, err := url.Parse(fmt.Sprintf("socks5://127.0.0.1:%d", pc.startPort+proxy.Index))
	if err != nil {
		logger.Error("Error parsing proxy URL for %s: %v", proxy.Name, err)
		return nil
	}
	transport := &http.Transport{
		Proxy:             http.ProxyURL(proxyURL),
		DisableKeepAlives: true,
	}
	actual, _ := pc.proxyTransports.LoadOrStore(proxy.Index, transport)
	return actual.(*http.Transport)
}

// ensureStableIDs generates StableIDs for all proxies that don't have one.
func ensureStableIDs(proxies []*models.ProxyConfig) {
	for _, p := range proxies {
		if p.StableID == "" {
			p.StableID = p.GenerateStableID()
		}
	}
}

func (pc *ProxyChecker) GetCurrentIP() (string, error) {
	if pc.ipInitialized && pc.currentIP != "" {
		return pc.currentIP, nil
	}

	resp, err := pc.httpClient.Get(pc.ipCheck)
	if err != nil {
		return "", fmt.Errorf("error getting current IP: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}

	pc.currentIP = string(body)
	pc.ipInitialized = true
	return pc.currentIP, nil
}

func (pc *ProxyChecker) CheckProxy(proxy *models.ProxyConfig) {
	pc.checkProxyInternal(proxy, 0, false)
}

func (pc *ProxyChecker) checkProxyInternal(proxy *models.ProxyConfig, expectedGeneration uint64, checkGeneration bool) {
	stableID := proxy.StableID
	serverAddr := fmt.Sprintf("%s:%d", proxy.Server, proxy.Port)

	isGenerationValid := func() bool {
		if !checkGeneration {
			return true
		}
		return atomic.LoadUint64(&pc.generation) == expectedGeneration
	}

	setFailedStatus := func() {
		if !isGenerationValid() {
			logger.Debug("%s | Skipping metric update: generation changed", proxy.Name)
			return
		}
		metrics.RecordProxyStatus(proxy.Protocol, serverAddr, proxy.Name, proxy.SubName, 0)
		pc.currentMetrics.Store(stableID, false)
	}

	setFailedLatency := func() {
		if !isGenerationValid() {
			return
		}
		metrics.RecordProxyLatency(proxy.Protocol, serverAddr, proxy.Name, proxy.SubName, time.Duration(0))
		pc.latencyMetrics.Store(stableID, time.Duration(0))
	}

	transport := pc.getOrCreateTransport(proxy)
	if transport == nil {
		setFailedStatus()
		setFailedLatency()
		return
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Second * time.Duration(pc.ipCheckTimeout),
	}

	var checkSuccess bool
	var checkErr error
	var logMessage string
	var latency time.Duration
	var exitIP string

	if pc.checkMethod == "ip" {
		checkSuccess, logMessage, latency, exitIP, checkErr = pc.checkByIP(client)
	} else if pc.checkMethod == "status" {
		checkSuccess, logMessage, latency, checkErr = pc.checkByGen(client)
	} else if pc.checkMethod == "download" {
		checkSuccess, logMessage, latency, checkErr = pc.checkByDownload(client)
	} else {
		logger.Error("Invalid check method: %s", pc.checkMethod)
		return
	}

	if checkErr != nil {
		logger.Error("%s | %v", proxy.Name, checkErr)
		setFailedStatus()
		setFailedLatency()

		return
	}

	if !checkSuccess {
		logger.Error("%s | Failed | %s | Latency: %s", proxy.Name, logMessage, latency)
		setFailedStatus()
		setFailedLatency()
		pc.previousStatus.Store(stableID, false)
	} else {
		logger.Result("%s | Success | %s | Latency: %s", proxy.Name, logMessage, latency)
		if !isGenerationValid() {
			logger.Debug("%s | Skipping metric update: generation changed", proxy.Name)
			return
		}

		// Check if proxy just came online (was previously offline or never checked)
		wasOnline := false
		if prev, ok := pc.previousStatus.Load(stableID); ok {
			wasOnline = prev.(bool)
		}

		// Look up GeoIP when:
		// 1. Proxy just came online (transition from offline), OR
		// 2. Proxy is online but has no cached GeoIP info yet
		needsGeoIP := !wasOnline
		if !needsGeoIP {
			if _, hasCached := pc.geoCache.Load(stableID); !hasCached {
				needsGeoIP = true
			}
		}
		if needsGeoIP {
			if exitIP != "" {
				go pc.updateProxyGeoIP(proxy, exitIP)
			} else if pc.ipCheck != "" {
				go pc.discoverAndUpdateGeoIP(proxy, client)
			}
		}

		metrics.RecordProxyStatus(proxy.Protocol, serverAddr, proxy.Name, proxy.SubName, 1)
		metrics.RecordProxyLatency(proxy.Protocol, serverAddr, proxy.Name, proxy.SubName, latency)

		pc.latencyMetrics.Store(stableID, latency)
		pc.currentMetrics.Store(stableID, true)
		pc.previousStatus.Store(stableID, true)

		// Append to latency history ring buffer (max 10 entries)
		if ms := latency.Milliseconds(); ms > 0 {
			var history []int64
			if val, ok := pc.latencyHistory.Load(stableID); ok {
				history = val.([]int64)
			}
			if len(history) >= 10 {
				copy(history, history[1:])
				history[9] = ms
			} else {
				history = append(history, ms)
			}
			pc.latencyHistory.Store(stableID, history)
		}
	}
}

// checkByIP returns: success, logMessage, latency, exitIP, error
func (pc *ProxyChecker) checkByIP(client *http.Client) (bool, string, time.Duration, string, error) {
	req, err := http.NewRequest("GET", pc.ipCheck, nil)
	if err != nil {
		return false, "", 0, "", err
	}

	var ttfb time.Duration
	start := time.Now()
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			ttfb = time.Since(start)
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(context.Background(), trace))

	resp, err := client.Do(req)
	if err != nil {
		return false, "", 0, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, "", ttfb, "", err
	}

	proxyIP := strings.TrimSpace(string(body))
	logMessage := fmt.Sprintf("Source IP: %s | Proxy IP: %s", pc.currentIP, proxyIP)
	return proxyIP != pc.currentIP, logMessage, ttfb, proxyIP, nil
}

func (pc *ProxyChecker) checkByGen(client *http.Client) (bool, string, time.Duration, error) {
	req, err := http.NewRequest("GET", pc.genMethodURL, nil)
	if err != nil {
		return false, "", 0, err
	}

	var ttfb time.Duration
	start := time.Now()
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			ttfb = time.Since(start)
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(context.Background(), trace))

	resp, err := client.Do(req)
	if err != nil {
		return false, "", 0, err
	}
	defer resp.Body.Close()

	logMessage := fmt.Sprintf("Status: %d", resp.StatusCode)
	return resp.StatusCode >= 200 && resp.StatusCode < 300, logMessage, ttfb, nil
}

func (pc *ProxyChecker) checkByDownload(client *http.Client) (bool, string, time.Duration, error) {
	if pc.downloadURL == "" {
		return false, "Download URL not configured", 0, fmt.Errorf("download URL not configured")
	}

	req, err := http.NewRequest("GET", pc.downloadURL, nil)
	if err != nil {
		return false, "", 0, err
	}

	var ttfb time.Duration
	start := time.Now()
	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			ttfb = time.Since(start)
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(context.Background(), trace))

	downloadClient := &http.Client{
		Transport: client.Transport,
		Timeout:   time.Second * time.Duration(pc.downloadTimeout),
	}

	resp, err := downloadClient.Do(req)
	if err != nil {
		return false, "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Sprintf("HTTP status: %d", resp.StatusCode), ttfb, nil
	}

	totalBytes := int64(0)
	buffer := downloadBufPool.Get().([]byte)
	defer downloadBufPool.Put(buffer)

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			totalBytes += int64(n)
		}

		if totalBytes >= pc.downloadMinSize {
			break
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return false, fmt.Sprintf("Download error after %d bytes: %v", totalBytes, err), ttfb, nil
		}
	}

	success := totalBytes >= pc.downloadMinSize
	logMessage := fmt.Sprintf("Downloaded: %d bytes (min: %d)", totalBytes, pc.downloadMinSize)

	return success, logMessage, ttfb, nil
}

// clearMetricsLocked clears Prometheus metrics and sync.Maps.
// Must be called under write lock.
func (pc *ProxyChecker) clearMetricsLocked() {
	for _, proxy := range pc.proxies {
		serverAddr := fmt.Sprintf("%s:%d", proxy.Server, proxy.Port)
		metrics.DeleteProxyStatus(proxy.Protocol, serverAddr, proxy.Name, proxy.SubName)
		metrics.DeleteProxyLatency(proxy.Protocol, serverAddr, proxy.Name, proxy.SubName)
	}

	pc.currentMetrics.Range(func(key, _ interface{}) bool {
		pc.currentMetrics.Delete(key)
		return true
	})
	pc.latencyMetrics.Range(func(key, _ interface{}) bool {
		pc.latencyMetrics.Delete(key)
		return true
	})
}

func (pc *ProxyChecker) UpdateProxies(newProxies []*models.ProxyConfig) {
	ensureStableIDs(newProxies)
	pc.mu.Lock()
	defer pc.mu.Unlock()
	atomic.AddUint64(&pc.generation, 1)
	pc.clearMetricsLocked()
	pc.proxies = newProxies
	pc.rebuildIndexLocked()
	// Clear cached transports (proxy ports may have changed)
	pc.proxyTransports.Range(func(key, _ interface{}) bool {
		pc.proxyTransports.Delete(key)
		return true
	})
}

func (pc *ProxyChecker) CheckAllProxies() {
	if _, err := pc.GetCurrentIP(); err != nil {
		logger.Warn("Error getting current IP: %v", err)
		return
	}

	pc.mu.RLock()
	proxiesToCheck := make([]*models.ProxyConfig, len(pc.proxies))
	copy(proxiesToCheck, pc.proxies)
	currentGeneration := atomic.LoadUint64(&pc.generation)
	pc.mu.RUnlock()

	var wg sync.WaitGroup
	for _, proxy := range proxiesToCheck {
		wg.Add(1)
		go func(p *models.ProxyConfig, gen uint64) {
			defer wg.Done()
			pc.checkProxyInternal(p, gen, true)
		}(proxy, currentGeneration)
	}
	wg.Wait()
}

// GetProxyStatus looks up status and latency by proxy name.
func (pc *ProxyChecker) GetProxyStatus(name string) (bool, time.Duration, error) {
	pc.mu.RLock()
	stableID := pc.nameIndex[name]
	pc.mu.RUnlock()

	if stableID == "" {
		return false, 0, fmt.Errorf("proxy not found")
	}

	status, ok := pc.currentMetrics.Load(stableID)
	if !ok {
		return false, 0, fmt.Errorf("metric not found")
	}

	latency, _ := pc.latencyMetrics.Load(stableID)
	if latency == nil {
		latency = time.Duration(0)
	}

	return status.(bool), latency.(time.Duration), nil
}

func (pc *ProxyChecker) GetProxyByStableID(stableID string) (*models.ProxyConfig, bool) {
	pc.mu.RLock()
	proxy := pc.idIndex[stableID]
	pc.mu.RUnlock()
	if proxy == nil {
		return nil, false
	}
	return proxy, true
}

func (pc *ProxyChecker) GetProxies() []*models.ProxyConfig {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	result := make([]*models.ProxyConfig, len(pc.proxies))
	copy(result, pc.proxies)
	return result
}

// storeGeoInfo stores GeoIP info in cache and updates the proxy config.
func (pc *ProxyChecker) storeGeoInfo(proxy *models.ProxyConfig, geoInfo *GeoIPInfo) {
	pc.geoCache.Store(proxy.StableID, geoInfo)

	pc.mu.Lock()
	if p := pc.idIndex[proxy.StableID]; p != nil {
		p.CountryCode = geoInfo.CountryCode
		p.Country = geoInfo.Country
	}
	pc.mu.Unlock()
}

// ensureProxyGeoCode ensures the proxy object has the country code from cached geo info.
// Returns true if cache was fresh and no new lookup is needed.
func (pc *ProxyChecker) ensureProxyGeoCode(proxy *models.ProxyConfig) bool {
	cached, ok := pc.geoCache.Load(proxy.StableID)
	if !ok {
		return false
	}
	geoInfo := cached.(*GeoIPInfo)
	if time.Since(geoInfo.LastChecked) >= time.Hour {
		return false
	}

	pc.mu.Lock()
	if p := pc.idIndex[proxy.StableID]; p != nil && p.CountryCode == "" {
		p.CountryCode = geoInfo.CountryCode
		p.Country = geoInfo.Country
	}
	pc.mu.Unlock()
	return true
}

// updateProxyGeoIP looks up GeoIP info for a known exit IP (used by "ip" check method).
func (pc *ProxyChecker) updateProxyGeoIP(proxy *models.ProxyConfig, exitIP string) {
	// Check if cached info is still fresh for this IP
	if cached, ok := pc.geoCache.Load(proxy.StableID); ok {
		geoInfo := cached.(*GeoIPInfo)
		if geoInfo.IP == exitIP && time.Since(geoInfo.LastChecked) < time.Hour {
			pc.ensureProxyGeoCode(proxy)
			return
		}
	}

	geoInfo, err := LookupGeoIP(exitIP, pc.geoClient)
	if err != nil {
		logger.Debug("%s | GeoIP lookup failed: %v", proxy.Name, err)
		return
	}

	pc.storeGeoInfo(proxy, geoInfo)
	logger.Debug("%s | GeoIP: %s (%s) from IP %s", proxy.Name, geoInfo.Country, geoInfo.CountryCode, exitIP)
}

// discoverAndUpdateGeoIP calls ip-api.com through the proxy connection to discover
// exit IP and country in a single request. Used for non-IP check methods.
func (pc *ProxyChecker) discoverAndUpdateGeoIP(proxy *models.ProxyConfig, proxyClient *http.Client) {
	if pc.ensureProxyGeoCode(proxy) {
		return
	}

	geoInfo, err := LookupGeoIPViaClient(proxyClient)
	if err != nil {
		logger.Debug("%s | GeoIP via proxy failed: %v", proxy.Name, err)
		return
	}

	pc.storeGeoInfo(proxy, geoInfo)
	logger.Debug("%s | GeoIP: %s (%s) from exit IP %s", proxy.Name, geoInfo.Country, geoInfo.CountryCode, geoInfo.IP)
}

// GetProxyGeoInfo returns the cached GeoIP info for a proxy
func (pc *ProxyChecker) GetProxyGeoInfo(stableID string) *GeoIPInfo {
	if cached, ok := pc.geoCache.Load(stableID); ok {
		return cached.(*GeoIPInfo)
	}
	return nil
}

// GetLatencyHistory returns the recent latency history for a proxy
func (pc *ProxyChecker) GetLatencyHistory(stableID string) []int64 {
	if val, ok := pc.latencyHistory.Load(stableID); ok {
		history := val.([]int64)
		result := make([]int64, len(history))
		copy(result, history)
		return result
	}
	return nil
}
