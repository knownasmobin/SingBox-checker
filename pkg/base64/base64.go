package base64

import (
	"encoding/base64"
	"strings"
)

// AutoDecode automatically detects whether the input uses standard or URL-safe Base64,
// and whether it includes padding or not. It then selects the appropriate decoder
// from the encoding/base64 package and returns the decoded bytes.
func AutoDecode(s string) ([]byte, error) {
	// Detect if the input is URL-safe (contains '-' or '_')
	isURLSafe := strings.ContainsAny(s, "-_")

	// Detect if the input is padded (ends with '=')
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
