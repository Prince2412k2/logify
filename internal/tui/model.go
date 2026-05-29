package tui

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/princepatel/logify/internal/api"
	"github.com/princepatel/logify/internal/binding"
	"github.com/princepatel/logify/internal/config"
	"github.com/princepatel/logify/internal/mock"
	"github.com/princepatel/logify/internal/theme"
)

// thin wrappers so we don't carry the binding package name through every call.
func bindingFind(dir string) (string, error)        { return binding.Find(dir) }
func bindingLoad(path string) (*binding.File, error) { return binding.Load(path) }
func bindingSave(path string, f *binding.File) error { return binding.Save(path, f) }

func (m *Model) workingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

type viewMode int

const (
	viewMain viewMode = iota
	viewFirstRun
	viewConnecting
	viewErrorUnreachable
	viewErrorAuth
	viewErrorEmpty
)

// Model holds the full TUI state.
type Model struct {
	cfg     config.Config
	cfgPath string

	client *api.Client
	mock   bool

	layout Layout
	theme  theme.Theme

	view         viewMode
	showHelp     bool
	showPicker   bool
	pickerIdx    int

	// nav
	projects        []api.Project
	containers      []api.Container
	navRows         []ServiceRow
	selectedIdx     int // index into selectableKeys
	selectableKeys  []string
	navQuery        string
	navFiltering    bool

	// right pane
	activeTab string
	focused   string

	// logs
	logs       []LogLine
	queued     int
	paused     bool
	levels     map[string]bool
	showLvlBar bool
	wrap       bool
	fullscreen bool
	// scroll offset from the newest line. 0 = follow tail.
	logsOffset int

	// v2: telescope picker (replaces sidebar)
	telescope TelescopeState

	// search
	searchOpen  bool
	searchQuery string

	// stream
	streamCtx     context.Context
	streamCancel  context.CancelFunc
	streamCh      chan LogLineMsg
	streamErr     chan LogStreamEndedMsg
	streamSvc     string // service display name
	streamKey     string // container name (unique stream identity)
	streamDropped bool

	// nav debounce
	navSeq int

	// transient notice (e.g. "Copied 42 lines")
	notice string

	// animation frame counter, advances on TickMsg
	frame int

	// deploys cache keyed by resource UUID
	deploys        map[string][]api.Deployment
	deploysLoading map[string]bool
	deploysErrs    map[string]string

	// build log cache
	builds        map[string]api.BuildLog
	buildsLoading map[string]bool
	buildsErrs    map[string]string

	// config cache
	configs        map[string]api.ResourceConfig
	configsLoading map[string]bool
	configsErrs    map[string]string

	// env cache
	envs        map[string][]api.EnvVar
	envsLoading map[string]bool
	envsErrs    map[string]string

	// first-run form
	frFocus     int    // 0=url 1=token 2=save 3=quit
	frURL       string
	frToken     string
	frError     string

	// error detail
	errorDetail string
	errorTarget string

	// transient
	width, height int
	lastTick      time.Time
}

// New builds a starting Model from config + flags.
func New(cfg config.Config, cfgPath string, useMock bool) Model {
	m := Model{
		cfg:            cfg,
		cfgPath:        cfgPath,
		mock:           useMock,
		theme:          theme.Get(cfg.Theme),
		layout:         NewLayout(DefTotalW, DefTotalH),
		view:           viewConnecting,
		focused:        "nav",
		activeTab:      "logs",
		levels:         map[string]bool{"err": true, "warn": true, "info": true, "debug": true},
		frURL:          cfg.URL,
		frToken:        cfg.Token,
		deploys:        map[string][]api.Deployment{},
		deploysLoading: map[string]bool{},
		deploysErrs:    map[string]string{},
		builds:         map[string]api.BuildLog{},
		buildsLoading:  map[string]bool{},
		buildsErrs:     map[string]string{},
		configs:        map[string]api.ResourceConfig{},
		configsLoading: map[string]bool{},
		configsErrs:    map[string]string{},
		envs:           map[string][]api.EnvVar{},
		envsLoading:    map[string]bool{},
		envsErrs:       map[string]string{},
	}
	if !useMock && cfg.Token == "" {
		m.view = viewFirstRun
		m.frFocus = 0
	}
	if !useMock {
		m.client = api.New(cfg.URL, cfg.Token)
	}
	return m
}

func (m Model) Init() tea.Cmd {
	if m.view == viewFirstRun {
		return nil
	}
	return tea.Batch(m.loadProjects(), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return TickMsg{} })
}

func (m *Model) loadProjects() tea.Cmd {
	if m.mock {
		return func() tea.Msg {
			return ProjectsLoadedMsg{Projects: mock.Projects()}
		}
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		ps, perr := m.client.Projects(ctx)
		var cs []api.Container
		if perr == nil && len(ps) == 0 {
			cs, _ = m.client.Containers(ctx)
		}
		if perr != nil {
			var authErr *api.ErrAuth
			if errors.As(perr, &authErr) {
				return ProjectsErrorMsg{Kind: "auth"}
			}
			var unr *api.ErrUnreachable
			if errors.As(perr, &unr) {
				return ProjectsErrorMsg{Kind: "unreachable", Detail: unr.Inner.Error()}
			}
			// Try containers as a last-resort fallback before erroring.
			cs2, cerr := m.client.Containers(ctx)
			if cerr == nil {
				return ProjectsLoadedMsg{Containers: cs2}
			}
			return ProjectsErrorMsg{Kind: "unreachable", Detail: perr.Error()}
		}
		return ProjectsLoadedMsg{Projects: ps, Containers: cs}
	}
}

