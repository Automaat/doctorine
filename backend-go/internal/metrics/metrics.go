package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "doctorine_http_requests_total",
		Help: "Total HTTP requests.",
	}, []string{"method", "route", "status"})

	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "doctorine_http_request_duration_seconds",
		Help:    "HTTP request duration.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status"})
)

func ObserveRequest(method string, route string, status int, duration time.Duration) {
	if route == "" {
		route = "unmatched"
	}
	statusText := strconv.Itoa(status)
	requestsTotal.WithLabelValues(method, route, statusText).Inc()
	requestDuration.WithLabelValues(method, route, statusText).Observe(duration.Seconds())
}

func Handler() http.Handler {
	return promhttp.Handler()
}
