package oneko

import (
	"fmt"
	"github.com/ReneKroon/ttlcache/v2"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/config"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/logger"
	"time"
)

type ONekoApi struct {
	client *resty.Client
	log    *zap.SugaredLogger
	cache  *ttlcache.Cache
}

func New(configuration *config.Config) *ONekoApi {
	requestCache := ttlcache.NewCache()
	api := &ONekoApi{
		client: buildClient(configuration),
		log:    logger.New("oneko"),
		cache:  requestCache,
	}

	requestCache.SetTTL(time.Duration(configuration.ONeko.Api.CacheRequestsInMinutes) * time.Minute)
	requestCache.SkipTTLExtensionOnHit(true)
	requestCache.SetLoaderFunction(func(key string) (data interface{}, ttl time.Duration, err error) {
		return api.loadIntoCache(key)
	})
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
	fromCache, err := o.cache.Get(cacheKey)
	if err != nil && err != ttlcache.ErrNotFound {
		return nil, nil, err
	} else if err != nil {
		return nil, nil, err
	} else if fromCache != nil {
		o.log.Infow("serving from cache", "deploymentUrl", deploymentUrl, "cacheKey", cacheKey)
		entry := fromCache.(*cacheEntry)
		return entry.Project, entry.Version, nil
	}
	return nil, nil, fmt.Errorf("failed to handle request")
}

func (o *ONekoApi) loadIntoCache(cacheKey string) (*cacheEntry, time.Duration, error) {
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
			o.log.Infow("no deployment matching version", "deploymentUrl", cacheKey)
			return nil, time.Duration(0), fmt.Errorf("no matching deployment found")
		}
	}
}

func (o *ONekoApi) wakeupDeployment(deploymentUrl string) (*Project, error) {
	response, err := o.client.R().
		SetBody(deploymentUrl).
		SetResult(&Project{}).
		Post("/api/project/deploy/url")
	if err != nil {
		return nil, err
	}
	return response.Result().(*Project), nil
}
