package redirect

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the redirect service
type Metrics struct {
	// Standard HTTP metrics
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec

	// Custom redirect metrics
	RedirectsTotal        *prometheus.CounterVec
	RedirectsNotFound     prometheus.Counter
	WikiRedirectsFollowed prometheus.Counter
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics() *Metrics {
	return &Metrics{
		// Total number of HTTP requests
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mediawiki_redirect_requests_total",
				Help: "Total number of HTTP requests received",
			},
			[]string{"method", "path", "status"},
		),

		// HTTP request duration
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "mediawiki_redirect_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),

		// Total number of successful redirects
		RedirectsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "mediawiki_redirects_total",
				Help: "Total number of successful redirects performed",
			},
			[]string{"type"}, // type can be "direct", "wiki_redirect", or "root"
		),

		// Total number of redirects not found (404s)
		RedirectsNotFound: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "mediawiki_redirects_not_found_total",
				Help: "Total number of redirect requests that resulted in 404 Not Found",
			},
		),

		// Total number of wiki redirects followed
		WikiRedirectsFollowed: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "mediawiki_wiki_redirects_followed_total",
				Help: "Total number of MediaWiki internal redirects followed",
			},
		),
	}
}