func (m *Model) startStream(svc, container string) tea.Cmd {
	// If the same container is already streaming, no-op.
	if m.streamKey == container && m.streamCancel != nil {
		return nil
	}
	m.stopStream()
	m.streamSvc = svc
	m.streamKey = container
	m.streamDropped = false
	m.logs = nil
	m.queued = 0
	m.logsOffset = 0
	ctx, cancel := context.WithCancel(context.Background())
	m.streamCtx, m.streamCancel = ctx, cancel
	out := make(chan string, 256)
	errCh := make(chan error, 1)
	m.streamCh = make(chan LogLineMsg, 256)
	m.streamErr = make(chan LogStreamEndedMsg, 1)

	if m.mock {
		go mock.Stream(ctx, svc, 900*time.Millisecond, out, errCh)
	} else {
		go m.client.StreamLogs(ctx, container, m.cfg.Tail, out, errCh)
	}
	go func() {
		for line := range out {
			select {
			case m.streamCh <- LogLineMsg{Service: svc, Line: line}:
			case <-ctx.Done():
				return
			}
		}
		var err error
		select {
		case err = <-errCh:
		default:
		}
		m.streamErr <- LogStreamEndedMsg{Service: svc, Err: err}
	}()
	return tea.Batch(m.recvLine(), m.recvEnd())
}

func (m *Model) recvLine() tea.Cmd {
	ch := m.streamCh
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *Model) recvEnd() tea.Cmd {
	ch := m.streamErr
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *Model) stopStream() {
	if m.streamCancel != nil {
		m.streamCancel()
		m.streamCancel = nil
	}
	m.streamKey = ""
}

// maybeFetchDeploys fires a fetch if the Deploys tab is active for an
// application resource and we don't already have data (or an in-flight load).
func (m *Model) maybeFetchDeploys() tea.Cmd {
	if m.activeTab != "deploys" {
		return nil
	}
	row, ok := m.currentRow()
	if !ok || row.UUID == "" || row.Type != "application" {
		return nil
	}
	if _, has := m.deploys[row.UUID]; has {
		return nil
	}
	if m.deploysLoading[row.UUID] {
		return nil
	}
	m.deploysLoading[row.UUID] = true
	delete(m.deploysErrs, row.UUID)
	return m.fetchDeployments(row.UUID)
}

func (m *Model) fetchDeployments(uuid string) tea.Cmd {
	if m.mock {
		return func() tea.Msg {
			return DeploymentsLoadedMsg{ResourceUUID: uuid, Deployments: mockDeployments()}
		}
	}
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		ds, err := client.Deployments(ctx, uuid)
		if err != nil {
			return DeploymentsErrorMsg{ResourceUUID: uuid, Err: err.Error()}
		}
		return DeploymentsLoadedMsg{ResourceUUID: uuid, Deployments: ds}
	}
}

func (m *Model) buildDeploysState() DeploysState {
	row, ok := m.currentRow()
	if !ok {
		return DeploysState{}
	}
	return DeploysState{
		ResourceUUID: row.UUID,
		Type:         row.Type,
		Loading:      m.deploysLoading[row.UUID],
		Err:          m.deploysErrs[row.UUID],
		Deployments:  m.deploys[row.UUID],
	}
}

func (m *Model) buildBuildState() BuildState {
	row, ok := m.currentRow()
	if !ok {
		return BuildState{}
	}
	return BuildState{
		ResourceUUID: row.UUID,
		Type:         row.Type,
		Loading:      m.buildsLoading[row.UUID],
		Err:          m.buildsErrs[row.UUID],
		Log:          m.builds[row.UUID],
	}
}

func (m *Model) buildConfigState() ConfigState {
	row, ok := m.currentRow()
	if !ok {
		return ConfigState{}
	}
	return ConfigState{
		ResourceUUID: row.UUID,
		Type:         row.Type,
		Loading:      m.configsLoading[row.UUID],
		Err:          m.configsErrs[row.UUID],
		Config:       m.configs[row.UUID],
	}
}

func (m *Model) buildEnvState() EnvState {
	row, ok := m.currentRow()
	if !ok {
		return EnvState{}
	}
	return EnvState{
		ResourceUUID: row.UUID,
		Loading:      m.envsLoading[row.UUID],
		Err:          m.envsErrs[row.UUID],
		Vars:         m.envs[row.UUID],
	}
}

// ── Build log fetch ─────────────────────────────────────────────────────

func (m *Model) maybeFetchBuild() tea.Cmd {
	if m.activeTab != "build" {
		return nil
	}
	row, ok := m.currentRow()
	if !ok || row.UUID == "" || row.Type != "application" {
		return nil
	}
	if _, has := m.builds[row.UUID]; has {
		return nil
	}
	if m.buildsLoading[row.UUID] {
		return nil
	}
	m.buildsLoading[row.UUID] = true
	delete(m.buildsErrs, row.UUID)
	uuid := row.UUID
	if m.mock {
		return func() tea.Msg { return BuildLogLoadedMsg{ResourceUUID: uuid, Log: mockBuildLog()} }
	}
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		bl, err := client.BuildLog(ctx, uuid)
		if err != nil {
			return BuildLogErrorMsg{ResourceUUID: uuid, Err: err.Error()}
		}
		return BuildLogLoadedMsg{ResourceUUID: uuid, Log: bl}
	}
}

// ── Config fetch ────────────────────────────────────────────────────────

func (m *Model) maybeFetchConfig() tea.Cmd {
	if m.activeTab != "config" {
		return nil
	}
	row, ok := m.currentRow()
	if !ok || row.UUID == "" {
		return nil
	}
	if _, has := m.configs[row.UUID]; has {
		return nil
	}
	if m.configsLoading[row.UUID] {
		return nil
	}
	m.configsLoading[row.UUID] = true
	delete(m.configsErrs, row.UUID)
	uuid := row.UUID
	if m.mock {
		return func() tea.Msg { return ConfigLoadedMsg{ResourceUUID: uuid, Config: mockConfig(row.Type)} }
	}
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		c, err := client.ResourceConfig(ctx, uuid)
		if err != nil {
			return ConfigErrorMsg{ResourceUUID: uuid, Err: err.Error()}
		}
		return ConfigLoadedMsg{ResourceUUID: uuid, Config: c}
	}
}

