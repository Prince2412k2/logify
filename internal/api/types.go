package api

// Container mirrors the gateway's ContainerInfo JSON.
type Container struct {
	Name        string `json:"name"`
	ID          string `json:"id"`
	Status      string `json:"status"`
	Image       string `json:"image"`
	Project     string `json:"project"`
	Environment string `json:"environment"`
	App         string `json:"app"`
	Service     string `json:"service"`
}

// ProjectService is one row inside a project stage from /api/projects.
type ProjectService struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	UUID          string `json:"uuid"`
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	Status        string `json:"status"`
	Image         string `json:"image"`
}

// ProjectStage is an environment within a project (e.g. production / staging).
type ProjectStage struct {
	Name     string           `json:"stage_name"`
	Services []ProjectService `json:"services"`
}

// Project as returned by /api/projects.
type Project struct {
	ID     string         `json:"project_id"`
	Name   string         `json:"project_name"`
	Stages []ProjectStage `json:"stages"`
}

// LogMsg is one frame received over the log WebSocket.
type LogMsg struct {
	Type    string `json:"type"`
	Line    string `json:"line,omitempty"`
	Message string `json:"message,omitempty"`
}

// BuildLog is the latest build log for an application.
type BuildLog struct {
	DeploymentUUID string   `json:"deployment_uuid"`
	Status         string   `json:"status"`
	Commit         string   `json:"commit"`
	CommitMessage  string   `json:"commit_message"`
	CreatedAt      int64    `json:"created_at"`
	FinishedAt     int64    `json:"finished_at"`
	Lines          []string `json:"lines"`
}

// ResourceConfig is the config card for an application or service.
type ResourceConfig struct {
	Kind                string `json:"kind"`
	Name                string `json:"name"`
	Description         string `json:"description,omitempty"`
	FQDN                string `json:"fqdn,omitempty"`
	GitRepository       string `json:"git_repository,omitempty"`
	GitBranch           string `json:"git_branch,omitempty"`
	GitCommit           string `json:"git_commit,omitempty"`
	BuildPack           string `json:"build_pack,omitempty"`
	BaseDirectory       string `json:"base_directory,omitempty"`
	InstallCommand      string `json:"install_command,omitempty"`
	BuildCommand        string `json:"build_command,omitempty"`
	StartCommand        string `json:"start_command,omitempty"`
	PortsExposed        string `json:"ports_exposed,omitempty"`
	PortsMappings       string `json:"ports_mappings,omitempty"`
	Dockerfile          string `json:"dockerfile,omitempty"`
	ImageName           string `json:"image_name,omitempty"`
	ImageTag            string `json:"image_tag,omitempty"`
	HealthCheckEnabled  bool   `json:"health_check_enabled,omitempty"`
	HealthCheckPath     string `json:"health_check_path,omitempty"`
	Status              string `json:"status,omitempty"`
	UpdatedAt           int64  `json:"updated_at,omitempty"`
	DockerComposeRaw    string `json:"docker_compose_raw,omitempty"`
}

// EnvVar is one row from /api/services/{uuid}/env (keys only, values not exposed).
type EnvVar struct {
	Key         string `json:"key"`
	IsPreview   bool   `json:"is_preview"`
	IsBuildTime bool   `json:"is_build_time"`
	IsRuntime   bool   `json:"is_runtime"`
	IsLiteral   bool   `json:"is_literal"`
	IsShared    bool   `json:"is_shared"`
	ValueLength int    `json:"value_length"`
}

// Deployment is one row from /api/services/{uuid}/deployments.
type Deployment struct {
	UUID            string `json:"uuid"`
	Status          string `json:"status"`
	Commit          string `json:"commit"`
	FullCommit      string `json:"full_commit"`
	CommitMessage   string `json:"commit_message"`
	PRID            string `json:"pr_id"`
	Trigger         string `json:"trigger"`
	ForceRebuild    bool   `json:"force_rebuild"`
	CreatedAt       int64  `json:"created_at"`
	FinishedAt      int64  `json:"finished_at"`
	DurationSeconds int64  `json:"duration_seconds"`
}
