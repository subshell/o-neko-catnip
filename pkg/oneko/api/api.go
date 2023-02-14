package api

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"net/http"
	"o-neko-catnip/pkg/config"
	"o-neko-catnip/pkg/logger"
	"o-neko-catnip/pkg/oneko"
	"sync"
	"time"
)

type Api struct {
	client            *resty.Client
	log               *zap.SugaredLogger
	wakeupCounter     prometheus.Counter
	errorCounter      prometheus.Counter
	apiCallDuration   prometheus.Histogram
	onekoPingDuration prometheus.Histogram
	onekoConnected    prometheus.Gauge
	pingOnlyOnce      sync.Once
}

func New(configuration *config.Config) *Api {
	client, err := buildClient(configuration)
	if err != nil {
		panic(err)
	}

	api := &Api{
		client: client,
		log:    logger.New("onekoApi"),
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
		onekoPingDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "oneko_catnip_api_ping_duration_seconds",
			Help:    "Ping duration to the O-Neko base application.",
			Buckets: prometheus.DefBuckets,
		}),
		onekoConnected: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "oneko_catnip_api_connected",
			Help: "1 if the API is connected, 0 if not",
		}),
		pingOnlyOnce: sync.Once{},
	}
	api.onekoConnected.Set(0)
	return api
}

func buildClient(conf *config.Config) (*resty.Client, error) {
	if len(conf.ONeko.Api.BaseUrl) == 0 {
		return nil, fmt.Errorf("API url must not be empty")
	}
	if len(conf.ONeko.Api.Auth.Username) == 0 || len(conf.ONeko.Api.Auth.Password) == 0 {
		return nil, fmt.Errorf("API username and password must be set")
	}
	return resty.New().
		SetBaseURL(conf.ONeko.Api.BaseUrl).
		SetDisableWarn(true).
		SetLogger(logger.New("rest-client")).
		SetBasicAuth(conf.ONeko.Api.Auth.Username, conf.ONeko.Api.Auth.Password), nil
}

func (api *Api) StartConnectionMonitor(ctx context.Context) {
	api.pingOnlyOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(5 * time.Second)

			handlePing := func() {
				err := api.ping(ctx)
				if err != nil {
					api.log.Errorw("error reaching o-neko", "error", err)
					api.onekoConnected.Set(0)
				} else {
					api.onekoConnected.Set(1)
				}
			}

			handlePing()

			for {
				select {
				case <-ticker.C:
					handlePing()
				case <-ctx.Done():
					ticker.Stop()
					return
				}
			}
		}()
	})
}

func (api *Api) ping(ctx context.Context) error {
	timer := prometheus.NewTimer(api.onekoPingDuration)
	defer timer.ObserveDuration()
	response, err := api.client.R().
		SetContext(ctx).
		Get("/api/session")
	if err != nil {
		return err
	} else if response.IsError() {
		return fmt.Errorf("encountered an error calling O-Neko API: %s (%d)", response.Status(), response.StatusCode())
	}
	return nil
}

func (api *Api) GetProjectById(id string, ctx context.Context) (*oneko.Project, error) {
	timer := prometheus.NewTimer(api.apiCallDuration)
	defer timer.ObserveDuration()

	response, err := api.client.R().
		SetContext(ctx).
		SetResult(&oneko.Project{}).
		Get("/api/project/" + id)

	if err != nil {
		api.errorCounter.Inc()
		return nil, err
	} else if response.IsError() {
		api.errorCounter.Inc()
		if response.StatusCode() == http.StatusNotFound {
			return nil, fmt.Errorf("no project found with id %s", id)
		} else {
			return nil, fmt.Errorf("encountered an error calling O-Neko API (status %s (%d)", response.Status(), response.StatusCode())
		}
	}

	project, ok := response.Result().(*oneko.Project)
	if !ok {
		return nil, fmt.Errorf("encountered an error parsing the response from the O-Neko API")
	}

	return project, nil
}

func (api *Api) GetAllProjects(ctx context.Context) ([]*oneko.Project, error) {
	timer := prometheus.NewTimer(api.apiCallDuration)
	defer timer.ObserveDuration()

	response, err := api.client.R().
		SetContext(ctx).
		SetResult(&[]*oneko.Project{}).
		Get("/api/project")

	if err != nil {
		api.errorCounter.Inc()
		return nil, err
	} else if response.IsError() {
		api.errorCounter.Inc()
		return nil, fmt.Errorf("encountered an error calling O-Neko API (status %s (%d)", response.Status(), response.StatusCode())
	}

	projects, ok := response.Result().(*[]*oneko.Project)
	if !ok {
		return nil, fmt.Errorf("encountered an error parsing the response from the O-Neko API")
	}

	return *projects, nil
}

func (api *Api) Deploy(projectId, versionId string, ctx context.Context) error {
	timer := prometheus.NewTimer(api.apiCallDuration)
	defer timer.ObserveDuration()
	response, err := api.client.R().
		SetContext(ctx).
		Post(fmt.Sprintf("/api/project/%s/version/%s/deploy", projectId, versionId))

	if err != nil {
		api.errorCounter.Inc()
		return err
	} else if response.IsError() {
		api.errorCounter.Inc()
		if response.StatusCode() == http.StatusNotFound {
			return fmt.Errorf(response.String())
		} else {
			return fmt.Errorf("encountered an error calling O-Neko API: %s (%d)", response.Status(), response.StatusCode())
		}
	}
	api.wakeupCounter.Inc()
	return nil
}
