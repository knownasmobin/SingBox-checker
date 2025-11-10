---
title: CLI
description: CLI usage of Xray Checker
---

### Basic Command Line Usage

The CLI interface provides complete control over Xray Checker's functionality through command-line arguments.

### Installation

Download the latest binary from releases:

```bash
# For Linux amd64
curl -sLo xray-checker.tar.gz https://github.com/kutovoys/xray-checker/releases/latest/download/xray-checker-$(curl -sI https://github.com/kutovoys/xray-checker/releases/latest/ | grep location | grep -Eo 'v([0-9]{1}\.?)+')-linux-amd64.tar.gz
tar -zxvf xray-checker.tar.gz
chmod +x xray-checker

# For Linux arm64
curl -sLo xray-checker.tar.gz https://github.com/kutovoys/xray-checker/releases/latest/download/xray-checker-$(curl -sI https://github.com/kutovoys/xray-checker/releases/latest/ | grep location | grep -Eo 'v([0-9]{1}\.?)+')-linux-arm64.tar.gz
tar -zxvf xray-checker.tar.gz
chmod +x xray-checker

# For macOS (Intel)
curl -sLo xray-checker.tar.gz https://github.com/kutovoys/xray-checker/releases/latest/download/xray-checker-$(curl -sI https://github.com/kutovoys/xray-checker/releases/latest/ | grep location | grep -Eo 'v([0-9]{1}\.?)+')-darwin-amd64.tar.gz
tar -zxvf xray-checker.tar.gz
chmod +x xray-checker

# For macOS (Silicon)
curl -sLo xray-checker.tar.gz https://github.com/kutovoys/xray-checker/releases/latest/download/xray-checker-$(curl -sI https://github.com/kutovoys/xray-checker/releases/latest/ | grep location | grep -Eo 'v([0-9]{1}\.?)+')-darwin-arm64.tar.gz
tar -zxvf xray-checker.tar.gz
chmod +x xray-checker
```

### Basic Usage

Minimum required configuration:

```bash
./xray-checker --subscription-url="https://your-subscription-url/sub"
```

### Full Configuration Example

```bash
./xray-checker \
  --subscription-url="https://your-subscription-url/sub" \
  --subscription-update=true \
  --subscription-update-interval=300 \
  --proxy-check-interval=300 \
  --proxy-timeout=5 \
  --proxy-check-method=ip \
  --proxy-ip-check-url="https://api.ipify.org?format=text" \
  --proxy-status-check-url="http://cp.cloudflare.com/generate_204" \
  --proxy-resolve-domains=false \
  --simulate-latency=true \
  --xray-start-port=10000 \
  --xray-log-level=none \
  --metrics-host=0.0.0.0 \
  --metrics-port=2112 \
  --metrics-protected=true \
  --metrics-username=custom_user \
  --metrics-password=custom_pass \
  --metrics-instance=node-1 \
  --metrics-push-url="https://push.example.com" \
  --metrics-base-path="/xray/monitor" \
  --run-once=false
```

### Common CLI Operations

Check version:

```bash
./xray-checker --version
```

Run single check cycle:

```bash
./xray-checker --subscription-url="https://your-sub-url" --run-once
```

Enable metrics authentication:

```bash
./xray-checker \
  --subscription-url="https://your-sub-url" \
  --metrics-protected=true \
  --metrics-username=user \
  --metrics-password=pass
```

Change ports:

```bash
./xray-checker \
  --subscription-url="https://your-sub-url" \
  --metrics-host=127.0.0.1 \
  --metrics-port=3000 \
  --xray-start-port=20000
```
