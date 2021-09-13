package oneko

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/config"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/logger"
	"regexp"
	"strings"
)

type ONekoApi struct {
	client *resty.Client
	log    *zap.SugaredLogger
}

func New(configuration *config.Config) *ONekoApi {
	return &ONekoApi{
		client: buildClient(configuration),
		log:    logger.New("oneko"),
	}
}

func buildClient(conf *config.Config) *resty.Client {
	client := resty.New()
	client.SetHostURL(conf.ONeko.Api.BaseUrl)
	client.SetBasicAuth(conf.ONeko.Api.Auth.Username, conf.ONeko.Api.Auth.Password)
	return client
}

func (o *ONekoApi) HandleRequest(host, uri string) {
	o.log.Infow("incoming request", "host", host, "uri", uri)
	deploymentUrl := fmt.Sprintf("%s%s", host, uri)
	project, err := o.wakeupDeployment(deploymentUrl)
	if err != nil {
		o.log.Errorw("failed to deploy", "deploymentUrl", deploymentUrl, "error", err)
	} else {
		version := getProjectVersionMatchingUrl(project.Versions, deploymentUrl)
		if version != nil {
			o.log.Infow("started deployment", "project", project.Name, "projectVersion", version.Name, "versionDate", version.ImageUpdatedDate)
		} else {
			o.log.Infow("no deployment matching version", "deploymentUrl", deploymentUrl)
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
