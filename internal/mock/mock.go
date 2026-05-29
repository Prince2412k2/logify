package mock

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/princepatel/logify/internal/api"
)

// Projects returns a mock service tree mirroring reference/mock-data.jsx.
func Projects() []api.Project {
	return []api.Project{
		{Name: "prod", Stages: []api.ProjectStage{{Name: "production", Services: []api.ProjectService{
			{Name: "api", ContainerName: "prod-api", Status: "running", Image: "ghcr.io/acme/api:1.42.0"},
			{Name: "web", ContainerName: "prod-web", Status: "running", Image: "ghcr.io/acme/web:1.42.0"},
			{Name: "worker", ContainerName: "prod-worker", Status: "running", Image: "ghcr.io/acme/worker:1.42.0"},
			{Name: "postgres", ContainerName: "prod-postgres", Status: "running", Image: "postgres:16.4-alpine"},
			{Name: "redis", ContainerName: "prod-redis", Status: "running", Image: "redis:7.4"},
		}}}},
		{Name: "staging", Stages: []api.ProjectStage{{Name: "staging", Services: []api.ProjectService{
			{Name: "api", ContainerName: "staging-api", Status: "exited", Image: "ghcr.io/acme/api:1.43.0-rc.2"},
			{Name: "web", ContainerName: "staging-web", Status: "unknown", Image: "ghcr.io/acme/web:1.43.0-rc.2"},
			{Name: "postgres", ContainerName: "staging-postgres", Status: "running", Image: "postgres:16.4-alpine"},
		}}}},
		{Name: "infra", Stages: []api.ProjectStage{{Name: "shared", Services: []api.ProjectService{
			{Name: "traefik", ContainerName: "infra-traefik", Status: "running", Image: "traefik:v3.1"},
			{Name: "loki", ContainerName: "infra-loki", Status: "restarting", Image: "grafana/loki:3.1.0"},
			{Name: "grafana", ContainerName: "infra-grafana", Status: "running", Image: "grafana/grafana:11.2.0"},
		}}}},
	}
}

type entry struct {
	level string
	msg   string
}

var pools = map[string][]entry{
	"api": {
		{"INFO", "GET  /api/v1/orders        200  18ms"},
		{"INFO", "GET  /api/v1/orders/:id    200   4ms"},
		{"INFO", "POST /api/v1/orders        201  62ms"},
		{"INFO", "GET  /api/v1/users/me      200   2ms"},
		{"INFO", "PATCH /api/v1/profile      200  31ms"},
		{"INFO", "GET  /healthz              200   0ms"},
		{"DEBUG", "cache hit user:48211 ttl=287s"},
		{"DEBUG", "auth: validated jwt sub=usr_48211 iss=clerk"},
		{"WARN", "slow query: SELECT * FROM events WHERE ... (1.24s)"},
		{"WARN", "rate-limit near cap for ip=10.4.7.91 (98/100/min)"},
		{"ERROR", "connection refused: dial tcp 10.4.1.3:5432: connect: connection refused"},
		{"ERROR", "    at db.go:88 (*Pool).Acquire"},
		{"INFO", "reconnecting to postgres (attempt 1/5)"},
		{"INFO", "postgres pool ready (15/15 conns)"},
	},
	"web": {
		{"INFO", "▲ Next.js 15.0.3"},
		{"INFO", "  - Local:   http://localhost:3000"},
		{"INFO", "✓ Ready in 1.2s"},
		{"INFO", "GET /dashboard              200 in 152ms"},
		{"WARN", "image domain not configured: cdn.example.com"},
		{"DEBUG", "RSC: streaming /dashboard tree=12 chunks"},
	},
	"worker": {
		{"INFO", "queue:email   processing job 8a3b21f (welcome)"},
		{"INFO", "queue:email   completed   8a3b21f in 412ms"},
		{"INFO", "queue:billing processing job 7c11ae0 (invoice.send)"},
		{"WARN", "queue:webhooks retry 2/5 job 9f02e1c (stripe.subscription.updated)"},
		{"DEBUG", "scheduler tick: 3 jobs scanned, 1 dispatched"},
	},
	"postgres": {
		{"INFO", "LOG:  database system is ready to accept connections"},
		{"INFO", "LOG:  checkpoint starting: time"},
		{"INFO", "LOG:  checkpoint complete: wrote 124 buffers (0.8%)"},
		{"DEBUG", "LOG:  duration: 0.512 ms  statement: SELECT 1"},
	},
	"redis": {
		{"INFO", "1:M Ready to accept connections tcp"},
		{"INFO", "1:M Background AOF rewrite finished successfully"},
		{"DEBUG", "1:M Background AOF rewrite started by pid 19"},
	},
	"traefik": {
		{"INFO", "Configuration loaded from flags."},
		{"INFO", "Provider connection established docker (3.2s)"},
		{"WARN", "Skipping configuration: router exists for service@docker"},
	},
	"loki": {
		{"INFO", "level=info ts=now caller=main.go:108 msg=\"Starting Loki\" version=3.1.0"},
		{"ERROR", "level=error caller=ingester.go:392 msg=\"failed to flush chunks\" err=\"permission denied\""},
		{"WARN", "level=warn caller=table_manager.go:171 msg=\"creating table chunks_19788\""},
	},
	"grafana": {
		{"INFO", "logger=server msg=\"HTTP Server Listen\" address=[::]:3000 protocol=http"},
		{"INFO", "logger=plugin.loader msg=\"Plugin registered\" pluginId=grafana-piechart-panel"},
		{"DEBUG", "logger=context userId=1 orgId=1 method=GET path=/api/dashboards/uid/a1b2c3"},
	},
}

