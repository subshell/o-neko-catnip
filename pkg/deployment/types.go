package deployment

type StatusResponse struct {
	DeploymentStatus DeploymentStatus `json:"deploymentStatus"`
	RedirectUrl      string           `json:"redirectUrl"`
	ErrorMessage     string           `json:"errorMessage"`
}

type DeploymentStatus string

const (
	Pending DeploymentStatus = "Pending"
	Ready   DeploymentStatus = "Ready"
	Error   DeploymentStatus = "Error"
)
