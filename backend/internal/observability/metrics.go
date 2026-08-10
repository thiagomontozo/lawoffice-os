package observability

import (
	"fmt"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type Metrics struct {
	startedAt          time.Time
	requestsTotal      atomic.Uint64
	requestsActive     atomic.Int64
	responseClasses    [6]atomic.Uint64
	durationNanosecond atomic.Uint64
}

func NewMetrics() *Metrics {
	return &Metrics{startedAt: time.Now().UTC()}
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		m.requestsTotal.Add(1)
		m.requestsActive.Add(1)
		defer m.requestsActive.Add(-1)
		wrapped := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(wrapped, r)
		statusClass := wrapped.Status() / 100
		if statusClass >= 1 && statusClass <= 5 {
			m.responseClasses[statusClass].Add(1)
		}
		m.durationNanosecond.Add(uint64(time.Since(started)))
	})
}

func (m *Metrics) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	writeMetric(w, "lawoffice_http_requests_total", "Total HTTP requests received.", "counter", m.requestsTotal.Load())
	writeMetric(w, "lawoffice_http_requests_active", "HTTP requests currently being served.", "gauge", m.requestsActive.Load())
	for class := 1; class <= 5; class++ {
		_, _ = fmt.Fprintf(w, "lawoffice_http_responses_total{class=\"%dxx\"} %d\n", class, m.responseClasses[class].Load())
	}
	_, _ = fmt.Fprintf(w, "# HELP lawoffice_http_request_duration_seconds_total Cumulative time spent serving HTTP requests.\n# TYPE lawoffice_http_request_duration_seconds_total counter\nlawoffice_http_request_duration_seconds_total %.6f\n", float64(m.durationNanosecond.Load())/float64(time.Second))
	_, _ = fmt.Fprintf(w, "# HELP lawoffice_process_uptime_seconds Process uptime.\n# TYPE lawoffice_process_uptime_seconds gauge\nlawoffice_process_uptime_seconds %.0f\n", time.Since(m.startedAt).Seconds())
	writeMetric(w, "lawoffice_go_goroutines", "Current number of Go goroutines.", "gauge", runtime.NumGoroutine())
	writeMetric(w, "lawoffice_go_heap_bytes", "Bytes of allocated heap objects.", "gauge", memory.HeapAlloc)
}

func writeMetric(w http.ResponseWriter, name, help, metricType string, value any) {
	_, _ = fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s %s\n%s %v\n", name, help, name, metricType, name, value)
}
