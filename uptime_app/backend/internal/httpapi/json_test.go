package httpapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStrictJSONInput(t *testing.T) {
	for _, body := range []string{`{"name":"ok","extra":true}`, `{"name":"ok"} {}`, `{"name":"` + strings.Repeat("a", 100) + `"}`, `{"name":12}`} {
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		w := httptest.NewRecorder()
		var target struct {
			Name string `json:"name"`
		}
		if jsonInput(w, req, 32, &target, "invalid") || w.Code != 400 {
			t.Fatal("accepted input", body, w.Code)
		}
	}
}
