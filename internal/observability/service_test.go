package observability

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appconfig "github.com/hanchuanchuan/goinception-plus/internal/config"
)

func TestMetricsLifecycle(t *testing.T) {
	s := New(appconfig.Log{Level: "error", Format: "json"})
	s.SetReady(true)
	done := s.Begin()
	done(true, time.Millisecond)
	r := httptest.NewRecorder()
	s.metrics(r, httptest.NewRequest("GET", "/metrics", nil))
	body := r.Body.String()
	for _, want := range []string{"goinception_plus_ready 1", "goinception_plus_audit_total 1", "goinception_plus_audit_failed_total 1"} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics missing %q:\n%s", want, body)
		}
	}
}
