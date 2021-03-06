package oneko

import (
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"o-neko-catnip/pkg/o-neko-catnip/config"
	"regexp"
	"strings"
	"time"
)

type cacheEntry struct {
	Project *Project
	Version *ProjectVersion
}

func buildClient(conf *config.Config) (*resty.Client, error) {
	if len(conf.ONeko.Api.BaseUrl) == 0 {
		return nil, fmt.Errorf("API url must not be empty")
	}
	if len(conf.ONeko.Api.Auth.Username) == 0 || len(conf.ONeko.Api.Auth.Password) == 0 {
		return nil, fmt.Errorf("API username and password must be set")
	}
	client := resty.New()
	client.SetBaseURL(conf.ONeko.Api.BaseUrl)
	client.SetBasicAuth(conf.ONeko.Api.Auth.Username, conf.ONeko.Api.Auth.Password)
	return client, nil
}

func startPingMonitor(api *ONekoApi, ctx context.Context) {
	onekoPingDuration := promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "oneko_catnip_api_ping_duration_seconds",
		Help:    "Ping duration to the O-Neko base application.",
		Buckets: prometheus.DefBuckets,
	})
	onekoConnected := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "oneko_catnip_api_connected",
		Help: "1 if the API is connected, 0 if not",
	})
	onekoConnected.Set(0)
	go func() {
		pingOneko := func() {
			timer := prometheus.NewTimer(onekoPingDuration)
			defer timer.ObserveDuration()
			err := api.ping()
			if err != nil {
				api.log.Errorw("error reaching o-neko", "error", err)
				onekoConnected.Set(0)
			} else {
				onekoConnected.Set(1)
			}
		}
		pingOneko()

		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ticker.C:
				pingOneko()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func getCacheKey(deploymentUrl string) (string, error) {
	protocolRegex, err := regexp.Compile("^https?://")

	if err != nil {
		return "", err
	}

	pathRegex, err := regexp.Compile("/.*$")
	if err != nil {
		return "", err
	}

	withoutProtocol := protocolRegex.ReplaceAllString(deploymentUrl, "")
	return pathRegex.ReplaceAllString(withoutProtocol, ""), nil
}

func getProjectVersionMatchingUrl(versions []ProjectVersion, url string) *ProjectVersion {
	r, err := regexp.Compile("^https?://")
	if err != nil {
		return nil
	}
	urlWithoutProtocol := r.ReplaceAllString(url, "")
	for _, version := range versions {
		for _, url := range version.Urls {
			versionUrlWithoutProtocol := r.ReplaceAllString(url, "")
			if strings.HasPrefix(urlWithoutProtocol, versionUrlWithoutProtocol) {
				return &version
			}
		}
	}
	return nil
}
