package oneko

import (
	"github.com/go-resty/resty/v2"
	"o-neko-url-trigger/pkg/o-neko-url-trigger/config"
	"regexp"
	"strings"
)

type cacheEntry struct {
	Project *Project
	Version *ProjectVersion
}

func buildClient(conf *config.Config) *resty.Client {
	client := resty.New()
	client.SetHostURL(conf.ONeko.Api.BaseUrl)
	client.SetBasicAuth(conf.ONeko.Api.Auth.Username, conf.ONeko.Api.Auth.Password)
	return client
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
