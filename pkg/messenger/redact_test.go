package messenger

import (
	"net/http"
	"testing"
)

func TestRedactHeaders_ScrubsSensitive(t *testing.T) {
	h := http.Header{
		"X-S21-Token":   []string{"alice:pw"},
		"X-Api-Key":     []string{"sk-secret"},
		"Authorization": []string{"Bearer xyz"},
		"Cookie":        []string{"session=abc"},
		"Content-Type":  []string{"application/json"},
	}
	got := RedactHeaders(h)
	for _, name := range []string{"X-S21-Token", "X-Api-Key", "Authorization", "Cookie"} {
		if got.Get(name) != "[REDACTED]" {
			t.Errorf("%s not redacted: %q", name, got.Get(name))
		}
	}
	if got.Get("Content-Type") == "[REDACTED]" {
		t.Errorf("Content-Type should not be redacted")
	}
}

func TestRedactHeaders_NoMutation(t *testing.T) {
	h := http.Header{"X-S21-Token": []string{"alice:pw"}}
	_ = RedactHeaders(h)
	if h.Get("X-S21-Token") != "alice:pw" {
		t.Errorf("original mutated: %q", h.Get("X-S21-Token"))
	}
}

func TestRedactHeaders_NilSafe(t *testing.T) {
	if RedactHeaders(nil) != nil {
		t.Error("nil input should return nil")
	}
}
