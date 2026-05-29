package cli

import (
	"strings"

	"github.com/princepatel/logify/internal/api"
)

// ResolvedService is one service after identifier resolution.
type ResolvedService struct {
	UUID          string `json:"uuid"`
	Name          string `json:"name"`
	Project       string `json:"project"`
	Stage         string `json:"stage,omitempty"`
	Path          string `json:"path"`
	Type          string `json:"type"`
	ContainerName string `json:"container_name"`
	Status        string `json:"status,omitempty"`
}

// fetchProjects returns the gateway's project tree.
func fetchProjects(env *Env) ([]api.Project, *CLIError) {
	ctx, cancel := httpCtx()
	defer cancel()
	ps, err := env.Client.Projects(ctx)
	if err != nil {
		_, body := translate(err)
		return nil, &body
	}
	return ps, nil
}

// projectServices returns every service inside one project (matched by
// project_name OR project_id), flattened with paths.
func projectServices(projects []api.Project, projectName, projectID string) []ResolvedService {
	var out []ResolvedService
	for _, p := range projects {
		if projectID != "" && p.ID != projectID {
			continue
		}
		if projectID == "" && !strings.EqualFold(p.Name, projectName) {
			continue
		}
		for _, st := range p.Stages {
			for _, s := range st.Services {
				if s.ContainerName == "" || s.ContainerName == "Not Found" {
					continue
				}
				path := p.Name + "/" + s.Name
				if st.Name != "" && st.Name != p.Name {
					path = p.Name + "/" + st.Name + "/" + s.Name
				}
				out = append(out, ResolvedService{
					UUID: s.UUID, Name: s.Name, Project: p.Name, Stage: st.Name,
					Path: path, Type: s.Type, ContainerName: s.ContainerName,
				})
			}
		}
	}
	return out
}

// allServices returns every service the API key can see, flattened.
func allServices(projects []api.Project) []ResolvedService {
	var out []ResolvedService
	for _, p := range projects {
		for _, st := range p.Stages {
			for _, s := range st.Services {
				if s.ContainerName == "" || s.ContainerName == "Not Found" {
					continue
				}
				path := p.Name + "/" + s.Name
				if st.Name != "" && st.Name != p.Name {
					path = p.Name + "/" + st.Name + "/" + s.Name
				}
				out = append(out, ResolvedService{
					UUID: s.UUID, Name: s.Name, Project: p.Name, Stage: st.Name,
					Path: path, Type: s.Type, ContainerName: s.ContainerName,
				})
			}
		}
	}
	return out
}

// Resolve picks a service from an identifier. If the project binding is set,
// bare names resolve within that project. Without a binding, bare names must
// be globally unique.
func Resolve(env *Env, arg string) (*ResolvedService, *CLIError) {
	q := strings.TrimSpace(arg)
	if q == "" {
		return nil, &CLIError{Code: "BAD_INPUT", Message: "service identifier required",
			Hint: "pass a service name (project-scoped via .logify) or use `logify list`"}
	}
	projects, errx := fetchProjects(env)
	if errx != nil {
		return nil, errx
	}
	var pool []ResolvedService
	if env.Bindings != nil && env.Bindings.IsBound() {
		pool = projectServices(projects, env.Bindings.Project, env.Bindings.ProjectID)
		// If we accidentally got nothing (project renamed/removed), fall back
		// to the full pool so the agent can still resolve global ids.
		if len(pool) == 0 {
			pool = allServices(projects)
		}
	} else {
		pool = allServices(projects)
	}
	return matchOne(pool, q)
}

func matchOne(all []ResolvedService, q string) (*ResolvedService, *CLIError) {
	var byUUID, byContainer, byPath, byName []ResolvedService
	for _, s := range all {
		if s.UUID == q {
			byUUID = append(byUUID, s)
		}
		if s.ContainerName == q {
			byContainer = append(byContainer, s)
		}
		if s.Path == q {
			byPath = append(byPath, s)
		}
		if s.Name == q {
			byName = append(byName, s)
		}
	}
	for _, group := range [][]ResolvedService{byUUID, byContainer, byPath} {
		if len(group) == 1 {
			r := group[0]
			return &r, nil
		}
	}
	if len(byName) == 1 {
		r := byName[0]
		return &r, nil
	}
	if len(byName) > 1 {
		cands := make([]ErrCandidate, 0, len(byName))
		for _, s := range byName {
			cands = append(cands, ErrCandidate{Path: s.Path, UUID: s.UUID})
		}
		return nil, &CLIError{
			Code: "AMBIGUOUS", Message: "multiple services match by name",
			Hint:       "specify a path or uuid",
			Candidates: cands,
		}
	}
	return nil, &CLIError{Code: "NOT_FOUND", Message: "no service matches '" + q + "'",
		Hint: "run `logify list` to discover services in this project"}
}
