---
title: Check Methods
description: Check methods options and examples
---

## Check Methods

Xray Checker supports three methods for verifying proxy functionality:

### IP Check Method (Default)

```bash
--proxy-check-method=ip
```

This method:

1. Gets current IP without proxy
2. Connects through proxy
3. Gets IP through proxy
4. Compares IPs to verify proxy is working

Benefits:

- More reliable verification
- Confirms actual proxy functionality
- Detects transparent proxies

Configuration:

```bash
PROXY_CHECK_METHOD=ip
PROXY_IP_CHECK_URL=https://api.ipify.org?format=text
PROXY_TIMEOUT=30
```

### Status Check Method

```bash
--proxy-check-method=status
```

This method:

1. Connects through proxy
2. Requests specified URL
3. Verifies response status code

Benefits:

- Faster verification
- Lower bandwidth usage
- Works with restrictive firewalls

Configuration:

```bash
PROXY_CHECK_METHOD=status
PROXY_STATUS_CHECK_URL=http://cp.cloudflare.com/generate_204
PROXY_TIMEOUT=30
```

### Download Check Method

```bash
--proxy-check-method=download
```

This method:

1. Connects through proxy
2. Downloads a specified file
3. Verifies download completion or minimum size threshold

Benefits:

- Tests actual data transfer through proxy
- Verifies proxy performance with real file downloads
- Configurable success criteria (minimum download size)
- Single-threaded download in one TCP connection

Configuration:

```bash
PROXY_CHECK_METHOD=download
PROXY_DOWNLOAD_URL=https://proof.ovh.net/files/1Mb.dat
PROXY_DOWNLOAD_TIMEOUT=60
PROXY_DOWNLOAD_MIN_SIZE=51200
```

The check is considered successful if:

- The file is completely downloaded, OR
- At least the minimum specified bytes are downloaded

This method is ideal for testing proxy performance with actual file transfers and ensuring the proxy can handle sustained data connections.
