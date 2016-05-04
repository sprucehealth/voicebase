package deploy

// BuildCompleteEvent represents the notification that is ingested by the deploy service to trigger build time deployments
type BuildCompleteEvent struct {
	DeployableID string `json:"deployable_id"`
	BuildNumber  string `json:"build_number"`
	Image        string `json:"image"`
	GitHash      string `json:"git_hash"`
}
