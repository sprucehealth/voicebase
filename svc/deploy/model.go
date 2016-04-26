package deploy

// Envelope represents the wrapper for multi form deployment notifications
type Envelope struct {
	Event Event `json:"event"`
}

// EventType represents a type of deployment event
type EventType string

const (
	// BuildComplete is an event representing that a build has been completed
	BuildComplete EventType = "BUILD_COMPLETE"
)

// Event represents the inner multi form artifacts for build events
type Event interface {
	Type() EventType
}

// BuildArtifactType represents a type of build artifact
type BuildArtifactType string

const (
	// DockerImage is a build artifact the represents a complete docker image
	DockerImage BuildArtifactType = "DOCKER_IMAGE"
)

// BuildArtifact represents the interface to encapsulate the various artifact inner types
type BuildArtifact interface {
	Type() BuildArtifactType
}

// BuildCompleteEvent represents the notification that is ingested by the deploy service to trigger build time deployments
type BuildCompleteEvent struct {
	DeployableID  string        `json:"deployable_id"`
	BuildNumber   string        `json:"build_number"`
	BuildArtifact BuildArtifact `json:"build_artifact"`
}

// Type returns the type of deploy event this represents
func (a *BuildCompleteEvent) Type() EventType {
	return BuildComplete
}

// DockerImageArtifact is the concrete implementation of BuildArtifact for Docker Images
type DockerImageArtifact struct {
	Image string `json:"image"`
}

// Type returns the type of build artifact this represents
func (a *DockerImageArtifact) Type() BuildArtifactType {
	return DockerImage
}
