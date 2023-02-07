package oneko

import (
	"regexp"
	"strings"
	"time"
)

type Project struct {
	Uuid      string           `json:"uuid"`
	Name      string           `json:"name"`
	ImageName string           `json:"imageName"`
	Versions  []ProjectVersion `json:"versions"`
}

type ProjectVersion struct {
	Uuid             string       `json:"uuid"`
	Name             string       `json:"name"`
	Urls             []string     `json:"urls"`
	ImageUpdatedDate time.Time    `json:"imageUpdatedDate"`
	DesiredState     DesiredState `json:"desiredState"`
	Deployment       Deployment   `json:"deployment"`
}

type DesiredState string

const (
	Deployed    DesiredState = "Deployed"
	NotDeployed DesiredState = "NotDeployed"
)

type Deployment struct {
	Status    DeployableStatus `json:"status"`
	Timestamp time.Time        `json:"timestamp"`
}

type DeployableStatus string

const (
	Pending      DeployableStatus = "Pending"
	Running      DeployableStatus = "Running"
	Failed       DeployableStatus = "Failed"
	Unknown      DeployableStatus = "Unknown"
	NotScheduled DeployableStatus = "NotScheduled"
)

func (p Project) GetProjectVersionMatchingUrl(url string) *ProjectVersion {
	r, err := regexp.Compile("^https?://")
	if err != nil {
		return nil
	}
	urlWithoutProtocol := r.ReplaceAllString(url, "")
	for _, version := range p.Versions {
		for _, url := range version.Urls {
			versionUrlWithoutProtocol := r.ReplaceAllString(url, "")
			if strings.HasPrefix(urlWithoutProtocol, versionUrlWithoutProtocol) {
				return &version
			}
		}
	}
	return nil
}

func (p Project) GetProjectVersionMatchingUuid(versionUuid string) *ProjectVersion {
	for _, version := range p.Versions {
		if strings.EqualFold(versionUuid, version.Uuid) {
			return &version
		}
	}
	return nil
}

func (v ProjectVersion) IsDeployed() bool {
	return v.DesiredState == Deployed
}
