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
)

type Service struct {
	log          *zap.SugaredLogger
	projectCache *ttlcache.Cache[string, *oneko.Project]
	api          *api.Api
}

func New(configuration *config.Config, ctx context.Context) *Service {

	log := logger.New("onekoSvc")
	api := api.New(configuration)

	projectCache := ttlcache.New[string, *oneko.Project](
		ttlcache.WithTTL[string, *oneko.Project](configuration.ONeko.Api.ApiCallCacheDuration),
		ttlcache.WithDisableTouchOnHit[string, *oneko.Project](),
		ttlcache.WithLoader[string, *oneko.Project](ttlcache.LoaderFunc[string, *oneko.Project](func(c *ttlcache.Cache[string, *oneko.Project], projectId string) *ttlcache.Item[string, *oneko.Project] {
			log.Infow("no cached entry found, calling o-neko api", "projectUuid", projectId)
			project, err := api.GetProjectById(projectId, ctx)
			if err != nil {
				log.Errorw("O-Neko API returned an error", "error", err)
				return nil
			}
			entry := c.Set(projectId, project, ttlcache.DefaultTTL)
			return entry
		})),
	)

	promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "oneko_catnip_cache_size",
		Help: "The number of cached projects",
	}, func() float64 {
		return float64(len(projectCache.Keys()))
	})

	api.StartConnectionMonitor(ctx)

	return &Service{
		log:          log,
		projectCache: projectCache,
		api:          api,
	}
}

func (o *Service) GetProjectAndVersionForUrl(url string, ctx context.Context) (*oneko.Project, *oneko.ProjectVersion, error) {
	project, version := searchForProjectAndVersionMatchingUrl(o.projectCache.Items(), url)

	if project != nil && version != nil {
		o.log.Debugw("serving from cache", "url", url, "project", project.Name, "version", version.Name)
		return project, version, nil
	}

	o.log.Debugw("calling api to get project by url", "url", url)

	deploymentUrl, err := getDeploymentUrlWithoutProtocolAndPath(url)

	if err != nil {
		return nil, nil, err
	}

	project, err = o.api.GetProjectForUrl(url, ctx)

	if err != nil {
		return nil, nil, err
	}

	o.projectCache.Set(project.Uuid, project, ttlcache.DefaultTTL)
	version = project.GetProjectVersionMatchingUrl(deploymentUrl)
	return project, version, nil
}

func searchForProjectAndVersionMatchingUrl(projects map[string]*ttlcache.Item[string, *oneko.Project], url string) (*oneko.Project, *oneko.ProjectVersion) {
	for _, entry := range projects {
		if entry.IsExpired() {
			continue
		}
		if version := entry.Value().GetProjectVersionMatchingUrl(url); version != nil {
			return entry.Value(), version
		}
	}
	return nil, nil
}

func (o *Service) getProjectById(projectId string) (*oneko.Project, error) {
	fromCache := o.projectCache.Get(projectId)
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
	o.projectCache.Delete(projectId)
	return err
}

func (o *Service) GetAllProjectDomains(ctx context.Context) (*utils.Set[string], error) {
	projects, err := o.api.GetAllProjects(ctx)

	if err != nil {
		return nil, err
	}

	domains := utils.NewSet[string]()

	for _, project := range projects {
		for _, version := range project.Versions {
			for _, url := range version.Urls {
				urlWithoutProtocolAndPath, err2 := getDeploymentUrlWithoutProtocolAndPath(url)
				if err2 == nil {
					domains.Add(urlWithoutProtocolAndPath)
				}
			}
		}
	}

	return domains, nil
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
