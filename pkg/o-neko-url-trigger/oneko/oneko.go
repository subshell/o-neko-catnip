package oneko

import (
	"fmt"
	"github.com/ReneKroon/ttlcache/v2"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/config"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/logger"
	"regexp"
	"strings"
	"time"
)

type ONekoApi struct {
	client *resty.Client
	log    *zap.SugaredLogger
	cache  *ttlcache.Cache
}

type cacheEntry struct {
	Project *Project
	Version *ProjectVersion
}

func New(configuration *config.Config) *ONekoApi {
	requestCache := ttlcache.NewCache()
	requestCache.SetTTL(5 * time.Minute)
	requestCache.SkipTTLExtensionOnHit(true)
	return &ONekoApi{
		client: buildClient(configuration),
		log:    logger.New("oneko"),
		cache:  requestCache,
	}
}

func buildClient(conf *config.Config) *resty.Client {
	client := resty.New()
	client.SetHostURL(conf.ONeko.Api.BaseUrl)
	client.SetBasicAuth(conf.ONeko.Api.Auth.Username, conf.ONeko.Api.Auth.Password)
	return client
}

func (o *ONekoApi) HandleRequest(host, uri string) (*Project, *ProjectVersion, error) {
	o.log.Infow("incoming request", "host", host, "uri", uri)
	deploymentUrl := fmt.Sprintf("%s%s", host, uri)

	fromCache, err := o.cache.Get(deploymentUrl)
	if err != nil && err != ttlcache.ErrNotFound {
		o.log.Warnw("error while checking cache", "deploymentUrl", deploymentUrl, "error", err)
	} else if fromCache != nil {
		o.log.Infow("serving from cache and not calling o-neko to prevent re-starting the application during startup", "deploymentUrl", deploymentUrl)
		entry := fromCache.(cacheEntry)
		return entry.Project, entry.Version, nil
	}

	o.log.Infow("no cached entry found, calling o-neko api", "deploymentUrl", deploymentUrl)
	project, err := o.wakeupDeployment(deploymentUrl)
	if err != nil {
		o.log.Errorw("failed to deploy", "deploymentUrl", deploymentUrl, "error", err)
		return nil, nil, err
	} else {
		version := getProjectVersionMatchingUrl(project.Versions, deploymentUrl)
		if version != nil {
			err = o.cache.Set(deploymentUrl, cacheEntry{
				Project: project,
				Version: version,
			})
			if err != nil {
				o.log.Warnw("failed to insert into cache", "deploymentUrl", deploymentUrl, "error", err)
			}
			o.log.Infow("started deployment", "project", project.Name, "projectVersion", version.Name, "versionDate", version.ImageUpdatedDate)
			return project, version, nil
		} else {
			o.log.Infow("no deployment matching version", "deploymentUrl", deploymentUrl)
			return nil, nil, fmt.Errorf("no matching deployment found")
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
