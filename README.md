# SingBox Checker

[![GitHub Release](https://img.shields.io/github/v/release/knownasmobin/singbox-checker?style=flat&color=blue)](https://github.com/knownasmobin/singbox-checker/releases/latest)
[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/knownasmobin/singbox-checker/build-publish.yml)](https://github.com/knownasmobin/singbox-checker/actions/workflows/build-publish.yml)
[![DockerHub](https://img.shields.io/badge/DockerHub-knownasmobin%2Fsingbox--checker-blue)](https://hub.docker.com/r/knownasmobin/singbox-checker/)
[![GitHub License](https://img.shields.io/github/license/knownasmobin/singbox-checker?color=greeen)](https://github.com/knownasmobin/singbox-checker/blob/main/LICENSE)

SingBox Checker is a tool for monitoring proxy server availability with support for VLESS, VMess, Trojan, and Shadowsocks protocols. It automatically tests connections through SingBox and provides metrics for Prometheus, as well as API endpoints for integration with monitoring systems.

<div align="center">
  <img src=".github/screen/singbox-checker.png" alt="SingBox Checker Dashboard">
</div>

## 🚀 Key Features

- 🔍 Monitoring of SingBox proxy servers (VLESS, VMess, Trojan, Shadowsocks)
- 🔄 Automatic configuration updates from subscription
- 📊 Prometheus metrics export
- 🌓 Web interface with dark/light theme
- 📥 Endpoints for monitoring system integration
- 🔒 Basic Auth protection for metrics and web interface
- 🐳 Docker and Docker Compose support with multi-architecture support (amd64, arm64)
- 📝 Flexible configuration loading:
  - URL-subscription
  - Base64-strings
  - JSON-files
  - Folders with configurations

## 📦 Installation

### Prerequisites

- Docker (for containerized deployment)
- Go 1.24+ (for building from source)

## 🚀 Quick Start

### Docker

```bash
docker run -d \
  -e SUBSCRIPTION_URL=https://your-subscription-url/sub \
  -p 2112:2112 \
  -v /path/to/config:/etc/singbox \
  knownasmobin/singbox-checker:latest
```

### Docker Compose

```yaml
version: '3.8'

services:
  singbox-checker:
    image: knownasmobin/singbox-checker:latest
    container_name: singbox-checker
    restart: unless-stopped
    ports:
      - "2112:2112"
    volumes:
      - ./config:/etc/singbox
    environment:
      - SUBSCRIPTION_URL=https://your-subscription-url/sub
      - SINGBOX_CONFIG_DIR=/etc/singbox
      - SINGBOX_START_PORT=10000
      - PROXY_CHECK_INTERVAL=300
      - PROXY_TIMEOUT=10
      - PROXY_IP_CHECK_URL=https://api.ipify.org
      - PROXY_CHECK_METHOD=ip
      - METRICS_HOST=0.0.0.0
      - METRICS_PORT=2112
      # Optional: Basic Auth for web interface
      - WEB_USERNAME=admin
      - WEB_PASSWORD=changeme
      # Optional: Push metrics to Prometheus Pushgateway
      - METRICS_PUSH_URL=http://prometheus:9091
      - METRICS_INSTANCE=singbox-checker-1"""

```

## 📚 Documentation

For detailed documentation, please refer to the [GitHub repository](https://github.com/knownasmobin/singbox-checker).

## 📈 Project Statistics

<a href="https://star-history.com/#kutovoys/xray-checker&Date">
 <picture>
   <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=kutovoys/xray-checker&type=Date&theme=dark" />
   <source media="(prefers-color-scheme: light)" srcset="https://api.star-history.com/svg?repos=kutovoys/xray-checker&type=Date" />
   <img alt="Star History Chart" src="https://api.star-history.com/svg?repos=kutovoys/xray-checker&type=Date" />
 </picture>
</a>

## 🤝 Contributing

We welcome any contributions to Xray Checker! If you want to help:

1. Fork the repository
2. Create a branch for your changes
3. Make and test your changes
4. Create a Pull Request

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
