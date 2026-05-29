package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// ErrNotAllowed is returned when the gateway rejects log access for the
// project the container belongs to.
var ErrNotAllowed = errors.New("not allowed for this project")

// ErrNotReachable is returned when the Docker host can't find a container
// with the given name (deploy-in-progress, on another host, etc.).
var ErrNotReachable = errors.New("container not reachable on this gateway")

// StreamLogs opens the log WebSocket and pushes lines into out until ctx is done.
// Errors are written to errCh; both channels close when the stream terminates.
func (c *Client) StreamLogs(ctx context.Context, containerName string, tail int, out chan<- string, errCh chan<- error) {
	defer close(out)
	defer close(errCh)

	q := url.Values{}
	q.Set("tail", strconv.Itoa(tail))
	u := c.wsURL("/api/logs/"+containerName, q)

	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = 8 * time.Second
	header := map[string][]string{
		"User-Agent": {"logify/" + UserAgent},
	}
	if c.Token != "" {
		header["Authorization"] = []string{"Bearer " + c.Token}
	}

	conn, _, err := dialer.DialContext(ctx, u, header)
	if err != nil {
		errCh <- fmt.Errorf("dial %s: %w", u, err)
		return
	}
	defer conn.Close()

	// First-message auth as a belt-and-suspenders: the gateway accepts header,
	// but explicit first-message is the path it's tuned for.
	authMsg := map[string]any{"type": "auth", "token": c.Token, "tail": tail}
	if b, err := json.Marshal(authMsg); err == nil {
		_ = conn.WriteMessage(websocket.TextMessage, b)
	}

	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		_ = conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second))
		_ = conn.Close()
		close(done)
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			errCh <- err
			return
		}
		var msg LogMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			out <- string(raw)
			continue
		}
		switch msg.Type {
		case "log":
			out <- msg.Line
		case "error":
			// Map known gateway error strings to typed errors so the CLI can
			// emit precise exit codes.
			switch msg.Message {
			case "API key not allowed for project", "API key not allowed for container":
				errCh <- ErrNotAllowed
			case "Container not reachable on this gateway", "Container not found":
				errCh <- ErrNotReachable
			default:
				errCh <- fmt.Errorf("%s", msg.Message)
			}
			return
		}
	}
}
