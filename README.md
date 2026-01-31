# Proxy Checker

<div align="center">

[![GitHub Release](https://img.shields.io/github/v/release/kutovoys/xray-checker?color=blue)](https://github.com/kutovoys/xray-checker/releases/latest)
[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/kutovoys/xray-checker/build-publish.yml)](https://github.com/kutovoys/xray-checker/actions/workflows/build-publish.yml)
[![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/kutovoys/xray-checker/total?logo=github&color=blue)](https://github.com/kutovoys/xray-checker/releases/latest)
[![Docker Pulls](https://img.shields.io/docker/pulls/kutovoys/xray-checker?logo=docker&label=pulls)](https://hub.docker.com/r/kutovoys/xray-checker/)
[![GitHub License](https://img.shields.io/github/license/kutovoys/xray-checker?color=greeen)](https://github.com/kutovoys/xray-checker/blob/main/LICENSE)
[![ru](https://img.shields.io/badge/lang-ru-blue)](https://github.com/kutovoys/xray-checker/blob/main/README_RU.md)
[![en](https://img.shields.io/badge/lang-en-red)](https://github.com/kutovoys/xray-checker/blob/main/README.md)

</div>
<div align="center">

[![Documentation](https://img.shields.io/badge/Docs-xray--checker.kutovoy.dev-blue)](https://xray-checker.kutovoy.dev/)
[![DockerHub](https://img.shields.io/badge/DockerHub-kutovoys%2Fxray--checker-blue)](https://hub.docker.com/r/kutovoys/xray-checker/)
[![Live Demo](https://img.shields.io/badge/Demo-live-green)](https://demo-xray-checker.kutovoy.dev/)
[![Telegram Chat](https://img.shields.io/badge/Telegram-Chat-blue?logo=telegram&)](https://t.me/+uZCGx_FRY0tiOGIy)

</div>

Proxy Checker is a tool for monitoring proxy server availability with support for VLESS, VMess, Trojan, Shadowsocks, and WireGuard protocols. It supports both **Xray Core** and **sing-box** backends, automatically tests connections and provides metrics for Prometheus, as well as API endpoints for integration with monitoring systems.

<div align="center">
  <img src=".github/screen/xray-checker.webp" alt="Dashboard Screenshot">
</div>

> [!TIP]
> **Try the Live Demo:** See Xray Checker in action at [demo-xray-checker.kutovoy.dev](https://demo-xray-checker.kutovoy.dev/)

## 🚀 Key Features

- 🔍 Monitoring of proxy servers (VLESS, VMess, Trojan, Shadowsocks, WireGuard)
- ⚡ Dual backend support: **Xray Core** and **sing-box**
- 🔄 Automatic configuration updates from subscription (multiple subscriptions supported)
- 📊 Prometheus metrics export with Pushgateway support
- 🌐 REST API with OpenAPI/Swagger documentation
- 🌓 Web interface with dark/light theme
- 🎨 Full web customization (custom logo, styles, or entire template)
- 📄 Public status page for VPN services (no authentication required)
- 📥 Endpoints for monitoring system integration (Uptime Kuma, etc.)
- 🔒 Basic Auth protection for metrics and web interface
- 🐳 Docker and Docker Compose support
- 📝 Flexible configuration loading:
  - URL subscriptions (base64, JSON)
  - Share links (vless://, vmess://, trojan://, ss://)
  - WireGuard config files (.conf)
  - JSON configuration files
  - Folders with configurations

Full list of features available in the [documentation](https://xray-checker.kutovoy.dev/intro/features).

## 🚀 Quick Start

### Docker

```bash
# Using Xray backend (default)
docker run -d \
  -e SUBSCRIPTION_URL=https://your-subscription-url/sub \
  -p 2112:2112 \
  kutovoys/xray-checker

# Using sing-box backend
docker run -d \
  -e BACKEND=singbox \
  -e SUBSCRIPTION_URL=https://your-subscription-url/sub \
  -p 2112:2112 \
  kutovoys/xray-checker

# With WireGuard configs
docker run -d \
  -e WIREGUARD_CONFIG=/app/wireguard \
  -v ./wireguard:/app/wireguard:ro \
  -p 2112:2112 \
  kutovoys/xray-checker
```

### Docker Compose

```yaml
services:
  proxy-checker:
    image: kutovoys/xray-checker
    environment:
      - BACKEND=xray                    # or singbox
      - SUBSCRIPTION_URL=https://your-subscription-url/sub
      - WIREGUARD_CONFIG=/app/wireguard # optional
    volumes:
      - ./wireguard:/app/wireguard:ro   # optional: WireGuard configs
    ports:
      - "2112:2112"
```

### Configuration Options

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `BACKEND` | Proxy backend: `xray` or `singbox` | `xray` |
| `SUBSCRIPTION_URL` | Subscription URL(s), comma-separated | - |
| `WIREGUARD_CONFIG` | WireGuard config path(s), comma-separated | - |
| `PROXY_CHECK_INTERVAL` | Check interval in seconds | `300` |
| `PROXY_CHECK_METHOD` | Check method: `ip`, `status`, `download` | `ip` |
| `METRICS_PROTECTED` | Enable basic auth | `false` |

See [.env.example](.env.example) for all configuration options.

Detailed installation and configuration documentation is available at [xray-checker.kutovoy.dev](https://xray-checker.kutovoy.dev/intro/quick-start)

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

For more details on how to contribute, read the [contributor's guide](https://xray-checker.kutovoy.dev/contributing/development-guide).

<p align="center">
Thanks to the all contributors who have helped improve Xray Checker:
</p>
<p align="center">
<a href="https://github.com/kutovoys/xray-checker/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=kutovoys/xray-checker" />
</a>
</p>
<p align="center">
  Made with <a rel="noopener noreferrer" target="_blank" href="https://contrib.rocks">contrib.rocks</a>
</p>

## VPN Recommendation

For secure and reliable internet access, we recommend [BlancVPN](https://getblancvpn.com/pricing?promo=klugscl&ref=xc-readme). Use promo code `KLUGSCL` for 15% off your subscription.
