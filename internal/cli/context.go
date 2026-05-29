package cli

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/princepatel/logify/internal/api"
	"github.com/princepatel/logify/internal/binding"
	"github.com/princepatel/logify/internal/config"
)

// Env is the shared per-invocation state every command receives.
type Env struct {
	Cfg         config.Config
	CfgPath     string
	BindingPath string
	Bindings    *binding.File
	Client      *api.Client
}

// NewEnv constructs an Env from config + cwd binding discovery.
func NewEnv(cfg config.Config, cfgPath string) (*Env, error) {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	bpath, err := binding.Find(cwd)
	if err != nil {
		return nil, err
	}
	bf, err := binding.Load(bpath)
	if err != nil {
		return nil, err
	}
	return &Env{
		Cfg:         cfg,
		CfgPath:     cfgPath,
		BindingPath: bpath,
		Bindings:    bf,
		Client:      api.New(cfg.URL, cfg.Token),
	}, nil
}

// translate maps an api error to the right CLI error/code pair.
func translate(err error) (int, CLIError) {
	if err == nil {
		return ExitOK, CLIError{}
	}
	var authErr *api.ErrAuth
	if errors.As(err, &authErr) {
		return ExitAuth, CLIError{
			Code:    "AUTH",
			Message: err.Error(),
			Hint:    "set LOGIFY_TOKEN or pass --token; verify the key with the admin",
		}
	}
	if errors.Is(err, api.ErrNotAllowed) {
		return ExitNotAllowed, CLIError{
			Code:    "NOT_ALLOWED",
			Message: err.Error(),
			Hint:    "ask an admin to grant your API key access to this project",
		}
	}
	if errors.Is(err, api.ErrNotReachable) {
		return ExitNotReachable, CLIError{
			Code:    "NOT_REACHABLE",
			Message: err.Error(),
			Hint:    "service runs on a Docker host the gateway can't see, or the container is between deploys",
		}
	}
	var unreach *api.ErrUnreachable
	if errors.As(err, &unreach) {
		return ExitNetwork, CLIError{
			Code:    "NETWORK",
			Message: err.Error(),
			Hint:    "verify --url or LOGIFY_URL points at the gateway; check connectivity",
		}
	}
	return ExitGeneric, CLIError{Code: "INTERNAL", Message: err.Error()}
}

func httpCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 12*time.Second)
}

// translateHTTPStatus is a small helper used by routes that emit a more
// specific code based on the gateway's HTTP status, when known.
func translateHTTPStatus(code int) int {
	switch code {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ExitAuth
	case http.StatusNotFound:
		return ExitNotFound
	}
	return ExitGeneric
}
