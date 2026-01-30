package singbox

import (
	"xray-checker/models"
)

func IsConfigsEqual(a, b []*models.ProxyConfig) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].StableID != b[i].StableID {
			return false
		}
	}

	return true
}

func PrepareProxyConfigs(configs []*models.ProxyConfig) {
	for i, cfg := range configs {
		cfg.Index = i
		cfg.StableID = cfg.GenerateStableID()
	}
}
