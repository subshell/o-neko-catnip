package oneko

import (
	"context"
	"fmt"
	"github.com/jellydator/ttlcache/v3"
	"net/http"
	"o-neko-catnip/pkg/o-neko-catnip/config"
	"o-neko-catnip/pkg/o-neko-catnip/logger"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

type ONekoApi struct {
	client          *resty.Client
	log             *zap.SugaredLogger
	cache           *ttlcache.Cache[string, *cacheEntry]
	wakeupCounter   prometheus.Counter
	errorCounter    prometheus.Counter
	apiCallDuration prometheus.Histogram
	loaderFunc      ttlcache.LoaderFunc[string, *cacheEntry]
}

func New(configuration *config.Config, ctx context.Context) *ONekoApi {
	client, err := buildClient(configuration)
	if err != nil {
		panic(err)
	}

	var api *ONekoApi

	requestCache := ttlcache.New[string, *cacheEntry](
		ttlcache.WithTTL[string, *cacheEntry](time.Duration(configuration.ONeko.Api.CacheRequestsInMinutes)*time.Minute),
		ttlcache.WithDisableTouchOnHit[string, *cacheEntry](),
	)

	api = &ONekoApi{
		client: client,
		log:    logger.New("oneko"),
		cache:  requestCache,
		wakeupCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "oneko_catnip_wakeups_total",
			Help: "The number of wakeup API requests done.",
			ConstLabels: map[string]string{
				"success": "true",
			},
		}),
		errorCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "oneko_catnip_wakeups_total",
			Help: "The number of wakeup API requests done.",
			ConstLabels: map[string]string{
				"success": "false",
			},
		}),
		apiCallDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "oneko_catnip_api_call_duration_seconds",
			Help:    "Wakeup API call duration.",
			Buckets: prometheus.DefBuckets,
		}),
		loaderFunc: func(c *ttlcache.Cache[string, *cacheEntry], cacheKey string) *ttlcache.Item[string, *cacheEntry] {
			api.log.Infow("no cached entry found, calling o-neko api", "deploymentUrl", cacheKey)
			project, err := api.wakeupDeployment(cacheKey)
			if err != nil {
				api.log.Errorw("failed to deploy", "deploymentUrl", cacheKey, "error", err)
				return nil
			} else {
				version := getProjectVersionMatchingUrl(project.Versions, cacheKey)
				if version != nil {
					entry := c.Set(cacheKey, &cacheEntry{
						Project: project,
						Version: version,
					}, ttlcache.DefaultTTL)
					api.log.Infow("added element to cache", "cacheKey", cacheKey)
					api.log.Infow("started deployment", "project", project.Name, "projectVersion", version.Name, "versionDate", version.ImageUpdatedDate)
					return entry
				} else {
					api.log.Infow("no version matching url found", "deploymentUrl", cacheKey)
					return nil
				}
			}
		},
	}

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "oneko_catnip_cache_size",
		Help: "The number of cached API responses",
	}, func() float64 {
		return float64(len(requestCache.Keys()))
	})

	startPingMonitor(api, ctx)

	return api
}

func (o *ONekoApi) HandleRequest(host, uri string) (*Project, *ProjectVersion, error) {
	o.log.Infow("incoming request", "host", host, "uri", uri)
	deploymentUrl := fmt.Sprintf("%s%s", host, uri)

	cacheKey, err := getCacheKey(deploymentUrl)
	if err != nil {
		o.log.Errorw("failed to generate cache key", "deploymentUrl", deploymentUrl, "error", err)
		return nil, nil, err
	}
	fromCache := o.cache.Get(cacheKey,
		ttlcache.WithLoader[string, *cacheEntry](o.loaderFunc),
	)
	if fromCache == nil {
		return nil, nil, fmt.Errorf("no version matching this url found")
	} else {
		o.log.Infow("serving from cache", "deploymentUrl", deploymentUrl, "cacheKey", cacheKey)
		entry := fromCache.Value()
		return entry.Project, entry.Version, nil
	}
}

func (o *ONekoApi) wakeupDeployment(deploymentUrl string) (*Project, error) {
	timer := prometheus.NewTimer(o.apiCallDuration)
	defer timer.ObserveDuration()
	response, err := o.client.R().
		SetBody(deploymentUrl).
		SetResult(&Project{}).
		Post("/api/project/deploy/url")
	if err != nil {
		o.errorCounter.Inc()
		return nil, err
	} else if response.IsError() {
		o.errorCounter.Inc()
		if response.StatusCode() == http.StatusNotFound {
			return nil, fmt.Errorf("no version matching this url found")
		} else {
			return nil, fmt.Errorf("encountered an error calling O-Neko API: %s (%d)", response.Status(), response.StatusCode())
		}
	}
	o.wakeupCounter.Inc()
	return response.Result().(*Project), nil
}

func (o *ONekoApi) ping() error {
	response, err := o.client.R().
		Get("/api/session")
	if err != nil {
		return err
	} else if response.IsError() {
		return fmt.Errorf("encountered an error calling O-Neko API: %s (%d)", response.Status(), response.StatusCode())
	}
	return nil
}