// ── Env fetch ───────────────────────────────────────────────────────────

func (m *Model) maybeFetchEnv() tea.Cmd {
	if m.activeTab != "env" {
		return nil
	}
	row, ok := m.currentRow()
	if !ok || row.UUID == "" {
		return nil
	}
	if _, has := m.envs[row.UUID]; has {
		return nil
	}
	if m.envsLoading[row.UUID] {
		return nil
	}
	m.envsLoading[row.UUID] = true
	delete(m.envsErrs, row.UUID)
	uuid := row.UUID
	if m.mock {
		return func() tea.Msg { return EnvLoadedMsg{ResourceUUID: uuid, Vars: mockEnvVars()} }
	}
	client := m.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		es, err := client.EnvVars(ctx, uuid)
		if err != nil {
			return EnvErrorMsg{ResourceUUID: uuid, Err: err.Error()}
		}
		return EnvLoadedMsg{ResourceUUID: uuid, Vars: es}
	}
}

func (m *Model) maybeFetchActive() tea.Cmd {
	switch m.activeTab {
	case "build":
		return m.maybeFetchBuild()
	case "config":
		return m.maybeFetchConfig()
	case "env":
		return m.maybeFetchEnv()
	case "deploys":
		return m.maybeFetchDeploys()
	}
	return nil
}

func mockBuildLog() api.BuildLog {
	now := time.Now().Unix()
	return api.BuildLog{
		DeploymentUUID: "dep-9f02e1c",
		Status:         "finished",
		Commit:         "a3b2c1f",
		CommitMessage:  "feat: stream build logs",
		CreatedAt:      now - 3600,
		FinishedAt:     now - 3520,
		Lines: []string{
			"#1 [internal] load build definition from Dockerfile",
			"#1 transferring dockerfile: 1.42kB done",
			"#1 DONE 0.1s",
			"#2 [internal] load metadata for docker.io/library/node:20-alpine",
			"#2 DONE 1.2s",
			"#3 [1/6] FROM docker.io/library/node:20-alpine",
			"#3 CACHED",
			"#4 [2/6] WORKDIR /app",
			"#4 CACHED",
			"#5 [3/6] COPY package*.json ./",
			"#5 DONE 0.3s",
			"#6 [4/6] RUN npm ci --omit=dev",
			"#6 17.4 added 412 packages in 17s",
			"#6 DONE 18.1s",
			"#7 [5/6] COPY . .",
			"#7 DONE 0.2s",
			"#8 [6/6] RUN npm run build",
			"#8 12.8 ready in 11824ms",
			"#8 DONE 13.0s",
			"exporting to image",
			"writing image sha256:8f3a2c…",
			"naming to ghcr.io/acme/api:1.42.0 done",
			"DONE 19.4s",
		},
	}
}

func mockConfig(kind string) api.ResourceConfig {
	if kind == "service" {
		return api.ResourceConfig{
			Kind: "service", Name: "redis",
			Description: "Cache for the API",
			DockerComposeRaw: "version: '3'\nservices:\n  redis:\n    image: redis:7.4\n    restart: unless-stopped\n    ports:\n      - 6379:6379\n",
			Status: "running:healthy",
		}
	}
	return api.ResourceConfig{
		Kind: "application", Name: "api",
		FQDN: "https://api.acme.test",
		GitRepository: "git@github.com:acme/api.git", GitBranch: "main", GitCommit: "a3b2c1f8e7d6",
		BuildPack: "nixpacks", InstallCommand: "npm ci", BuildCommand: "npm run build", StartCommand: "node dist/server.js",
		BaseDirectory: "/", PortsExposed: "8080", PortsMappings: "8080:8080",
		HealthCheckEnabled: true, HealthCheckPath: "/healthz",
		Status: "running:healthy",
	}
}

func mockEnvVars() []api.EnvVar {
	return []api.EnvVar{
		{Key: "DATABASE_URL", ValueLength: 84, IsBuildTime: false},
		{Key: "REDIS_URL", ValueLength: 32, IsBuildTime: false},
		{Key: "JWT_SECRET", ValueLength: 64, IsBuildTime: false},
		{Key: "LOG_LEVEL", ValueLength: 4, IsBuildTime: false},
		{Key: "NODE_ENV", ValueLength: 10, IsBuildTime: true},
		{Key: "NEXT_PUBLIC_API_URL", ValueLength: 28, IsBuildTime: true, IsPreview: false},
		{Key: "SENTRY_DSN", ValueLength: 96, IsBuildTime: false, IsPreview: true},
	}
}

func mockDeployments() []api.Deployment {
	now := time.Now().Unix()
	return []api.Deployment{
		{UUID: "dep-9f02e1c", Status: "in_progress", Commit: "a3b2c1f", CommitMessage: "feat: stream build logs over WS", Trigger: "webhook", CreatedAt: now - 90, FinishedAt: 0, DurationSeconds: 0},
		{UUID: "dep-7c11ae0", Status: "finished", Commit: "d4e5f6a", CommitMessage: "fix: pool exhaustion under load", Trigger: "webhook", CreatedAt: now - 3600, FinishedAt: now - 3520, DurationSeconds: 80},
		{UUID: "dep-8a3b21f", Status: "failed", Commit: "b9c8d7e", CommitMessage: "wip: experimental cache layer", Trigger: "manual", CreatedAt: now - 86400, FinishedAt: now - 86355, DurationSeconds: 45},
		{UUID: "dep-3c2b1a0", Status: "finished", Commit: "f0e1d2c", CommitMessage: "rollback to last known good", Trigger: "rollback", CreatedAt: now - 172800, FinishedAt: now - 172680, DurationSeconds: 120},
	}
}

