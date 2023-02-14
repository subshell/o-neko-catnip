package server

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"net/http"
	"o-neko-catnip/pkg/oneko/service"
	"o-neko-catnip/pkg/utils"
	"time"
)

type catnipMux struct {
	defaultHandler http.Handler
	otherHandler   http.Handler
	svc            *service.Service
	domains        *utils.Memoized[*utils.Set[string]]
	domainCount    prometheus.Gauge
}

func newMux(defaultHandler, otherHandler http.Handler, svc *service.Service) catnipMux {
	domainCount := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "oneko_catnip_oneko_projectversion_domains",
		Help: "The number of unique domains across all O-Neko projects and versions",
	})
	memoize := utils.Memoize(15*time.Second, func() (*utils.Set[string], error) {
		domains := svc.GetAllProjectDomains()
		domainCount.Set(float64(domains.Size()))
		return domains, nil
	})
	return catnipMux{
		defaultHandler: defaultHandler,
		otherHandler:   otherHandler,
		svc:            svc,
		domains:        memoize,
		domainCount:    domainCount,
	}
}

func (m catnipMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	domains, err := m.domains.Get()

	if err != nil || !domains.Contains(r.Host) {
		m.defaultHandler.ServeHTTP(w, r)
	} else {
		m.otherHandler.ServeHTTP(w, r)
	}
}
