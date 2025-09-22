---
title: Environment Variables
description: Environment variables for Xray Checker
---

## Subscription

### SUBSCRIPTION_URL

- CLI: `--subscription-url`
- Required: Yes
- Default: None

URL, Base64 string or file path for proxy configuration. Supports multiple formats:

- HTTP/HTTPS URL with Base64 encoded content
- Direct Base64 encoded string
- Local file path with prefix `file://`
- Local folder path with prefix `folder://`

### SUBSCRIPTION_UPDATE

- CLI: `--subscription-update`
- Required: No
- Default: `true`

Enables automatic updates of proxy configuration from subscription source. When enabled, Xray Checker will periodically check for changes and update configurations accordingly.

### SUBSCRIPTION_UPDATE_INTERVAL

- CLI: `--subscription-update-interval`
- Required: No
- Default: `300`

Time in seconds between subscription update checks. Only used when `SUBSCRIPTION_UPDATE` is enabled.

## Proxy

### PROXY_CHECK_INTERVAL

- CLI: `--proxy-check-interval`
- Required: No
- Default: `300`

Time in seconds between proxy availability checks. Each check verifies all configured proxies.

### PROXY_CHECK_METHOD

- CLI: `--proxy-check-method`
- Required: No
- Default: `ip`
- Values: `ip`, `status`, `download`

Method used to verify proxy functionality:

- `ip`: Compares IP addresses with and without proxy
- `status`: Checks HTTP status code from a test request
- `download`: Downloads a file through proxy to verify functionality

### PROXY_IP_CHECK_URL

- CLI: `--proxy-ip-check-url`
- Required: No
- Default: `https://api.ipify.org?format=text`

URL used for IP verification when `PROXY_CHECK_METHOD=ip`. Should return current IP address in plain text format.

### PROXY_STATUS_CHECK_URL

- CLI: `--proxy-status-check-url`
- Required: No
- Default: `http://cp.cloudflare.com/generate_204`

URL used for status verification when `PROXY_CHECK_METHOD=status`. Should return HTTP 204/200 status code.

### PROXY_DOWNLOAD_URL

- CLI: `--proxy-download-url`
- Required: No
- Default: https://proof.ovh.net/files/1Mb.dat

URL of the file to download when `PROXY_CHECK_METHOD=download`. The file will be downloaded through the proxy to verify functionality.

### PROXY_DOWNLOAD_TIMEOUT

- CLI: `--proxy-download-timeout`
- Required: No
- Default: `60`

Maximum time in seconds to wait for file download completion when using `PROXY_CHECK_METHOD=download`.

### PROXY_DOWNLOAD_MIN_SIZE

- CLI: `--proxy-download-min-size`
- Required: No
- Default: `51200`

Minimum number of bytes to download for a successful check when using `PROXY_CHECK_METHOD=download`. Default is 50KB.

### PROXY_TIMEOUT

- CLI: `--proxy-timeout`
- Required: No
- Default: `30`

Maximum time in seconds to wait for proxy response during checks.

### SIMULATE_LATENCY

- CLI: `--simulate-latency`
- Required: No
- Default: `true`

Adds measured latency to endpoint responses, useful for monitoring systems.

## Xray

### XRAY_START_PORT

- CLI: `--xray-start-port`
- Required: No
- Default: `10000`

Starting port number for SOCKS5 proxies. Each proxy will use sequential ports starting from this number.

### XRAY_LOG_LEVEL

- CLI: `--xray-log-level`
- Required: No
- Default: `none`
- Values: `debug`, `info`, `warning`, `error`, `none`

Controls Xray Core logging verbosity.

## Metrics

### METRICS_HOST

- CLI: `--metrics-host`
- Required: No
- Default: `0.0.0.0`

Host address for metrics and status endpoints.

### METRICS_PORT

- CLI: `--metrics-port`
- Required: No
- Default: `2112`

Port number for HTTP server exposing metrics and status endpoints.

### METRICS_PROTECTED

- CLI: `--metrics-protected`
- Required: No
- Default: `false`

Enables basic authentication for metrics and status endpoints.

### METRICS_USERNAME

- CLI: `--metrics-username`
- Required: No
- Default: `metricsUser`

Username for basic authentication when `METRICS_PROTECTED=true`.

### METRICS_PASSWORD

- CLI: `--metrics-password`
- Required: No
- Default: `MetricsVeryHardPassword`

Password for basic authentication when `METRICS_PROTECTED=true`.

### METRICS_INSTANCE

- CLI: `--metrics-instance`
- Required: No
- Default: None

Instance label added to all metrics. Useful for distinguishing multiple Xray Checker instances.

### METRICS_PUSH_URL

- CLI: `--metrics-push-url`
- Required: No
- Default: None

Prometheus Pushgateway URL for metric pushing. Format: `https://user:pass@host:port`

### METRICS_BASE_PATH

- CLI: `--metrics-base-path`
- Required: No
- Default: ""

URL path for host metrics and monitoring. Format: `/vpn/metrics`. Monitoring page could be available on `http://localhost:port/metrics-base-path`

## Other

### RUN_ONCE

- CLI: `--run-once`
- Required: No
- Default: `false`

Performs single check cycle and exits. Useful for scheduled execution environments.
