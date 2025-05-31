package utils

import (
	"encoding/base64"
	"strings"
)

func AutoDecode(s string) ([]byte, error) {
	isURLSafe := strings.ContainsAny(s, "-_")

	isPadded := strings.HasSuffix(s, "=")

	var enc *base64.Encoding
	switch {
	case isURLSafe && isPadded:
		enc = base64.URLEncoding
	case isURLSafe && !isPadded:
		enc = base64.RawURLEncoding
	case !isURLSafe && isPadded:
		enc = base64.StdEncoding
	case !isURLSafe && !isPadded:
		enc = base64.RawStdEncoding
	}

	return enc.DecodeString(s)
}
