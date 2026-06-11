package metrics

import (
	"net/http"
	"os"
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
			Help: "Total HTTP requests processed by task-service.",
		},
		[]string{"service", "instance", "method", "route", "status"},
	)
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "instance", "method", "route"},
	)
	cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "task_cache_hits_total",
			Help: "Task list cache hits in Redis.",
		},
		[]string{"instance", "scope"},
	)
	cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "task_cache_misses_total",
			Help: "Task list cache misses in Redis.",
		},
		[]string{"instance", "scope"},
	)
	instanceUp = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "task_service_up",
			Help: "Task-service instance is running (1 = up).",
		},
		[]string{"instance"},
	)
	eventsPublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rabbitmq_messages_published_total",
			Help: "Task events published to RabbitMQ.",
		},
		[]string{"event_type"},
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

func MarkInstanceUp() {
	instanceUp.WithLabelValues(instanceID()).Set(1)
}

func RecordCacheHit(scope string) {
	cacheHits.WithLabelValues(instanceID(), scope).Inc()
}

func RecordCacheMiss(scope string) {
	cacheMisses.WithLabelValues(instanceID(), scope).Inc()
}

func RecordEventPublished(eventType string) {
	eventsPublished.WithLabelValues(eventType).Inc()
}

func Middleware(next http.Handler) http.Handler {
	const service = "task-service"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		route := normalizeRoute(r.URL.Path)
		inst := instanceID()
		status := strconv.Itoa(rec.status)
		requestsTotal.WithLabelValues(service, inst, r.Method, route, status).Inc()
		requestDuration.WithLabelValues(service, inst, r.Method, route).Observe(time.Since(start).Seconds())
	})
}

func instanceID() string {
	if id := os.Getenv("INSTANCE_ID"); id != "" {
		return id
	}
	return "unknown"
}

func normalizeRoute(path string) string {
	path = strings.TrimPrefix(path, "/")
	if path == "tasks" {
		return "/tasks"
	}
	if strings.HasPrefix(path, "tasks/") {
		rest := strings.TrimPrefix(path, "tasks/")
		if strings.HasSuffix(rest, "/take") {
			return "/tasks/{id}/take"
		}
		return "/tasks/{id}"
	}
	return "/" + path
}
