package common

// DeploymentStatus defines Overall and ready deployment counts
type DeploymentStatus struct {
	ManagedDeployments int32 `json:"managedDeployments"`
	ReadyDeployments   int32 `json:"readyDeployments"`
}
