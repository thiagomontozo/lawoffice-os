package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsCollectRequestsAndExposePrometheusText(t *testing.T) {
	metrics := NewMetrics()
	handler := metrics.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/matters", nil))
	if response.Code != http.StatusCreated {
		t.Fatalf("unexpected application response: %d", response.Code)
	}
	scrape := httptest.NewRecorder()
	metrics.ServeHTTP(scrape, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := scrape.Body.String()
	for _, expected := range []string{
		"lawoffice_http_requests_total 1",
		"lawoffice_http_requests_active 0",
		"lawoffice_http_responses_total{class=\"2xx\"} 1",
		"lawoffice_process_uptime_seconds",
		"lawoffice_go_heap_bytes",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("metrics output does not contain %q:\n%s", expected, body)
		}
	}
}