var boot = []entry{
	{"INFO", "starting server (build 1.42.0 · go1.23.4)"},
	{"INFO", "loading config from /etc/api/config.yaml"},
	{"INFO", "connecting to postgres://prod-db:5432/api ..."},
	{"INFO", "postgres pool ready (15/15 conns)"},
	{"INFO", "connecting to redis://prod-cache:6379 ..."},
	{"INFO", "redis ready"},
	{"INFO", "migrations: up to date (47 applied)"},
	{"INFO", "listening :8080"},
}

func poolFor(service string) []entry {
	for k, v := range pools {
		if k == service || (len(service) >= len(k) && service[len(service)-len(k):] == k) {
			return v
		}
	}
	return pools["api"]
}

func pick(pool []entry, r *rand.Rand) entry {
	v := r.Float64()
	want := "INFO"
	switch {
	case v < 0.04:
		want = "ERROR"
	case v < 0.14:
		want = "WARN"
	case v < 0.30:
		want = "DEBUG"
	}
	for _, n := range []int{0, 1, 2, 3} {
		idx := (r.Intn(len(pool)) + n) % len(pool)
		if pool[idx].level == want {
			return pool[idx]
		}
	}
	return pool[r.Intn(len(pool))]
}

// InitialLines returns a seeded buffer (boot + N follow-on lines) for service.
func InitialLines(service string, n int) []string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pool := poolFor(service)
	t := time.Now().Add(-time.Duration(n) * 800 * time.Millisecond)
	out := make([]string, 0, n)
	for _, e := range boot {
		out = append(out, formatLine(t, e))
		t = t.Add(250 * time.Millisecond)
	}
	for i := 0; i < n-len(boot); i++ {
		e := pick(pool, r)
		out = append(out, formatLine(t, e))
		t = t.Add(time.Duration(600+r.Intn(800)) * time.Millisecond)
	}
	return out
}

func formatLine(t time.Time, e entry) string {
	return fmt.Sprintf("%s %-5s %s", t.Format("15:04:05"), e.level, e.msg)
}

// Stream pushes mock lines into out at speed until ctx is done.
func Stream(ctx context.Context, service string, speed time.Duration, out chan<- string, errCh chan<- error) {
	defer close(out)
	defer close(errCh)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pool := poolFor(service)
	tick := time.NewTicker(speed)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			e := pick(pool, r)
			out <- formatLine(time.Now(), e)
		}
	}
}
