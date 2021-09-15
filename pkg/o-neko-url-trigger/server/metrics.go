package server

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"time"
)

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func startMonitoringUptime() {
	startTime := time.Now()
	promauto.NewCounterFunc(prometheus.CounterOpts{
		Name:        "oneko_url_trigger_uptime_duration_seconds",
		Help:        "The uptime of the application",
	}, func() float64 {
		return time.Now().Sub(startTime).Seconds()
	})
}
