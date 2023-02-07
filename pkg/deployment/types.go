package deployment

type StatusResponse struct {
	DeploymentReady bool   `json:"deploymentReady"`
	RedirectUrl     string `json:"redirectUrl"`
	IsError         bool   `json:"isError"`
	ErrorMessage    string `json:"errorMessage"`
}
