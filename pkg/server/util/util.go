package util

import (
	"net/http"
)

func MarshalRequest(r *http.Request) map[string]interface{} {
	return map[string]interface{}{
		"headers": r.Header,
		"path":    r.RequestURI,
		"method":  r.Method,
	}
}