// queueStreamForSelection bumps the debounce counter and schedules a settle msg
// that fires after `debounce` if no further nav moves arrive.
func (m *Model) queueStreamForSelection() tea.Cmd {
	m.navSeq++
	token := m.navSeq
	key := m.selectedKey()
	return tea.Tick(180*time.Millisecond, func(time.Time) tea.Msg {
		return SelectionSettledMsg{Token: token, Key: key}
	})
}

// selectFirstService picks the first usable service (or none).
func (m *Model) selectFirstService() {
	m.selectableKeys = nil
	for _, r := range m.navRows {
		if !r.IsHeader {
			m.selectableKeys = append(m.selectableKeys, r.Key)
		}
	}
	if len(m.selectableKeys) == 0 {
		m.view = viewErrorEmpty
		return
	}
	m.selectedIdx = 0
}

func (m *Model) currentRow() (ServiceRow, bool) {
	if m.selectedIdx < 0 || m.selectedIdx >= len(m.selectableKeys) {
		return ServiceRow{}, false
	}
	want := m.selectableKeys[m.selectedIdx]
	for _, r := range m.navRows {
		if !r.IsHeader && r.Key == want {
			return r, true
		}
	}
	return ServiceRow{}, false
}

func (m *Model) selectedKey() string {
	if m.selectedIdx >= 0 && m.selectedIdx < len(m.selectableKeys) {
		return m.selectableKeys[m.selectedIdx]
	}
	return ""
}

func (m *Model) selectedPath() string {
	if r, ok := m.currentRow(); ok {
		return r.Path
	}
	return ""
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout = NewLayout(msg.Width, msg.Height)
		return m, nil

	case TickMsg:
		m.frame++
		return m, tickCmd()

	case ProjectsLoadedMsg:
		m.projects = msg.Projects
		m.containers = msg.Containers
		if len(msg.Projects) > 0 {
			m.navRows = FlattenProjects(msg.Projects)
		} else {
			m.navRows = FlattenContainers(msg.Containers)
		}
		m.selectFirstService()
		if m.view == viewErrorEmpty {
			return m, nil
		}
		m.view = viewMain
		// v2: if .logify remembers a last_service, reselect it.
		if last := m.loadLastService(); last != "" {
			for j, k := range m.selectableKeys {
				if rowMatchesService(m.navRows, k, last) {
					m.selectedIdx = j
					break
				}
			}
		}
		row, ok := m.currentRow()
		if !ok {
			return m, nil
		}
		return m, m.startStream(row.Service, row.ContainerName)

	case SelectionSettledMsg:
		// Only act if no nav move happened in the meantime AND the key still
		// matches what's selected (i.e. the user actually settled here).
		if msg.Token != m.navSeq {
			return m, nil
		}
		if msg.Key != m.selectedKey() {
			return m, nil
		}
		row, ok := m.currentRow()
		if !ok {
			return m, nil
		}
		return m, tea.Batch(m.startStream(row.Service, row.ContainerName), m.maybeFetchActive())

	case ClipboardCopiedMsg:
		m.notice = ""
		return m, nil

	case DeploymentsLoadedMsg:
		m.deploys[msg.ResourceUUID] = msg.Deployments
		delete(m.deploysLoading, msg.ResourceUUID)
		return m, nil

	case DeploymentsErrorMsg:
		m.deploysErrs[msg.ResourceUUID] = msg.Err
		delete(m.deploysLoading, msg.ResourceUUID)
		return m, nil

	case BuildLogLoadedMsg:
		m.builds[msg.ResourceUUID] = msg.Log
		delete(m.buildsLoading, msg.ResourceUUID)
		return m, nil
	case BuildLogErrorMsg:
		m.buildsErrs[msg.ResourceUUID] = msg.Err
		delete(m.buildsLoading, msg.ResourceUUID)
		return m, nil

	case ConfigLoadedMsg:
		m.configs[msg.ResourceUUID] = msg.Config
		delete(m.configsLoading, msg.ResourceUUID)
		return m, nil
	case ConfigErrorMsg:
		m.configsErrs[msg.ResourceUUID] = msg.Err
		delete(m.configsLoading, msg.ResourceUUID)
		return m, nil

	case EnvLoadedMsg:
		m.envs[msg.ResourceUUID] = msg.Vars
		delete(m.envsLoading, msg.ResourceUUID)
		return m, nil
	case EnvErrorMsg:
		m.envsErrs[msg.ResourceUUID] = msg.Err
		delete(m.envsLoading, msg.ResourceUUID)
		return m, nil

	case ProjectsErrorMsg:
		switch msg.Kind {
		case "auth":
			m.view = viewErrorAuth
		case "empty":
			m.view = viewErrorEmpty
		default:
			m.view = viewErrorUnreachable
			m.errorTarget = m.cfg.URL + "/api/projects"
			m.errorDetail = msg.Detail
		}
		return m, nil

	case LogLineMsg:
		if msg.Service != m.streamSvc {
			return m, m.recvLine()
		}
		if m.paused {
			m.queued++
		} else {
			ll := ParseLine(msg.Line, time.Now().Format("15:04:05"))
			m.logs = append(m.logs, ll)
			if len(m.logs) > 2000 {
				m.logs = m.logs[len(m.logs)-2000:]
			}
			// If user has scrolled up, keep their anchor point stable by
			// growing the offset along with the buffer.
			if m.logsOffset > 0 {
				m.logsOffset++
				if m.logsOffset >= len(m.logs) {
					m.logsOffset = len(m.logs) - 1
				}
			}
		}
		return m, m.recvLine()

	case LogStreamEndedMsg:
		if msg.Service == m.streamSvc {
			m.streamDropped = true
		}
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// ── mouse ──────────────────────────────────────────────────────────────

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.view != viewMain {
		return m, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.scrollLogs(+3)
		return m, nil
	case tea.MouseButtonWheelDown:
		m.scrollLogs(-3)
		return m, nil
	}
	// Only react to fresh clicks of the left button.
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return m, nil
	}
	lay := m.currentLayout()

	// Tab strip is the first content row of the logs pane (Y=3).
	if msg.Y == 3 {
		// Logs-content X-origin depends on fullscreen vs split mode.
		var logsStartX int
		if m.fullscreen {
			logsStartX = 3 // outer-pad(1) + gutter(1) + inner-border(1)
		} else if msg.X >= 4+NavW {
			logsStartX = 4 + NavW
		} else {
			// Click landed on the nav-side filter row, fall through to nav click.
			logsStartX = -1
		}
		if logsStartX >= 0 {
			if tab := tabAtX(msg.X-logsStartX, m.activeTab); tab != "" {
				m.activeTab = tab
				m.focused = "logs"
				return m, m.maybeFetchActive()
			}
		}
	}

	// Nav pane occupies columns [3, 3+NavW). Logs covers the rest.
	if !m.fullscreen && msg.X < 3+NavW {
		return m.clickNav(msg.X, msg.Y, lay)
	}
	m.focused = "logs"
	return m, nil
}

