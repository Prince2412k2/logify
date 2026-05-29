package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client talks to the log-gateway over HTTP/WS.
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

type ErrAuth struct{ Status int }

func (e *ErrAuth) Error() string { return fmt.Sprintf("authentication failed (%d)", e.Status) }

type ErrUnreachable struct{ URL string; Inner error }

func (e *ErrUnreachable) Error() string {
	if e.Inner != nil {
		return fmt.Sprintf("cannot reach %s: %s", e.URL, e.Inner)
	}
	return fmt.Sprintf("cannot reach %s", e.URL)
}
func (e *ErrUnreachable) Unwrap() error { return e.Inner }

func (c *Client) request(ctx context.Context, method, path string, out any) error {
	u := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, u, nil)
	if err != nil {
		return err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return &ErrUnreachable{URL: u, Inner: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		io.Copy(io.Discard, resp.Body)
		return &ErrAuth{Status: resp.StatusCode}
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) Health(ctx context.Context) error {
	return c.request(ctx, http.MethodGet, "/health", nil)
}

func (c *Client) Projects(ctx context.Context) ([]Project, error) {
	var out []Project
	if err := c.request(ctx, http.MethodGet, "/api/projects", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) Containers(ctx context.Context) ([]Container, error) {
	var out []Container
	if err := c.request(ctx, http.MethodGet, "/api/containers", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) Deployments(ctx context.Context, resourceUUID string) ([]Deployment, error) {
	var out []Deployment
	if err := c.request(ctx, http.MethodGet, "/api/services/"+resourceUUID+"/deployments", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) BuildLog(ctx context.Context, resourceUUID string) (BuildLog, error) {
	var out BuildLog
	if err := c.request(ctx, http.MethodGet, "/api/services/"+resourceUUID+"/build-log", &out); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) ResourceConfig(ctx context.Context, resourceUUID string) (ResourceConfig, error) {
	var out ResourceConfig
	if err := c.request(ctx, http.MethodGet, "/api/services/"+resourceUUID+"/config", &out); err != nil {
		return out, err
	}
	return out, nil
}

func (c *Client) EnvVars(ctx context.Context, resourceUUID string) ([]EnvVar, error) {
	var out []EnvVar
	if err := c.request(ctx, http.MethodGet, "/api/services/"+resourceUUID+"/env", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// wsURL converts http(s):// → ws(s):// for the gateway base.
func (c *Client) wsURL(path string, q url.Values) string {
	u := c.BaseURL + path
	if strings.HasPrefix(u, "https://") {
		u = "wss://" + strings.TrimPrefix(u, "https://")
	} else {
		u = "ws://" + strings.TrimPrefix(u, "http://")
	}
	if q != nil {
		u += "?" + q.Encode()
	}
	return u
}
