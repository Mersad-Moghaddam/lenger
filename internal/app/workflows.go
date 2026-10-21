package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Mersad-Moghaddam/lenger/internal/core/services"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
)

var (
	runRestarts = []string{"no", "unless-stopped", "always", "on-failure"}
	runLabels   = []string{"name", "ports", "env", "volumes", "restart", "network", "command"}
	execShells  = []string{"sh", "bash", "ash"}
)

type runForm struct {
	name, ports, env, volumes, network, command string
	restart                                     int
}

func (f *runForm) value(field int) string {
	switch field {
	case 0:
		return f.name
	case 1:
		return f.ports
	case 2:
		return f.env
	case 3:
		return f.volumes
	case 4:
		return runRestarts[max(0, min(f.restart, len(runRestarts)-1))]
	case 5:
		return f.network
	case 6:
		return f.command
	}
	return ""
}

func (f *runForm) set(field int, value string) {
	switch field {
	case 0:
		f.name = value
	case 1:
		f.ports = value
	case 2:
		f.env = value
	case 3:
		f.volumes = value
	case 5:
		f.network = value
	case 6:
		f.command = value
	}
}

func (f *runForm) spec(image string) (engine.RunSpec, error) {
	command, err := services.ParseCommandLine(f.command)
	if err != nil {
		return engine.RunSpec{}, err
	}
	return engine.RunSpec{
		Name: f.name, Image: image, Network: f.network,
		Restart: runRestarts[max(0, min(f.restart, len(runRestarts)-1))],
		Command: command, Ports: splitList(f.ports), Env: splitList(f.env), Volumes: splitList(f.volumes),
	}, nil
}

func splitList(value string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '\n' }) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (m *Model) openRun() tea.Cmd {
	items := m.items()
	if len(items) == 0 {
		return m.notify("Select an image first.")
	}
	name := items[m.cursor].Name()
	if name == "<none>:<none>" {
		return m.notify("Untagged images cannot be run by name. Tag it or pull a tagged image first.")
	}
	m.runImage = name
	m.runForm = runForm{restart: 1}
	m.runField = 0
	m.runErr = ""
	m.runBusy = false
	m.overlay = "run"
	m.input.Reset()
	m.input.Placeholder = "optional container name"
	return m.input.Focus()
}

func (m *Model) saveRunInput() {
	if m.runField == 4 {
		return
	}
	m.runForm.set(m.runField, strings.TrimSpace(m.input.Value()))
}

func (m *Model) switchRunField(delta int) tea.Cmd {
	m.saveRunInput()
	m.runField = (m.runField + delta + len(runLabels)) % len(runLabels)
	m.input.SetValue(m.runForm.value(m.runField))
	m.input.CursorEnd()
	m.input.Placeholder = runPlaceholder(m.runField)
	if m.runField == 4 {
		m.input.Blur()
		return nil
	}
	return m.input.Focus()
}

func runPlaceholder(field int) string {
	switch field {
	case 0:
		return "optional container name"
	case 1:
		return "8080:80, 127.0.0.1:5432:5432"
	case 2:
		return "KEY=value, KEY2=value"
	case 3:
		return "volume-name:/data, /host/path:/data:ro"
	case 5:
		return "bridge or a network name"
	case 6:
		return "sh -c \"echo hello\""
	}
	return ""
}

func (m *Model) runSpec() (engine.RunSpec, error) {
	m.saveRunInput()
	return m.runForm.spec(m.runImage)
}

func (m *Model) submitRun() tea.Cmd {
	if m.runBusy {
		return nil
	}
	spec, err := m.runSpec()
	if err != nil {
		m.runErr = err.Error()
		return nil
	}
	if _, err := services.NormalizeRunSpec(spec); err != nil {
		m.runErr = err.Error()
		return nil
	}
	m.runBusy = true
	m.runErr = ""
	ctx, service := m.ctx, m.service
	return func() tea.Msg {
		id, err := service.Run(ctx, spec)
		return runMsg{id: id, err: err}
	}
}