// tabAtX maps a logs-pane-relative X to a tab id, or "" if outside any tab.
// Mirrors the math in logsTabsRow().
func tabAtX(relX int, activeTab string) string {
	if relX < 1 {
		return "" // leading space
	}
	relX -= 1
	for i, t := range Tabs {
		var w int
		if t.ID == activeTab {
			w = 4 + runeLen(t.Label) // "[ Label ]"
		} else {
			w = runeLen(t.Label)
			if t.Stub {
				w++ // "·" suffix
			}
		}
		if relX < w {
			return t.ID
		}
		relX -= w
		if i < len(Tabs)-1 {
			if relX < 2 {
				return ""
			}
			relX -= 2
		}
	}
	return ""
}

func (m *Model) scrollLogs(delta int) {
	maxOffset := len(m.logs) - 1
	if maxOffset < 0 {
		maxOffset = 0
	}
	m.logsOffset += delta
	if m.logsOffset < 0 {
		m.logsOffset = 0
	}
	if m.logsOffset > maxOffset {
		m.logsOffset = maxOffset
	}
}

func (m Model) currentLayout() Layout {
	if m.width > 0 && m.height > 0 {
		return NewLayout(m.width, m.height)
	}
	return m.layout
}

// clickNav resolves a nav-pane click. y=3 is the filter row, y=4 a blank,
// then project headers + service rows. Click on a service row selects it.
func (m Model) clickNav(x, y int, lay Layout) (tea.Model, tea.Cmd) {
	_ = x
	contentRows := lay.TotalH - 7
	if contentRows < 4 {
		contentRows = 4
	}
	rowIdx := y - 3
	if rowIdx < 0 || rowIdx >= contentRows {
		return m, nil
	}
	m.focused = "nav"
	if rowIdx < 2 {
		return m, nil // filter row or its blank padding
	}
	cursorAt := 2
	q := strings.ToLower(m.navQuery)
	for _, r := range m.navRows {
		if cursorAt > rowIdx {
			break
		}
		if r.IsHeader {
			if cursorAt == rowIdx {
				return m, nil
			}
			cursorAt++
			continue
		}
		if q != "" {
			if !strings.Contains(strings.ToLower(r.Path), q) &&
				!strings.Contains(strings.ToLower(r.Service), q) {
				continue
			}
		}
		if cursorAt == rowIdx {
			for j, k := range m.selectableKeys {
				if k == r.Key {
					m.selectedIdx = j
					m.focused = "logs"
					return m, m.queueStreamForSelection()
				}
			}
			return m, nil
		}
		cursorAt++
	}
	return m, nil
}

func (m Model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit
	if k.Type == tea.KeyCtrlC {
		m.stopStream()
		return m, tea.Quit
	}

	// First-run form
	if m.view == viewFirstRun {
		return m.handleFirstRunKey(k)
	}

	// Theme picker overlay
	if m.showPicker {
		return m.handlePickerKey(k)
	}

	// Help overlay
	if m.showHelp {
		switch k.String() {
		case "?", "esc", "q":
			m.showHelp = false
		}
		return m, nil
	}

	// Telescope overlay capture
	if m.telescope.Open {
		return m.handleTelescopeKey(k)
	}

	// Search overlay capture
	if m.searchOpen && m.focused == "logs" {
		return m.handleSearchKey(k)
	}

	// Nav-filter capture
	if m.navFiltering {
		return m.handleNavFilterKey(k)
	}

	switch k.String() {
	case "q":
		m.stopStream()
		return m, tea.Quit
	case "?":
		m.showHelp = true
		return m, nil
	case "t":
		m.showPicker = true
		m.pickerIdx = themeIndex(m.cfg.Theme)
		return m, nil
	case "o", " ":
		// Telescope picker (replaces the old sidebar).
		m.openTelescope()
		return m, nil
	case "left":
		return m.cycleTab(-1)
	case "right":
		return m.cycleTab(+1)
	case "z":
		m.fullscreen = !m.fullscreen
		if m.fullscreen {
			m.focused = "logs"
		}
		return m, nil
	case "w":
		m.wrap = !m.wrap
		if m.wrap {
			m.notice = "Line wrap: on"
		} else {
			m.notice = "Line wrap: off"
		}
		return m, dismissNotice()
	case "tab":
		if m.focused == "nav" {
			m.focused = "logs"
		} else {
			m.focused = "nav"
		}
		return m, nil
	case "shift+tab":
		if m.focused == "nav" {
			m.focused = "logs"
		} else {
			m.focused = "nav"
		}
		return m, nil
	case "y", "Y":
		// Yank works on any tab — copies whatever is currently displayed.
		return m.yankLogs()
	case "r":
		if m.view == viewErrorUnreachable || m.view == viewErrorEmpty || m.streamDropped {
			m.view = viewConnecting
			return m, m.loadProjects()
		}
		// On data tabs, r refreshes the underlying data for the current resource.
		if row, ok := m.currentRow(); ok && row.UUID != "" {
			switch m.activeTab {
			case "deploys":
				delete(m.deploys, row.UUID)
				return m, m.maybeFetchDeploys()
			case "build":
				delete(m.builds, row.UUID)
				return m, m.maybeFetchBuild()
			case "config":
				delete(m.configs, row.UUID)
				return m, m.maybeFetchConfig()
			case "env":
				delete(m.envs, row.UUID)
				return m, m.maybeFetchEnv()
			}
		}
	}

	// Tab numeric jump
	if len(k.String()) == 1 {
		switch k.String() {
		case "1":
			if m.showLvlBar && m.focused == "logs" {
				m.levels["err"] = !m.levels["err"]
				return m, nil
			}
			m.activeTab = "logs"
			return m, nil
		case "2":
			if m.showLvlBar && m.focused == "logs" {
				m.levels["warn"] = !m.levels["warn"]
				return m, nil
			}
			m.activeTab = "build"
			return m, m.maybeFetchBuild()
		case "3":
			if m.showLvlBar && m.focused == "logs" {
				m.levels["info"] = !m.levels["info"]
				return m, nil
			}
			m.activeTab = "config"
			return m, m.maybeFetchConfig()
		case "4":
			if m.showLvlBar && m.focused == "logs" {
				m.levels["debug"] = !m.levels["debug"]
				return m, nil
			}
			m.activeTab = "env"
			return m, m.maybeFetchEnv()
		case "5":
			m.activeTab = "deploys"
			return m, m.maybeFetchDeploys()
		}
	}

	if m.focused == "nav" {
		return m.handleNavKey(k)
	}
	return m.handleLogsKey(k)
}

