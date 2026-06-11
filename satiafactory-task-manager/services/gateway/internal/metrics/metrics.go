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
			Help: "Total HTTP requests processed by the gateway.",
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
	const service = "gateway"
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
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "/"
	}
	switch parts[0] {
	case "tasks":
		if len(parts) == 1 {
			return "/tasks"
		}
		return "/tasks/" + parts[1]
	case "recipes":
		if len(parts) == 1 {
			return "/recipes"
		}
		return "/recipes/" + parts[1]
	case "users":
		if len(parts) == 1 {
			return "/users"
		}
		return "/users/" + parts[1]
	case "icons":
		return "/icons"
	case "login", "register", "logout", "my-tasks":
		return "/" + parts[0]
	default:
		return "/" + parts[0]
	}
}
