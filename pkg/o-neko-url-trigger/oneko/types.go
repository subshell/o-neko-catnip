package oneko

import "time"

type Project struct {
	Uuid      string           `json:"uuid"`
	Name      string           `json:"name"`
	ImageName string           `json:"imageName"`
	Versions  []ProjectVersion `json:"versions"`
}

type ProjectVersion struct {
	Uuid             string    `json:"uuid"`
	Name             string    `json:"name"`
	Urls             []string  `json:"urls"`
	ImageUpdatedDate time.Time `json:"imageUpdatedDate"`
}