// ── telescope picker ───────────────────────────────────────────────────

// openTelescope shows the picker overlay seeded with the current selection.
func (m *Model) openTelescope() {
	sources := m.selectableServiceRows()
	m.telescope.Open = true
	m.telescope.Filter = ""
	// Position cursor on current selection if possible.
	cur := m.selectedKey()
	m.telescope.Cursor = 0
	for i, r := range sources {
		if r.Key == cur {
			m.telescope.Cursor = i
			break
		}
	}
	m.telescope.Recompute(sources)
}

func (m *Model) selectableServiceRows() []ServiceRow {
	out := make([]ServiceRow, 0, len(m.navRows))
	for _, r := range m.navRows {
		if !r.IsHeader {
			out = append(out, r)
		}
	}
	return out
}

func (m Model) handleTelescopeKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	sources := m.selectableServiceRows()
	switch k.String() {
	case "esc":
		m.telescope.Open = false
		m.telescope.Filter = ""
		return m, nil
	case "enter":
		if row, ok := m.telescope.SelectedRow(sources); ok {
			// find this row in selectableKeys + select it
			for j, k := range m.selectableKeys {
				if k == row.Key {
					m.selectedIdx = j
					m.telescope.Open = false
					m.telescope.Filter = ""
					m.focused = "logs"
					m.persistLastService(row.Service)
					return m, m.queueStreamForSelection()
				}
			}
		}
		m.telescope.Open = false
		return m, nil
	case "up", "ctrl+k", "ctrl+p":
		if m.telescope.Cursor > 0 {
			m.telescope.Cursor--
		}
		return m, nil
	case "down", "ctrl+j", "ctrl+n":
		if m.telescope.Cursor < len(m.telescope.Filtered)-1 {
			m.telescope.Cursor++
		}
		return m, nil
	case "backspace":
		if len(m.telescope.Filter) > 0 {
			m.telescope.Filter = m.telescope.Filter[:len(m.telescope.Filter)-1]
			m.telescope.Recompute(sources)
		}
		return m, nil
	}
	if k.Type == tea.KeyRunes {
		m.telescope.Filter += string(k.Runes)
		m.telescope.Recompute(sources)
	}
	return m, nil
}

// cycleTab shifts activeTab by delta (left/right keys on desktop, also useful
// on mobile where numeric jumps + arrows are the primary tab nav).
func (m Model) cycleTab(delta int) (tea.Model, tea.Cmd) {
	idx, _ := TabByID(m.activeTab)
	idx = (idx + delta + len(Tabs)) % len(Tabs)
	m.activeTab = Tabs[idx].ID
	return m, m.maybeFetchActive()
}

// persistLastService writes the chosen service name into .logify so the next
// launch re-opens it. Best-effort; failures are silent.
func (m *Model) persistLastService(name string) {
	if m.cfgPath == "" {
		return
	}
	bpath, _ := bindingFind(m.workingDir())
	if bpath == "" {
		return
	}
	f, err := bindingLoad(bpath)
	if err != nil || !f.IsBound() {
		return
	}
	if f.LastService == name {
		return
	}
	f.LastService = name
	_ = bindingSave(bpath, f)
}

func themeIndex(id string) int {
	for i, n := range theme.Order {
		if n == id {
			return i
		}
	}
	return 0
}

// loadLastService reads the .logify file (if present) and returns the
// remembered service name. Empty string when not set / not bound.
func (m *Model) loadLastService() string {
	bpath, _ := bindingFind(m.workingDir())
	if bpath == "" {
		return ""
	}
	f, err := bindingLoad(bpath)
	if err != nil || !f.IsBound() {
		return ""
	}
	return f.LastService
}

func rowMatchesService(navRows []ServiceRow, key, svcName string) bool {
	for _, r := range navRows {
		if !r.IsHeader && r.Key == key {
			return r.Service == svcName
		}
	}
	return false
}

