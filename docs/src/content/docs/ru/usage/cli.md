---
title: CLI
description: Использование CLI в Xray Checker
---

### Базовое использование командной строки

CLI интерфейс предоставляет полный контроль над функциональностью Xray Checker через аргументы командной строки.

### Установка

Скачайте последнюю версию бинарного файла из релизов:

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

### Базовое использование

Минимально необходимая конфигурация:

```bash
./xray-checker --subscription-url="https://your-subscription-url/sub"
```

### Пример полной конфигурации

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

### Основные операции CLI

Проверка версии:

```bash
./xray-checker --version
```

Запуск одного цикла проверки:

```bash
./xray-checker --subscription-url="https://your-sub-url" --run-once
```

Включение аутентификации метрик:

```bash
./xray-checker \
  --subscription-url="https://your-sub-url" \
  --metrics-protected=true \
  --metrics-username=user \
  --metrics-password=pass
```

Изменение портов:

```bash
./xray-checker \
  --subscription-url="https://your-sub-url" \
  --metrics-host=127.0.0.1 \
  --metrics-port=3000 \
  --xray-start-port=20000
```