func (m *Model) containerIDs() []string {
	items := m.containers()
	ids := make([]string, 0, len(items))
	for _, c := range items {
		if m.selectedContain[c.ID] {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 && len(items) > 0 {
		ids = append(ids, items[m.containerCursor].ID)
	}
	return ids
}

func (m *Model) containerByID(id string) (engine.Container, bool) {
	for _, c := range m.store.Containers {
		if c.ID == id {
			return c, true
		}
	}
	return engine.Container{}, false
}

func (m *Model) triggerContainerAction(action engine.ContainerAction, confirm bool) tea.Cmd {
	ids := m.containerIDs()
	if len(ids) == 0 {
		return m.notify("No container selected.")
	}
	if action == engine.ActionPause {
		allPaused := true
		for _, id := range ids {
			if c, ok := m.containerByID(id); !ok || c.State != "paused" {
				allPaused = false
			}
		}
		if allPaused {
			action = engine.ActionUnpause
		}
	}
	if confirm {
		m.actionIDs = ids
		m.actionName = action
		m.overlay = "container-action"
		return nil
	}
	return m.runContainerAction(action, ids)
}

func (m *Model) runContainerAction(action engine.ContainerAction, ids []string) tea.Cmd {
	ctx, service := m.ctx, m.service
	return func() tea.Msg {
		return containerActionMsg{action: action, ids: ids, errs: service.ActContainers(ctx, ids, action)}
	}
}

func (m *Model) hasRunningContainers() bool {
	for _, c := range m.store.Containers {
		if c.Running() {
			return true
		}
	}
	return false
}

func (m *Model) statsTick() tea.Cmd {
	if m.statsTicking || m.tab != 2 || !m.hasRunningContainers() {
		return nil
	}
	m.statsTicking = true
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return statsTickMsg{} })
}

