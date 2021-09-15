package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"time"
)

func PrometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func RegisterCommonMetrics(appVersion string) {
	startMonitoringUptime()
	registerAppInfoMetric(appVersion)
}

func startMonitoringUptime() {
	startTime := time.Now()
	promauto.NewCounterFunc(prometheus.CounterOpts{
		Name: "oneko_url_trigger_uptime_duration_seconds",
		Help: "The uptime of the application",
	}, func() float64 {
		return time.Now().Sub(startTime).Seconds()
	})
}

func registerAppInfoMetric(appVersion string) {
	promauto.NewGauge(prometheus.GaugeOpts{
		Name: "oneko_url_trigger_info",
		Help: "Information about this application",
		ConstLabels: map[string]string{
			"version": appVersion,
		},
	}).Set(1)
}
