package service

import (
	"context"
	"fmt"
	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
	"o-neko-catnip/pkg/config"
	"o-neko-catnip/pkg/logger"
	"o-neko-catnip/pkg/oneko"
	"o-neko-catnip/pkg/oneko/api"
	"o-neko-catnip/pkg/utils"
	"regexp"
	"strings"
)

type projectAndVersionIds struct {
	project        string
	projectVersion string
}

type Service struct {
	log                            *zap.SugaredLogger
	projectIdToProjectCache        *ttlcache.Cache[string, *oneko.Project]
	urlToProjectAndVersionIdsCache *ttlcache.Cache[string, projectAndVersionIds]
	api                            *api.Api
}

func New(configuration *config.Config, ctx context.Context) *Service {

	log := logger.New("onekoSvc")
	onekoApi := api.New(configuration)

	projectIdToProjectCache := ttlcache.New[string, *oneko.Project](
		ttlcache.WithTTL[string, *oneko.Project](configuration.ONeko.Api.ApiCallCacheDuration),
		ttlcache.WithDisableTouchOnHit[string, *oneko.Project](),
		ttlcache.WithLoader[string, *oneko.Project](ttlcache.LoaderFunc[string, *oneko.Project](func(c *ttlcache.Cache[string, *oneko.Project], projectId string) *ttlcache.Item[string, *oneko.Project] {
			log.Infow("no cached entry found, calling o-neko api", "projectUuid", projectId)
			project, err := onekoApi.GetProjectById(projectId, ctx)
			if err != nil {
				log.Errorw("O-Neko API returned an error", "error", err)
				return nil
			}
			entry := c.Set(projectId, project, ttlcache.DefaultTTL)
			return entry
		})),
	)

	urlToProjectAndVersionIdsCache := ttlcache.New[string, projectAndVersionIds](
		ttlcache.WithTTL[string, projectAndVersionIds](configuration.ONeko.Api.ApiCallCacheDuration),
		ttlcache.WithDisableTouchOnHit[string, projectAndVersionIds](),
		ttlcache.WithLoader[string, projectAndVersionIds](ttlcache.LoaderFunc[string, projectAndVersionIds](func(c *ttlcache.Cache[string, projectAndVersionIds], deploymentUrl string) *ttlcache.Item[string, projectAndVersionIds] {
			deploymentUrl, err := getDeploymentUrlWithoutProtocolAndPath(deploymentUrl)
			if err != nil {
				return nil
			}

			entry, err := populateUrlToIdCacheAndReturnEntryForUrl(c, log, onekoApi, deploymentUrl)
			if err != nil {
				return nil
			}
			return entry
		})),
	)

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "oneko_catnip_cache_size",
		Help: "The number of cached projects",
	}, func() float64 {
		return float64(len(projectIdToProjectCache.Keys()))
	})

	onekoApi.StartConnectionMonitor(ctx)

	return &Service{
		log:                            log,
		projectIdToProjectCache:        projectIdToProjectCache,
		urlToProjectAndVersionIdsCache: urlToProjectAndVersionIdsCache,
		api:                            onekoApi,
	}
}

func (o *Service) GetProjectAndVersionForUrl(url string) (*oneko.Project, *oneko.ProjectVersion, error) {
	fromCache := o.urlToProjectAndVersionIdsCache.Get(url)
	if fromCache == nil {
		return nil, nil, fmt.Errorf("no project found with url " + url)
	} else {
		value := fromCache.Value()
		return o.GetProjectAndVersionByIds(value.project, value.projectVersion)
	}
}

func (o *Service) getProjectById(projectId string) (*oneko.Project, error) {
	fromCache := o.projectIdToProjectCache.Get(projectId)
	if fromCache == nil {
		return nil, fmt.Errorf("no project found with id " + projectId)
	} else {
		o.log.Infow("serving project from cache", "projectId", projectId)
		return fromCache.Value(), nil
	}
}

func (o *Service) GetProjectAndVersionByIds(projectUuid, versionUuid string) (*oneko.Project, *oneko.ProjectVersion, error) {
	project, err := o.getProjectById(projectUuid)
	if err != nil {
		return nil, nil, err
	}
	version := project.GetProjectVersionMatchingUuid(versionUuid)
	if version == nil {
		return nil, nil, fmt.Errorf("did not find version with id %s in project with id %s", versionUuid, projectUuid)
	}
	return project, version, nil
}

func (o *Service) TriggerDeployment(projectId, versionId string, ctx context.Context) error {
	err := o.api.Deploy(projectId, versionId, ctx)
	o.projectIdToProjectCache.Delete(projectId)
	return err
}

func (o *Service) GetAllProjectDomains() *utils.Set[string] {
	err := o.ensureUrlToIdCacheIsPopulated()
	if err != nil {
		o.log.Infow("encountered an error populating the url cache", err)
	}

	set := utils.NewSet[string]()
	set.AddAll(o.urlToProjectAndVersionIdsCache.Keys())
	return set
}

func (o *Service) ensureUrlToIdCacheIsPopulated() error {
	if o.urlToProjectAndVersionIdsCache.Len() == 0 {
		return o.populateUrlToIdCache()
	}
	return nil
}

func (o *Service) populateUrlToIdCache() error {
	_, err := populateUrlToIdCacheAndReturnEntryForUrl(o.urlToProjectAndVersionIdsCache, o.log, o.api, "")
	return err
}

func populateUrlToIdCacheAndReturnEntryForUrl(cache *ttlcache.Cache[string, projectAndVersionIds], log *zap.SugaredLogger, onekoApi *api.Api, deploymentUrl string) (*ttlcache.Item[string, projectAndVersionIds], error) {
	projects, err := onekoApi.GetAllProjects(context.Background())
	if err != nil {
		log.Errorw("O-Neko API returned an error", "error", err)
		return nil, err
	}
	var searchEntry *ttlcache.Item[string, projectAndVersionIds]
	for _, project := range projects {
		for _, version := range project.Versions {
			for _, url := range version.Urls {
				entry := projectAndVersionIds{
					project:        project.Uuid,
					projectVersion: version.Uuid,
				}
				urlWithoutProtocolAndPath, err2 := getDeploymentUrlWithoutProtocolAndPath(url)
				if err2 == nil {
					cacheEntry := cache.Set(urlWithoutProtocolAndPath, entry, ttlcache.DefaultTTL)
					if strings.EqualFold(deploymentUrl, urlWithoutProtocolAndPath) {
						searchEntry = cacheEntry
					}
				}
			}
		}
	}
	return searchEntry, nil
}

func getDeploymentUrlWithoutProtocolAndPath(deploymentUrl string) (string, error) {
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
