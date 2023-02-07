package deployment

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/jellydator/ttlcache/v3"
	"go.uber.org/zap"
	"o-neko-catnip/pkg/logger"
	"time"
)

type DeploymentMonitor struct {
	client      *resty.Client
	statusCache *ttlcache.Cache[string, *StatusResponse]
	log         *zap.SugaredLogger
}

func New() *DeploymentMonitor {
	client := resty.New()
	cache := ttlcache.New[string, *StatusResponse](
		ttlcache.WithTTL[string, *StatusResponse](5*time.Second),
		ttlcache.WithDisableTouchOnHit[string, *StatusResponse](),
		ttlcache.WithLoader[string, *StatusResponse](ttlcache.LoaderFunc[string, *StatusResponse](func(c *ttlcache.Cache[string, *StatusResponse], deploymentUrl string) *ttlcache.Item[string, *StatusResponse] {
			status := calculateDeploymentStatus(client, deploymentUrl)
			item := c.Set(deploymentUrl, status, ttlcache.DefaultTTL)
			return item
		})),
	)
	return &DeploymentMonitor{
		client:      client,
		statusCache: cache,
		log:         logger.New("deployment-monitor"),
	}
}

func (d *DeploymentMonitor) DeploymentStatus(url string) (*StatusResponse, error) {
	item := d.statusCache.Get(url)
	if item != nil && !item.IsExpired() {
		return item.Value(), nil
	}
	return nil, fmt.Errorf("could not find item in cache and loader did not load item for deployment url %s", url)
}

func calculateDeploymentStatus(client *resty.Client, url string) *StatusResponse {
	response, err := client.R().Head(url)

	if err != nil {
		return &StatusResponse{
			DeploymentReady: false,
			RedirectUrl:     url,
			IsError:         true,
			ErrorMessage:    err.Error(),
		}
	} else if response.StatusCode() >= 500 {
		return &StatusResponse{
			DeploymentReady: false,
			RedirectUrl:     url,
			IsError:         false, // >= 500 is the default when a deployment is starting
			ErrorMessage:    "",
		}
	}

	// All status codes < 500 are considered a ready deployment as long as
	// we don't read the catnip header.
	// This is equal to the behavior of the first catnip version.

	if len(response.Header().Get("oneko-catnip")) > 0 {
		// this is us - the deployment has not happened yet
		return &StatusResponse{
			DeploymentReady: false,
			RedirectUrl:     url,
			IsError:         false,
			ErrorMessage:    "",
		}
	}

	return &StatusResponse{
		DeploymentReady: true,
		RedirectUrl:     url,
		IsError:         false,
		ErrorMessage:    "",
	}
}
