package messenger

import (
	"net/http"
	"strings"
)

// sensitiveHeaderNames lists HTTP headers ttbot or its dependencies
// emit/receive that carry secrets. RedactHeaders replaces their values
// with "[REDACTED]" before any log line might dump them.
//
// Today's call sites: the Telegram client (telegram.go) sets
// Authorization-style URL paths (token in the URL), and the
// identity-service client sets X-S21-Token + X-Api-Key. Neither logs
// headers today; this helper exists so they can't start.
var sensitiveHeaderNames = map[string]struct{}{
	"x-s21-token":   {},
	"x-api-key":     {},
	"authorization": {},
	"cookie":        {},
}

// RedactHeaders returns a shallow copy of h with values of sensitive
// headers replaced by "[REDACTED]". The original h is not mutated.
func RedactHeaders(h http.Header) http.Header {
	if h == nil {
		return nil
	}
	out := make(http.Header, len(h))
	for k, vs := range h {
		if isSensitiveHeader(k) {
			out[k] = []string{"[REDACTED]"}
			continue
		}
		out[k] = append([]string(nil), vs...)
	}
	return out
}

func isSensitiveHeader(name string) bool {
	_, ok := sensitiveHeaderNames[strings.ToLower(name)]
	return ok
}