func (m *Model) loadStats() tea.Cmd {
	if m.statsLoading || m.tab != 2 {
		return nil
	}
	ids := make([]string, 0, len(m.store.Containers))
	for _, c := range m.store.Containers {
		if c.Running() {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	m.statsLoading = true
	ctx, service := m.ctx, m.service
	return func() tea.Msg {
		data := make(map[string]engine.ContainerStats, len(ids))
		var mu sync.Mutex
		var wg sync.WaitGroup
		for _, id := range ids {
			id := id
			wg.Add(1)
			go func() {
				defer wg.Done()
				if sample, err := service.ContainerStats(ctx, id); err == nil {
					mu.Lock()
					data[id] = sample
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		return statsMsg{data: data}
	}
}

func (m *Model) loadUsage() tea.Cmd {
	if m.usageLoading {
		return nil
	}
	m.usageLoading = true
	ctx, service := m.ctx, m.service
	return func() tea.Msg {
		usage, err := service.SystemUsage(ctx)
		return usageMsg{data: usage, err: err}
	}
}

func defaultCleanupSelection() map[services.CleanupKind]bool {
	return map[services.CleanupKind]bool{
		services.CleanupDanglingImages:    true,
		services.CleanupUnusedImages:      true,
		services.CleanupStoppedContainers: true,
		services.CleanupBuildCache:        true,
	}
}

func (m *Model) openCleanup() tea.Cmd {
	m.overlay = "cleanup"
	m.cleanupLoaded = false
	m.cleanupLoading = false
	m.cleanupErr = nil
	m.cleanupPlan = services.CleanupPlan{}
	m.cleanupCursor = 0
	m.cleanupSelected = defaultCleanupSelection()
	return m.loadCleanup()
}

func (m *Model) loadCleanup() tea.Cmd {
	if m.cleanupLoading {
		return nil
	}
	m.cleanupLoading = true
	m.cleanupErr = nil
	ctx, service := m.ctx, m.service
	return func() tea.Msg {
		plan, err := service.CleanupPreview(ctx)
		return cleanupPlanMsg{plan: plan, err: err}
	}
}

func (m *Model) cleanupKinds() []services.CleanupKind {
	var kinds []services.CleanupKind
	for _, kind := range services.CleanupKinds() {
		if m.cleanupSelected[kind] {
			kinds = append(kinds, kind)
		}
	}
	return kinds
}

func (m *Model) selectedCleanupTotal() int64 {
	var total int64
	for _, group := range m.cleanupPlan.Groups {
		if m.cleanupSelected[group.Kind] {
			total += group.Size
		}
	}
	return total
}

func (m *Model) executeCleanup() tea.Cmd {
	if m.cleanupLoading {
		return nil
	}
	kinds := m.cleanupKinds()
	if len(kinds) == 0 {
		return m.notify("Select at least one cleanup category.")
	}
	confirmation := ""
	if m.cleanupSelected[services.CleanupUnusedVolumes] {
		confirmation = m.input.Value()
	}
	m.cleanupLoading = true
	ctx, service := m.ctx, m.service
	return func() tea.Msg {
		result, err := service.Cleanup(ctx, kinds, confirmation)
		return cleanupDoneMsg{result: result, err: err}
	}
}

func (m *Model) openRemoveVolume() tea.Cmd {
	items := m.volumes()
	if len(items) == 0 {
		return m.notify("No volume selected.")
	}
	v := items[m.volumeCursor]
	if v.RefCount > 0 || len(v.UsedBy) > 0 {
		return m.notify("Volume is in use; remove its container first.")
	}
	m.removeVolume = v.Name
	m.overlay = "remove-volume"
	m.input.Reset()
	m.input.Placeholder = "volumes"
	return m.input.Focus()
}

func (m *Model) confirmRemoveVolume() tea.Cmd {
	if strings.TrimSpace(m.input.Value()) != "volumes" {
		return m.notify(`Type "volumes" to confirm.`)
	}
	name := m.removeVolume
	ctx, service := m.ctx, m.service
	m.closeOverlay()
	return func() tea.Msg {
		return volumeRemovedMsg{name: name, err: service.RemoveVolume(ctx, name)}
	}
}

func (m *Model) openLogs(id string) tea.Cmd {
	m.stopLogs()
	m.logs = nil
	m.logsID = id
	m.logsFollow = true
	m.logsPaused = false
	m.logsFilter = ""
	m.logsNotice = ""
	m.overlay = "logs"
	return m.streamLogs(true, false)
}

func (m *Model) stopLogs() {
	if m.logsCancel != nil {
		m.logsCancel()
		m.logsCancel = nil
	}
}

func (m *Model) streamLogs(follow, clear bool) tea.Cmd {
	m.stopLogs()
	if clear {
		m.logs = nil
	}
	m.logsFollow = follow
	m.logsPaused = false
	m.logsNotice = ""
	m.logsGen++
	gen := m.logsGen
	ctx, cancel := context.WithCancel(m.ctx)
	m.logsCancel = cancel
	ch := make(chan tea.Msg, 256)
	m.logCh = ch
	id, service := m.logsID, m.service
	worker := func() tea.Msg {
		err := service.Logs(ctx, id, engine.LogOptions{Follow: follow, Tail: 500}, func(entry engine.LogEntry) error {
			select {
			case ch <- logLineMsg{gen: gen, entry: entry}:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		select {
		case ch <- logEndMsg{gen: gen, err: err}:
		default:
		}
		return nil
	}
	return tea.Batch(worker, waitLog(ch, gen, ctx))
}

func (m *Model) pauseLogs() tea.Cmd {
	if m.logsPaused {
		m.logsNotice = "Resuming from the last 500 lines…"
		return m.streamLogs(true, true)
	}
	m.stopLogs()
	m.logsPaused = true
	m.logsFollow = false
	m.logsNotice = "Log stream paused. Press p to resume."
	return nil
}

func (m *Model) visibleLogs() []engine.LogEntry {
	if m.logsFilter == "" {
		return m.logs
	}
	re, err := regexp.Compile("(?i)" + m.logsFilter)
	if err != nil {
		return m.logs
	}
	out := make([]engine.LogEntry, 0, len(m.logs))
	for _, entry := range m.logs {
		if re.MatchString(entry.Text) || re.MatchString(entry.Stream) {
			out = append(out, entry)
		}
	}
	return out
}

func (m *Model) logLines(limit int) []string {
	entries := m.visibleLogs()
	start := max(0, len(entries)-limit)
	lines := make([]string, 0, len(entries)-start)
	for _, entry := range entries[start:] {
		prefix := " "
		if entry.Stream == "stderr" {
			prefix = "!"
		}
		text := style.Clean(entry.Text)
		if m.logsTimestamps && !entry.At.IsZero() {
			text = entry.At.Format("15:04:05.000") + " " + text
		}
		lines = append(lines, prefix+" "+text)
	}
	return lines
}

func (m *Model) copyLogs() tea.Cmd {
	lines := m.logLines(200)
	if len(lines) == 0 {
		return m.notify("No log lines to copy.")
	}
	return tea.Batch(tea.SetClipboard(strings.Join(lines, "\n")), m.notify("Copied the last log lines."))
}

func (m *Model) saveLogs() tea.Cmd {
	lines := m.logLines(50_000)
	if len(lines) == 0 {
		return m.notify("No log lines to save.")
	}
	content := strings.Join(lines, "\n") + "\n"
	id := m.logsID
	return func() tea.Msg {
		dir := stateDir()
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return logSavedMsg{err: err}
		}
		path := filepath.Join(dir, fmt.Sprintf("logs-%s-%s.log", engine.ShortID(id), time.Now().Format("20060102-150405")))
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			return logSavedMsg{err: err}
		}
		return logSavedMsg{path: path}
	}
}

func stateDir() string {
	if value := os.Getenv("XDG_STATE_HOME"); value != "" {
		return filepath.Join(value, "lenger")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "lenger")
	}
	return filepath.Join(home, ".local", "state", "lenger")
}

func (m *Model) openExec() tea.Cmd {
	items := m.containers()
	if len(items) == 0 {
		return m.notify("No container selected.")
	}
	c := items[m.containerCursor]
	if c.State != "running" {
		return m.notify("Exec requires a running container.")
	}
	m.execContainer = c.ID
	m.execShell = 0
	m.overlay = "exec"
	return nil
}

func (m *Model) runExec() tea.Cmd {
	container := m.execContainer
	shell := execShells[max(0, min(m.execShell, len(execShells)-1))]
	m.closeOverlay()
	cmd := exec.Command("docker", "exec", "-it", container, shell)
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return execDoneMsg{err: err} })
}

func (m *Model) copyEquivalent() tea.Cmd {
	var command string
	switch m.tab {
	case 1:
		if items := m.items(); len(items) > 0 {
			command = "docker image inspect " + items[m.cursor].ID
		}
	case 2:
		if ids := m.containerIDs(); len(ids) > 0 {
			command = "docker inspect " + ids[0]
		}
	case 3:
		if items := m.projects(); len(items) > 0 {
			command = "docker compose -p " + items[m.projectCursor].Name + " ps"
		}
	case 4:
		if items := m.volumes(); len(items) > 0 {
			command = "docker volume inspect " + items[m.volumeCursor].Name
		}
	case 5:
		if items := m.networks(); len(items) > 0 {
			command = "docker network inspect " + items[m.networkCursor].ID
		}
	case 6:
		command = "docker system df; docker info"
	}
	if command == "" {
		return m.notify("Nothing to copy on this screen.")
	}
	return tea.Batch(tea.SetClipboard(command), m.notify("Copied: "+command))
}

func (m *Model) loadDaemonInfo() tea.Cmd {
	if m.daemonLoading {
		return nil
	}
	m.daemonLoading = true
	ctx, service := m.ctx, m.service
	return func() tea.Msg {
		info, err := service.DaemonInfo(ctx)
		return daemonInfoMsg{data: info, err: err}
	}
}

func (m *Model) openComposeAction(action engine.ComposeAction) tea.Cmd {
	items := m.projects()
	if len(items) == 0 {
		return m.notify("No project selected.")
	}
	p := items[m.projectCursor]
	m.composeProject = p.Name
	m.composeAction = action
	m.composeVolumes = false
	m.overlay = "compose-action"
	return nil
}

func (m *Model) confirmComposeAction() tea.Cmd {
	action := m.composeAction
	project := m.composeProject
	removeVols := m.composeVolumes
	m.overlay = ""
	return m.runCompose(action, project, removeVols)
}

func (m *Model) runCompose(action engine.ComposeAction, projectName string, removeVolumes bool) tea.Cmd {
	var opts engine.ComposeOptions
	opts.ProjectName = projectName
	opts.RemoveVolumes = removeVolumes

	for _, p := range m.projects() {
		if p.Name == projectName {
			opts.ProjectDir = p.WorkingDir
			opts.ConfigFiles = p.ConfigFiles
			break
		}
	}

	ch := make(chan tea.Msg, 128)
	m.composeCh = ch
	m.composeStream = []string{fmt.Sprintf("Running compose %s on %s...", action, projectName)}
	ctx, service := m.ctx, m.service

	go func() {
		defer close(ch)
		err := service.Compose(ctx, action, opts, func(line string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case ch <- composeLineMsg{line: line}:
				return nil
			}
		})
		ch <- composeDoneMsg{action: action, err: err}
	}()

	return m.waitComposeCh()
}

func (m *Model) waitComposeCh() tea.Cmd {
	ch := m.composeCh
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *Model) openCreateNetwork() tea.Cmd {
	m.overlay = "create-network"
	m.createNetDriver = "bridge"
	m.input.Reset()
	m.input.Placeholder = "network-name"
	return m.input.Focus()
}

func (m *Model) confirmCreateNetwork() tea.Cmd {
	name := strings.TrimSpace(m.input.Value())
	if name == "" {
		return m.notify("Network name cannot be empty.")
	}
	driver := m.createNetDriver
	if driver == "" {
		driver = "bridge"
	}
	ctx, service := m.ctx, m.service
	m.input.Blur()
	m.overlay = ""
	return func() tea.Msg {
		id, err := service.CreateNetwork(ctx, name, driver)
		return networkCreatedMsg{id: id, err: err}
	}
}

func (m *Model) openRemoveNetwork() tea.Cmd {
	items := m.networks()
	if len(items) == 0 {
		return m.notify("No network selected.")
	}
	net := items[m.networkCursor]
	if len(net.Containers) > 0 {
		return m.notify(fmt.Sprintf("Network %s has active containers attached; disconnect them first.", net.Name))
	}
	m.removeNetwork = net.ID
	m.overlay = "remove-network"
	return nil
}

func (m *Model) confirmRemoveNetwork() tea.Cmd {
	id := m.removeNetwork
	ctx, service := m.ctx, m.service
	m.overlay = ""
	return func() tea.Msg {
		err := service.RemoveNetwork(ctx, id)
		return networkRemovedMsg{id: id, err: err}
	}
}

func (m *Model) cleanupSummary() string {
	groups := m.cleanupPlan.Groups
	parts := make([]string, 0, len(groups))
	for _, group := range groups {
		if m.cleanupSelected[group.Kind] {
			parts = append(parts, fmt.Sprintf("%s %d", group.Kind, group.Count))
		}
	}
	if len(parts) == 0 {
		return "No categories selected."
	}
	return strings.Join(parts, " · ") + " · " + components.Bytes(m.selectedCleanupTotal())
}