func (m Model) handleNavKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	moved := false
	switch k.String() {
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
			moved = true
		}
	case "down", "j":
		if m.selectedIdx < len(m.selectableKeys)-1 {
			m.selectedIdx++
			moved = true
		}
	case "g":
		if m.selectedIdx != 0 {
			m.selectedIdx = 0
			moved = true
		}
	case "G":
		last := len(m.selectableKeys) - 1
		if m.selectedIdx != last {
			m.selectedIdx = last
			moved = true
		}
	case "/":
		m.navFiltering = true
		return m, nil
	case "enter":
		// Enter focuses the logs pane; the stream is already running thanks to
		// auto-follow on nav move.
		if _, ok := m.currentRow(); !ok {
			return m, nil
		}
		m.focused = "logs"
		return m, nil
	}
	if moved {
		return m, m.queueStreamForSelection()
	}
	return m, nil
}

func (m Model) handleLogsKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "/":
		m.searchOpen = true
		return m, nil
	case "f":
		m.showLvlBar = !m.showLvlBar
		return m, nil
	case " ":
		m.paused = !m.paused
		if !m.paused && m.queued > 0 {
			m.queued = 0
		}
		return m, nil
	case "c":
		m.logs = nil
		m.queued = 0
		m.logsOffset = 0
		return m, nil
	case "up", "k":
		m.scrollLogs(+1)
		return m, nil
	case "down", "j":
		m.scrollLogs(-1)
		return m, nil
	case "pgup", "ctrl+b":
		m.scrollLogs(+15)
		return m, nil
	case "pgdown", "ctrl+f":
		m.scrollLogs(-15)
		return m, nil
	case "g":
		m.scrollLogs(+len(m.logs))
		return m, nil
	case "G":
		m.logsOffset = 0
		return m, nil
	}
	return m, nil
}

// yankLogs copies whatever the active tab is showing to the terminal's
// clipboard via OSC 52. Output via stderr (stdout is owned by bubble tea's
// alt-screen renderer; direct writes get mangled, but stderr passes through
// to the terminal which interprets OSC 52 from either stream).
//
// Works in: kitty, wezterm, iTerm2, ghostty, foot, recent xterm; alacritty if
// `selection.save_to_clipboard = true` + OSC 52 allowed; tmux 3.3+.
func (m Model) yankLogs() (tea.Model, tea.Cmd) {
	var (
		body  string
		label string
	)
	switch m.activeTab {
	case "logs":
		body, label = m.runtimeLogsAsText()
	case "build":
		body, label = m.buildLogAsText()
	case "config":
		body, label = m.configAsText()
	case "env":
		body, label = m.envAsText()
	case "deploys":
		body, label = m.deploysAsText()
	}
	if body == "" {
		m.notice = "nothing to copy on this tab"
		return m, dismissNotice()
	}
	payload := base64.StdEncoding.EncodeToString([]byte(body))
	// ESC ] 52 ; c ; <base64> BEL — stderr so the alt-screen renderer
	// doesn't fight us.
	_, _ = os.Stderr.WriteString("\x1b]52;c;" + payload + "\x07")
	m.notice = "copied " + label + " to clipboard"
	return m, dismissNotice()
}

func (m Model) runtimeLogsAsText() (string, string) {
	if len(m.logs) == 0 {
		return "", ""
	}
	var b strings.Builder
	for _, l := range m.logs {
		if l.TS != "" {
			b.WriteString(l.TS)
			b.WriteByte(' ')
		}
		if l.Level != "" {
			b.WriteString(l.Level)
			b.WriteByte(' ')
		}
		b.WriteString(l.Msg)
		b.WriteByte('\n')
	}
	return b.String(), fmt.Sprintf("%d log lines", len(m.logs))
}

func (m Model) buildLogAsText() (string, string) {
	row, ok := m.currentRow()
	if !ok {
		return "", ""
	}
	bl := m.builds[row.UUID]
	if len(bl.Lines) == 0 {
		return "", ""
	}
	var b strings.Builder
	if bl.DeploymentUUID != "" {
		fmt.Fprintf(&b, "deployment: %s\nstatus: %s\ncommit: %s\n\n", bl.DeploymentUUID, bl.Status, bl.Commit)
	}
	for _, ln := range bl.Lines {
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	return b.String(), fmt.Sprintf("%d build lines", len(bl.Lines))
}

func (m Model) configAsText() (string, string) {
	row, ok := m.currentRow()
	if !ok {
		return "", ""
	}
	c := m.configs[row.UUID]
	if c.Name == "" {
		return "", ""
	}
	var b strings.Builder
	add := func(k, v string) {
		if v != "" {
			fmt.Fprintf(&b, "%-18s %s\n", k+":", v)
		}
	}
	add("name", c.Name)
	add("status", c.Status)
	add("fqdn", c.FQDN)
	add("git", c.GitRepository)
	add("branch", c.GitBranch)
	add("commit", c.GitCommit)
	add("build_pack", c.BuildPack)
	add("install", c.InstallCommand)
	add("build", c.BuildCommand)
	add("start", c.StartCommand)
	add("ports", c.PortsExposed)
	if c.DockerComposeRaw != "" {
		b.WriteString("\n# docker-compose\n")
		b.WriteString(c.DockerComposeRaw)
		if !strings.HasSuffix(c.DockerComposeRaw, "\n") {
			b.WriteByte('\n')
		}
	}
	return b.String(), "config"
}

func (m Model) envAsText() (string, string) {
	row, ok := m.currentRow()
	if !ok {
		return "", ""
	}
	vars := m.envs[row.UUID]
	if len(vars) == 0 {
		return "", ""
	}
	var b strings.Builder
	for _, v := range vars {
		b.WriteString(v.Key)
		b.WriteByte('\n')
	}
	return b.String(), fmt.Sprintf("%d env keys", len(vars))
}

func (m Model) deploysAsText() (string, string) {
	row, ok := m.currentRow()
	if !ok {
		return "", ""
	}
	deps := m.deploys[row.UUID]
	if len(deps) == 0 {
		return "", ""
	}
	var b strings.Builder
	for _, d := range deps {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", d.UUID, d.Status, d.Commit, d.CommitMessage)
	}
	return b.String(), fmt.Sprintf("%d deploys", len(deps))
}

func dismissNotice() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return ClipboardCopiedMsg{} })
}

