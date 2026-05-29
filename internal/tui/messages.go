package tui

import "github.com/princepatel/logify/internal/api"

type ProjectsLoadedMsg struct {
	Projects   []api.Project
	Containers []api.Container
}
type ProjectsErrorMsg struct {
	Kind   string // "unreachable" | "auth" | "empty"
	Detail string
}
type LogLineMsg struct {
	Service string
	Line    string
}
type LogStreamEndedMsg struct {
	Service string
	Err     error
}
type TickMsg struct{}

// SelectionSettledMsg fires after a short debounce when the nav cursor stops
// moving. Token is the debounce sequence; only act if it matches m.navSeq.
type SelectionSettledMsg struct {
	Token int
	Key   string
}

// ClipboardCopiedMsg dismisses the transient "copied" toast.
type ClipboardCopiedMsg struct{}

// DeploymentsLoadedMsg carries deploys for one resource.
type DeploymentsLoadedMsg struct {
	ResourceUUID string
	Deployments  []api.Deployment
}

// DeploymentsErrorMsg reports a failed fetch.
type DeploymentsErrorMsg struct {
	ResourceUUID string
	Err          string
}

type BuildLogLoadedMsg struct {
	ResourceUUID string
	Log          api.BuildLog
}
type BuildLogErrorMsg struct {
	ResourceUUID string
	Err          string
}

type ConfigLoadedMsg struct {
	ResourceUUID string
	Config       api.ResourceConfig
}
type ConfigErrorMsg struct {
	ResourceUUID string
	Err          string
}

type EnvLoadedMsg struct {
	ResourceUUID string
	Vars         []api.EnvVar
}
type EnvErrorMsg struct {
	ResourceUUID string
	Err          string
}

// AdminCheckedMsg carries the result of pinging /api/admin/audit at startup
// to figure out whether the current key is admin (so the header chip + the
// destructive-action shortcuts are gated appropriately).
type AdminCheckedMsg struct {
	IsAdmin bool
}

// AdminActionDoneMsg fires after an admin restart/redeploy POST returns.
type AdminActionDoneMsg struct {
	Action string // "restart" | "redeploy"
	OK     bool
	Detail string
}
