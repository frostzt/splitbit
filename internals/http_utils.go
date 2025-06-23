package internals

import (
	"strings"
)

func IsHTTPRequest(request string) bool {
	if len(request) < 5 { // Minimum "GET /"
		return false
	}

	return strings.HasPrefix(request, "GET ") ||
		strings.HasPrefix(request, "POST ") ||
		strings.HasPrefix(request, "PUT ") ||
		strings.HasPrefix(request, "DELETE ") ||
		strings.HasPrefix(request, "HEAD ") ||
		strings.HasPrefix(request, "OPTIONS ") ||
		strings.HasPrefix(request, "PATCH ")
}