func (m Model) handleSearchKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.Type {
	case tea.KeyEsc:
		m.searchOpen = false
		m.searchQuery = ""
		return m, nil
	case tea.KeyEnter:
		m.searchOpen = false
		return m, nil
	case tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
		return m, nil
	}
	if k.Type == tea.KeyRunes {
		m.searchQuery += string(k.Runes)
	} else if len(k.String()) == 1 {
		m.searchQuery += k.String()
	}
	return m, nil
}

func (m Model) handleNavFilterKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.Type {
	case tea.KeyEsc:
		m.navFiltering = false
		m.navQuery = ""
		return m, nil
	case tea.KeyEnter:
		m.navFiltering = false
		return m, nil
	case tea.KeyBackspace:
		if len(m.navQuery) > 0 {
			m.navQuery = m.navQuery[:len(m.navQuery)-1]
		}
		return m, nil
	}
	if k.Type == tea.KeyRunes {
		m.navQuery += string(k.Runes)
	}
	return m, nil
}

func (m Model) handlePickerKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "esc", "q":
		m.showPicker = false
	case "up", "k":
		if m.pickerIdx > 0 {
			m.pickerIdx--
		}
		m.applyPicker()
	case "down", "j":
		if m.pickerIdx < len(theme.Order)-1 {
			m.pickerIdx++
		}
		m.applyPicker()
	case "enter":
		m.applyPicker()
		m.cfg.Theme = theme.Order[m.pickerIdx]
		_, _ = config.Save(m.cfg)
		m.showPicker = false
	}
	return m, nil
}

func (m *Model) applyPicker() {
	m.theme = theme.Get(theme.Order[m.pickerIdx])
}

func (m Model) handleFirstRunKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "tab", "down":
		m.frFocus = (m.frFocus + 1) % 4
		return m, nil
	case "shift+tab", "up":
		m.frFocus = (m.frFocus + 3) % 4
		return m, nil
	case "enter":
		switch m.frFocus {
		case 0, 1:
			m.frFocus++
		case 2:
			// Save & continue
			m.cfg.URL = strings.TrimSpace(m.frURL)
			m.cfg.Token = strings.TrimSpace(m.frToken)
			if m.cfg.URL == "" || m.cfg.Token == "" {
				m.frError = "URL and API key are required."
				return m, nil
			}
			if _, err := config.Save(m.cfg); err != nil {
				m.frError = err.Error()
				return m, nil
			}
			m.client = api.New(m.cfg.URL, m.cfg.Token)
			m.view = viewConnecting
			return m, m.loadProjects()
		case 3:
			return m, tea.Quit
		}
	case "esc":
		return m, tea.Quit
	}
	// Text editing for fields
	if m.frFocus > 1 {
		return m, nil
	}
	if k.Type == tea.KeyBackspace {
		if m.frFocus == 0 && len(m.frURL) > 0 {
			m.frURL = m.frURL[:len(m.frURL)-1]
		} else if m.frFocus == 1 && len(m.frToken) > 0 {
			m.frToken = m.frToken[:len(m.frToken)-1]
		}
		return m, nil
	}
	if k.Type == tea.KeyRunes {
		s := string(k.Runes)
		if m.frFocus == 0 {
			m.frURL += s
		} else {
			m.frToken += s
		}
	}
	return m, nil
}

func (m Model) View() string {
	// Auto-size to the current terminal if available.
	lay := m.layout
	if m.width > 0 && m.height > 0 {
		lay = NewLayout(m.width, m.height)
	}
	th := m.theme
	switch m.view {
	case viewFirstRun:
		return JoinRows(buildFirstRun(th, lay, m.frURL, m.frToken, m.frFocus, m.cfgPath), th.Bg)
	case viewConnecting:
		return JoinRows(buildConnecting(th, lay, m.cfg.URL, m.frame), th.Bg)
	case viewErrorUnreachable:
		return JoinRows(buildErrorUnreachable(th, lay, m.errorTarget, m.errorDetail, m.frame), th.Bg)
	case viewErrorAuth:
		return JoinRows(buildErrorAuth(th, lay, m.cfgPath, m.frame), th.Bg)
	case viewErrorEmpty:
		return JoinRows(buildErrorEmpty(th, lay, m.frame), th.Bg)
	}
	// Main
	logsState := LogsState{
		ActiveTab:       m.activeTab,
		Logs:            m.logs,
		Levels:          m.levels,
		ShowLevelFilter: m.showLvlBar,
		SearchOpen:      m.searchOpen,
		SearchQuery:     m.searchQuery,
		Paused:          m.paused,
		QueuedLogs:      m.queued,
		StreamDropped:   m.streamDropped,
		Wrap:            m.wrap,
		ScrollOffset:    m.logsOffset,
		Deploys:         m.buildDeploysState(),
		Build:           m.buildBuildState(),
		Config:          m.buildConfigState(),
		Env:             m.buildEnvState(),
	}
	conn := "connected"
	if m.streamDropped {
		conn = "connecting"
	}
	st := screenState{
		SelectedPath: m.selectedPath(),
		SelectedKey:  m.selectedKey(),
		Focused:      m.focused,
		Connection:   conn,
		TailSize:     m.cfg.Tail,
		NavRows:      m.navRows,
		NavQuery:     m.navQuery,
		LogsState:    logsState,
		Notice:       m.notice,
		Fullscreen:   m.fullscreen,
	}
	base := buildMain(th, lay, st)
	if m.telescope.Open {
		base = buildTelescopeOverlay(th, lay, base, m.telescope, m.selectableServiceRows())
	}
	if m.showHelp {
		base = buildHelpOverlay(th, lay, base)
	} else if m.showPicker {
		base = buildThemePicker(th, lay, base, theme.Order, m.pickerIdx)
	}
	return JoinRows(base, th.Bg)
}

