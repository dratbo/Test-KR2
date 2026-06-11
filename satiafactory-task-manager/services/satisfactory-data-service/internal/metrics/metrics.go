package metrics

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests processed by data-service.",
		},
		[]string{"service", "method", "route", "status"},
	)
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "route"},
	)
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rw *statusRecorder) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func Handler() http.Handler {
	return promhttp.Handler()
}

func Middleware(next http.Handler) http.Handler {
	const service = "data-service"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		route := normalizeRoute(r.URL.Path)
		status := strconv.Itoa(rec.status)
		requestsTotal.WithLabelValues(service, r.Method, route, status).Inc()
		requestDuration.WithLabelValues(service, r.Method, route).Observe(time.Since(start).Seconds())
	})
}

func normalizeRoute(path string) string {
	switch {
	case strings.HasPrefix(path, "/api/items/"):
		return "/api/items/{className}"
	case strings.HasPrefix(path, "/api/recipes/by-product/"):
		return "/api/recipes/by-product/{className}"
	case strings.HasPrefix(path, "/api/recipes/has-product/"):
		return "/api/recipes/has-product/{className}"
	case strings.HasPrefix(path, "/api/recipes/"):
		return "/api/recipes/{className}"
	default:
		return path
	}
}
