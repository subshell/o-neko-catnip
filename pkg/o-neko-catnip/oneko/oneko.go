package oneko

import (
	"context"
	"fmt"
	"net/http"
	"o-neko-catnip/pkg/o-neko-catnip/config"
	"o-neko-catnip/pkg/o-neko-catnip/logger"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

type ONekoApi struct {
	client          *resty.Client
	log             *zap.SugaredLogger
	cache           *ttlcache.Cache
	wakeupCounter   prometheus.Counter
	errorCounter    prometheus.Counter
	apiCallDuration prometheus.Histogram
}

func New(configuration *config.Config, ctx context.Context) *ONekoApi {
	client, err := buildClient(configuration)
	if err != nil {
		panic(err)
	}

	requestCache := ttlcache.NewCache()

	api := &ONekoApi{
		client: client,
		log:    logger.New("oneko"),
		cache:  requestCache,
		wakeupCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "oneko_url_trigger_wakeups_total",
			Help: "The number of wakeup API requests done.",
			ConstLabels: map[string]string{
				"success": "true",
			},
		}),
		errorCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "oneko_url_trigger_wakeups_total",
			Help: "The number of wakeup API requests done.",
			ConstLabels: map[string]string{
				"success": "false",
			},
		}),
		apiCallDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "oneko_url_trigger_api_call_duration_seconds",
			Help:    "Wakeup API call duration.",
			Buckets: prometheus.DefBuckets,
		}),
	}

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "oneko_url_trigger_cache_size",
		Help: "The number of cached API responses",
	}, func() float64 {
		return float64(len(requestCache.GetKeys()))
	})

	_ = requestCache.SetTTL(time.Duration(configuration.ONeko.Api.CacheRequestsInMinutes) * time.Minute)
	requestCache.SkipTTLExtensionOnHit(true)

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
	fromCache, err := o.cache.GetByLoader(cacheKey, o.loadIntoCache)
	if err != nil {
		return nil, nil, err
	} else if fromCache != nil {
		o.log.Infow("serving from cache", "deploymentUrl", deploymentUrl, "cacheKey", cacheKey)
		entry := fromCache.(*cacheEntry)
		return entry.Project, entry.Version, nil
	}
	return nil, nil, fmt.Errorf("failed to handle request")
}

func (o *ONekoApi) loadIntoCache(cacheKey string) (interface{}, time.Duration, error) {
	o.log.Infow("no cached entry found, calling o-neko api", "deploymentUrl", cacheKey)
	project, err := o.wakeupDeployment(cacheKey)
	if err != nil {
		o.log.Errorw("failed to deploy", "deploymentUrl", cacheKey, "error", err)
		return nil, time.Duration(0), err
	} else {
		version := getProjectVersionMatchingUrl(project.Versions, cacheKey)
		if version != nil {
			err = o.cache.Set(cacheKey, cacheEntry{
				Project: project,
				Version: version,
			})
			if err != nil {
				o.log.Warnw("failed to insert into cache", "deploymentUrl", cacheKey, "error", err)
			} else {
				o.log.Infow("added element to cache", "cacheKey", cacheKey)
			}
			o.log.Infow("started deployment", "project", project.Name, "projectVersion", version.Name, "versionDate", version.ImageUpdatedDate)
			entry := &cacheEntry{
				Project: project,
				Version: version,
			}
			return entry, time.Duration(0), nil
		} else {
			o.log.Infow("no version matching url found", "deploymentUrl", cacheKey)
			return nil, time.Duration(0), fmt.Errorf("no version matches the current URL")
		}
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
