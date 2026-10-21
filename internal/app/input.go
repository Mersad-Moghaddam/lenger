package app

import (
	tea "charm.land/bubbletea/v2"
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/core/services"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/components"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/tokens"
	"github.com/sahilm/fuzzy"
	"regexp"
	"strconv"
	"strings"
)

type action struct{ name, key, command string }

func actions() []action {
	return []action{
		{"Dashboard", "1", ""},
		{"Images", "2", "docker image ls"},
		{"Containers", "3", "docker ps -a"},
		{"Projects", "4", "docker compose ps"},
		{"Volumes", "5", "docker volume ls"},
		{"Networks", "6", "docker network ls"},
		{"System info", "7", "docker system df; docker info"},
		{"Pull image", "p", "docker pull <reference>"},
		{"Run from image", "run-image", "docker run -d <image>"},
		{"Start container", "x", "docker start <container>"},
		{"Stop container", "s", "docker stop <container>"},
		{"Restart container", "r", "docker restart <container>"},
		{"Logs", "l", "docker logs -f <container>"},
		{"Exec shell", "e", "docker exec -it <container> sh"},
		{"Inspect", "i", "docker inspect <resource>"},
		{"Remove", "d", "docker rm <container> / docker image rm <image>"},
		{"Compose Up", "u", "docker compose up -d"},
		{"Compose Down", "compose-down", "docker compose down"},
		{"Compose Restart", "compose-restart", "docker compose restart"},
		{"Create Network", "create-net", "docker network create"},
		{"Smart cleanup", "cleanup", "docker system df"},
		{"Registry search", "S", "docker search <term>"},
		{"Check image update", "U", "check remote manifest digest"},
		{"Tag image", "t", "docker tag <source> <target>"},
		{"Push image", "P", "docker push <ref>"},
		{"Export tarball", "E", "docker save -o <file> <image>"},
		{"Import tarball", "I", "docker load -i <file>"},
		{"Scan vulnerabilities", "V", "trivy / grype / docker scout"},
		{"Daemon mirrors diff", "M", "configure /etc/docker/daemon.json registry-mirrors"},
		{"Filter list", "/", "filter current resources"},
		{"Refresh resources", "R", "docker ps -a; docker image ls; docker volume ls"},
		{"Job history", "J", ""},
		{"Help", "?", ""},
	}
}
func (m *Model) paletteActions() []action {
	all := actions()
	for _, image := range m.items() {
		all = append(all, action{"Image: " + image.Name(), "image:" + image.ID, "docker image inspect " + image.ID})
	}
	for _, container := range m.containers() {
		all = append(all, action{"Container: " + container.Name, "container:" + container.ID, "docker inspect " + container.ID})
	}
	for _, project := range m.projects() {
		all = append(all, action{"Project: " + project.Name, "project:" + project.Name, "docker compose -p " + project.Name + " ps"})
	}
	for _, network := range m.networks() {
		all = append(all, action{"Network: " + network.Name, "network:" + network.ID, "docker network inspect " + network.ID})
	}
	if m.input.Value() == "" {
		return all
	}
	names := make([]string, len(all))
	for n, a := range all {
		names[n] = a.name
	}
	matches := fuzzy.Find(m.input.Value(), names)
	result := make([]action, 0, len(matches))
	for _, match := range matches {
		result = append(result, all[match.Index])
	}
	return result
}
func (m *Model) openInput(kind string) tea.Cmd {
	m.overlay = kind
	m.input.Reset()
	m.input.Placeholder = ""
	if kind == "pull-input" {
		m.input.Placeholder = "redis:7.4"
	}
	if kind == "filter" {
		m.input.SetValue(m.activeFilter())
	}
	if kind == "log-filter" {
		m.input.SetValue(m.logsFilter)
		m.input.Placeholder = "error|timeout|regex"
	}
	m.paletteCursor = 0
	return m.input.Focus()
}
func (m *Model) closeOverlay() {
	if m.overlay == "logs" {
		m.stopLogs()
	}
	m.overlay = ""
	m.input.Blur()
}

func (m *Model) key(msg tea.KeyPressMsg) tea.Cmd {
	k := msg.String()
	if m.overlay != "" {
		return m.overlayKey(msg)
	}
	for action, binding := range m.config.Keys {
		if binding == k {
			switch action {
			case "pull":
				k = "p"
			case "inspect":
				k = "i"
			case "delete":
				k = "d"
			case "filter":
				k = "/"
			case "palette":
				k = ":"
			case "help":
				k = "?"
			case "quit":
				k = "q"
			}
		}
	}
	if strings.HasPrefix(k, "image:") {
		id := strings.TrimPrefix(k, "image:")
		m.tab = 1
		for n, i := range m.items() {
			if i.ID == id {
				m.cursor = n
				return m.inspect()
			}
		}
		return nil
	}
	if strings.HasPrefix(k, "container:") {
		id := strings.TrimPrefix(k, "container:")
		m.tab = 2
		for n, c := range m.containers() {
			if c.ID == id {
				m.containerCursor = n
				return tea.Batch(m.inspectContainer(), m.loadStats())
			}
		}
		return nil
	}
	if strings.HasPrefix(k, "project:") {
		name := strings.TrimPrefix(k, "project:")
		m.tab = 3
		for n, p := range m.projects() {
			if p.Name == name {
				m.projectCursor = n
				return nil
			}
		}
		return nil
	}
	if strings.HasPrefix(k, "network:") {
		id := strings.TrimPrefix(k, "network:")
		m.tab = 5
		for n, net := range m.networks() {
			if net.ID == id || net.Name == id {
				m.networkCursor = n
				return nil
			}
		}
		return nil
	}
	if number, err := strconv.Atoi(k); err == nil && number >= 1 && number <= 7 {
		m.tab = number - 1
		m.detail = false
		m.containerDetail = false
		if m.tab == 6 {
			return tea.Batch(m.loadUsage(), m.loadDaemonInfo())
		}
		if m.tab == 2 {
			return m.loadStats()
		}
		return nil
	}
	switch k {
	case "q":
		return m.quit()
	case "esc":
		if m.tab == 2 && m.containerDetail {
			m.containerDetail = false
		} else if m.tab == 1 && m.detail {
			m.detail = false
		} else if m.activeFilter() != "" {
			m.setActiveFilter("")
			m.clamp()
		}
		return nil
	case "?":
		m.overlay = "help"
	case ":", "ctrl+p":
		return m.openInput("palette")
	case "p":
		if m.tab == 2 {
			return m.triggerContainerAction(engine.ActionPause, false)
		}
		return m.openInput("pull-input")
	case "J":
		m.overlay = "jobs"
	case "R":
		cmds := []tea.Cmd{m.refresh()}
		if m.tab == 6 {
			cmds = append(cmds, m.loadUsage(), m.loadDaemonInfo())
		}
		return tea.Batch(cmds...)
	case "D":
		m.compact = !m.compact
	case "/":
		if m.tab == 0 {
			m.tab = 1
		}
		if m.tab >= 1 && m.tab <= 5 {
			return m.openInput("filter")
		}
		return m.notify("Filtering is available on Images, Containers, Projects, Volumes, and Networks.")
	case "o":
		if m.tab == 1 {
			m.sortMode = (m.sortMode + 1) % 3
			m.cursor = 0
		}
	case "j", "down", "shift+down":
		m.moveCursor(k, 1)
	case "k", "up", "shift+up":
		m.moveCursor(k, -1)
	case "space", " ":
		if m.tab == 1 {
			items := m.items()
			if len(items) > 0 {
				id := items[m.cursor].ID
				m.selected[id] = !m.selected[id]
			}
		}
		if m.tab == 2 {
			items := m.containers()
			if len(items) > 0 {
				id := items[m.containerCursor].ID
				m.selectedContain[id] = !m.selectedContain[id]
			}
		}
	case "i", "enter":
		if m.tab == 1 {
			return m.inspect()
		}
		if m.tab == 2 {
			return m.inspectContainer()
		}
	case "tab":
		if m.tab == 1 {
			return m.inspect()
		}
		if m.tab == 2 {
			m.containerDetail = !m.containerDetail
			if m.containerDetail {
				return m.inspectContainer()
			}
		}
	case "u":
		if m.tab == 3 {
			return m.openComposeAction(engine.ComposeUp)
		}
	case "r":
		if m.tab == 1 {
			return m.openRun()
		}
		if m.tab == 2 {
			return m.triggerContainerAction(engine.ActionRestart, false)
		}
		if m.tab == 3 {
			return m.openComposeAction(engine.ComposeRestart)
		}
	case "x":
		if m.tab == 2 {
			return m.triggerContainerAction(engine.ActionStart, false)
		}
	case "s":
		if m.tab == 2 {
			return m.triggerContainerAction(engine.ActionStop, false)
		}
	case "K":
		if m.tab == 2 {
			return m.triggerContainerAction(engine.ActionKill, true)
		}
	case "y":
		return m.copyEquivalent()
	case "d":
		if m.tab == 1 && len(m.removalIDs()) > 0 {
			m.overlay = "remove"
		}
		if m.tab == 2 {
			return m.triggerContainerAction(engine.ActionRemove, true)
		}
		if m.tab == 3 {
			return m.openComposeAction(engine.ComposeDown)
		}
		if m.tab == 4 {
			return m.openRemoveVolume()
		}
		if m.tab == 5 {
			return m.openRemoveNetwork()
		}
	case "l":
		if m.tab == 2 {
			ids := m.containerIDs()
			if len(ids) == 0 {
				return m.notify("No container selected.")
			}
			return m.openLogs(ids[0])
		}
	case "e":
		if m.tab == 2 {
			return m.openExec()
		}
	case "c":
		if m.tab == 5 {
			return m.openCreateNetwork()
		}
		return m.openCleanup()
	case "S":
		return m.openRegistrySearch()
	case "U":
		if m.tab == 1 {
			return m.checkSelectedImageUpdate()
		}
	case "t":
		if m.tab == 1 {
			return m.openTagImage()
		}
	case "P":
		if m.tab == 1 {
			return m.openPushImage()
		}
	case "E":
		if m.tab == 1 {
			return m.openExportImage()
		}
	case "I":
		if m.tab == 1 {
			return m.openImportImage()
		}
	case "V":
		if m.tab == 1 {
			return m.scanSelectedImage()
		}
	case "M":
		if m.tab == 1 || m.tab == 6 {
			return m.showDaemonMirrors()
		}
	}
	return nil
}

func (m *Model) moveCursor(key string, delta int) {
	switch m.tab {
	case 1:
		items := m.items()
		if key == "shift+down" && len(items) > 0 {
			m.selected[items[m.cursor].ID] = true
		}
		if key == "shift+up" && len(items) > 0 {
			m.selected[items[m.cursor].ID] = true
		}
		m.cursor = max(0, min(m.cursor+delta, len(items)-1))
		if (key == "shift+down" || key == "shift+up") && len(items) > 0 {
			m.selected[items[m.cursor].ID] = true
		}
	case 2:
		items := m.containers()
		if key == "shift+down" && len(items) > 0 {
			m.selectedContain[items[m.containerCursor].ID] = true
		}
		if key == "shift+up" && len(items) > 0 {
			m.selectedContain[items[m.containerCursor].ID] = true
		}
		m.containerCursor = max(0, min(m.containerCursor+delta, len(items)-1))
		if (key == "shift+down" || key == "shift+up") && len(items) > 0 {
			m.selectedContain[items[m.containerCursor].ID] = true
		}
	case 3:
		m.projectCursor = max(0, min(m.projectCursor+delta, len(m.projects())-1))
	case 4:
		m.volumeCursor = max(0, min(m.volumeCursor+delta, len(m.volumes())-1))
	case 5:
		m.networkCursor = max(0, min(m.networkCursor+delta, len(m.networks())-1))
	}
}

func (m *Model) overlayKey(msg tea.KeyPressMsg) tea.Cmd {
	k := msg.String()
	if k == "esc" {
		switch m.overlay {
		case "cleanup-confirm":
			m.overlay = "cleanup"
			m.input.Blur()
			return nil
		case "log-filter":
			m.overlay = "logs"
			m.input.Blur()
			return nil
		case "registry-search":
			if m.registryBrowsingTags {
				m.registryBrowsingTags = false
				return nil
			}
		}
		m.closeOverlay()
		return nil
	}
	switch m.overlay {
	case "help":
		if k == "?" || k == "q" {
			m.closeOverlay()
		}
		return nil
	case "remove":
		if k == "y" {
			ids := m.removalIDs()
			ctx, service := m.ctx, m.service
			m.closeOverlay()
			return func() tea.Msg { return removedMsg(service.Remove(ctx, ids)) }
		}
		if k == "n" || k == "q" {
			m.closeOverlay()
		}
		return nil
	case "pull":
		if k == "x" {
			m.jobs.Cancel(m.jobID)
		}
		if k == "J" {
			m.overlay = "jobs"
		}
		if k == "R" {
			for _, j := range m.jobList {
				if j.ID == m.jobID && j.State != "running" {
					return m.startPull(j.Target)
				}
			}
		}
		return nil
	case "jobs":
		if k == "enter" && len(m.jobList) > 0 {
			m.jobID = m.jobList[len(m.jobList)-1].ID
			m.overlay = "pull"
		}
		return nil
	case "pull-input":
		if k == "enter" {
			ref, err := services.NormalizeReference(m.input.Value())
			if err != nil {
				return m.notify(err.Error())
			}
			return m.startPull(ref)
		}
	case "filter":
		if k == "enter" {
			m.setActiveFilter(m.input.Value())
			m.closeOverlay()
			return nil
		}
	case "log-filter":
		if k == "enter" {
			value := m.input.Value()
			if value != "" {
				if _, err := regexp.Compile(value); err != nil {
					return m.notify("Invalid regular expression: " + err.Error())
				}
			}
			m.logsFilter = value
			m.overlay = "logs"
			m.input.Blur()
			return nil
		}
	case "palette":
		results := m.paletteActions()
		if k == "down" || k == "ctrl+n" {
			m.paletteCursor = min(max(0, len(results)-1), m.paletteCursor+1)
			return nil
		}
		if k == "up" {
			m.paletteCursor = max(0, m.paletteCursor-1)
			return nil
		}
		if k == "enter" && len(results) > 0 {
			index := min(m.paletteCursor, len(results)-1)
			action := results[index]
			m.closeOverlay()
			if action.key == "run-image" {
				m.tab = 1
				return m.openRun()
			}
			if action.key == "cleanup" {
				return m.openCleanup()
			}
			if strings.HasPrefix(action.key, "image:") {
				id := strings.TrimPrefix(action.key, "image:")
				m.tab = 1
				for n, i := range m.items() {
					if i.ID == id {
						m.cursor = n
						return m.inspect()
					}
				}
				return nil
			}
			if strings.HasPrefix(action.key, "container:") {
				id := strings.TrimPrefix(action.key, "container:")
				m.tab = 2
				for n, c := range m.containers() {
					if c.ID == id {
						m.containerCursor = n
						return tea.Batch(m.inspectContainer(), m.loadStats())
					}
				}
				return nil
			}
			return m.key(tea.KeyPressMsg{Code: []rune(action.key)[0], Text: action.key})
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.paletteCursor = 0
		return cmd
	case "container-action":
		if k == "y" {
			ids, action := m.actionIDs, m.actionName
			m.closeOverlay()
			return m.runContainerAction(action, ids)
		}
		if k == "n" || k == "q" {
			m.closeOverlay()
		}
		return nil
	case "compose-action":
		if k == "y" || k == "enter" {
			return m.confirmComposeAction()
		}
		if k == "v" && m.composeAction == engine.ComposeDown {
			m.composeVolumes = !m.composeVolumes
			return nil
		}
		if k == "n" || k == "q" {
			m.closeOverlay()
		}
		return nil
	case "create-network":
		if k == "enter" {
			return m.confirmCreateNetwork()
		}
		if k == "tab" {
			if m.createNetDriver == "bridge" {
				m.createNetDriver = "overlay"
			} else {
				m.createNetDriver = "bridge"
			}
			return nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	case "remove-network":
		if k == "y" || k == "enter" {
			return m.confirmRemoveNetwork()
		}
		if k == "n" || k == "q" {
			m.closeOverlay()
		}
		return nil
	case "run":
		if k == "tab" || k == "down" {
			return m.switchRunField(1)
		}
		if k == "shift+tab" || k == "up" {
			return m.switchRunField(-1)
		}
		if m.runField == 4 {
			if k == "left" || k == "h" {
				m.runForm.restart = (m.runForm.restart - 1 + len(runRestarts)) % len(runRestarts)
				return nil
			}
			if k == "right" || k == "l" {
				m.runForm.restart = (m.runForm.restart + 1) % len(runRestarts)
				return nil
			}
			if k == "enter" || k == "ctrl+s" {
				return m.submitRun()
			}
			return nil
		}
		if k == "enter" || k == "ctrl+s" {
			return m.submitRun()
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.runErr = ""
		return cmd
	case "cleanup":
		if k == "q" {
			m.closeOverlay()
			return nil
		}
		if k == "j" || k == "down" {
			m.cleanupCursor = min(len(services.CleanupKinds())-1, m.cleanupCursor+1)
			return nil
		}
		if k == "k" || k == "up" {
			m.cleanupCursor = max(0, m.cleanupCursor-1)
			return nil
		}
		if k == "space" || k == " " {
			kind := services.CleanupKinds()[m.cleanupCursor]
			m.cleanupSelected[kind] = !m.cleanupSelected[kind]
			return nil
		}
		if k == "R" {
			return m.loadCleanup()
		}
		if k == "enter" {
			if len(m.cleanupKinds()) == 0 {
				return m.notify("Select at least one cleanup category.")
			}
			m.overlay = "cleanup-confirm"
			m.input.Reset()
			if m.cleanupSelected[services.CleanupUnusedVolumes] {
				m.input.Placeholder = "volumes"
				return m.input.Focus()
			}
			m.input.Blur()
			return nil
		}
		return nil
	case "cleanup-confirm":
		if k == "q" {
			m.closeOverlay()
			return nil
		}
		if k == "enter" {
			if m.cleanupSelected[services.CleanupUnusedVolumes] && strings.TrimSpace(m.input.Value()) != "volumes" {
				return m.notify(`Type "volumes" to confirm volume deletion.`)
			}
			return m.executeCleanup()
		}
		if m.cleanupSelected[services.CleanupUnusedVolumes] {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return cmd
		}
		return nil
	case "remove-volume":
		if k == "enter" {
			return m.confirmRemoveVolume()
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	case "logs":
		switch k {
		case "q":
			m.closeOverlay()
		case "p":
			return m.pauseLogs()
		case "f":
			return m.streamLogs(!m.logsFollow, true)
		case "t":
			m.logsTimestamps = !m.logsTimestamps
		case "w":
			m.logsWrap = !m.logsWrap
		case "/":
			return m.openInput("log-filter")
		case "c":
			return m.copyLogs()
		case "s":
			return m.saveLogs()
		}
		return nil
	case "exec":
		if k == "q" {
			m.closeOverlay()
			return nil
		}
		if k == "j" || k == "down" || k == "right" {
			m.execShell = (m.execShell + 1) % len(execShells)
			return nil
		}
		if k == "k" || k == "up" || k == "left" {
			m.execShell = (m.execShell - 1 + len(execShells)) % len(execShells)
			return nil
		}
		if k == "enter" {
			return m.runExec()
		}
		return nil
	case "registry-search":
		if k == "esc" {
			if m.registryBrowsingTags {
				m.registryBrowsingTags = false
				return nil
			}
			m.closeOverlay()
			return nil
		}
		if k == "down" || k == "j" {
			if m.registryBrowsingTags {
				if len(m.registryTags) > 0 {
					m.registryTagCursor = min(len(m.registryTags)-1, m.registryTagCursor+1)
				}
			} else {
				if len(m.registryResults) > 0 {
					m.registryCursor = min(len(m.registryResults)-1, m.registryCursor+1)
				}
			}
			return nil
		}
		if k == "up" || k == "k" {
			if m.registryBrowsingTags {
				m.registryTagCursor = max(0, m.registryTagCursor-1)
			} else {
				m.registryCursor = max(0, m.registryCursor-1)
			}
			return nil
		}
		if k == "t" && !m.registryBrowsingTags && len(m.registryResults) > 0 {
			repo := m.registryResults[m.registryCursor].Name
			return m.fetchRemoteTags(repo)
		}
		if k == "enter" {
			if m.registryBrowsingTags && len(m.registryTags) > 0 {
				repo := m.registryResults[m.registryCursor].Name
				tag := m.registryTags[m.registryTagCursor].Name
				ref := repo + ":" + tag
				m.closeOverlay()
				return m.startPull(ref)
			}
			if len(m.registryResults) > 0 && strings.TrimSpace(m.input.Value()) == m.registryQuery {
				repo := m.registryResults[m.registryCursor].Name
				m.closeOverlay()
				return m.startPull(repo + ":latest")
			}
			m.registryQuery = strings.TrimSpace(m.input.Value())
			return m.executeRegistrySearch(m.input.Value())
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	case "tag-image":
		if k == "enter" {
			return m.submitTagImage()
		}
		if k == "esc" {
			m.closeOverlay()
			return nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	case "push-image":
		if k == "esc" || k == "q" {
			m.closeOverlay()
			return nil
		}
		return nil
	case "export-image":
		if k == "enter" {
			return m.submitExportImage()
		}
		if k == "esc" {
			m.closeOverlay()
			return nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	case "import-image":
		if k == "enter" {
			return m.submitImportImage()
		}
		if k == "esc" {
			m.closeOverlay()
			return nil
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return cmd
	case "daemon-mirrors":
		if k == "esc" || k == "q" {
			m.closeOverlay()
			return nil
		}
		if k == "y" && m.mirrorSudoCmd != "" {
			return tea.Batch(tea.SetClipboard(m.mirrorSudoCmd), m.notify("Copied mirror config command."))
		}
		return nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.paletteCursor = 0
	return cmd
}

func (m *Model) startPull(ref string) tea.Cmd {
	m.input.Blur()
	m.overlay = "pull"
	ctx, manager, service := m.ctx, m.jobs, m.service
	return func() tea.Msg { id := manager.Start(ctx, service, ref); return pullStartedMsg{id} }
}

type pullStartedMsg struct{ id uint64 }

func (m *Model) removalIDs() []string {
	ids := []string{}
	for _, i := range m.store.Images {
		if m.selected[i.ID] {
			ids = append(ids, i.ID)
		}
	}
	if len(ids) == 0 {
		items := m.items()
		if len(items) > 0 {
			ids = append(ids, items[m.cursor].ID)
		}
	}
	return ids
}

func (m *Model) mouse(x, y int) tea.Cmd {
	if m.overlay != "" {
		return nil
	}
	if y == 1 {
		offset := 0
		for n, name := range components.TabNames() {
			label := fmt.Sprintf("%d %s", n+1, name)
			if m.width < tokens.DetailWidth {
				label = fmt.Sprintf("%d %.3s", n+1, name)
			}
			end := offset + len(label) + tokens.Gap
			if x >= offset && x < end {
				m.tab = n
				m.detail = false
				m.containerDetail = false
				if n == 6 {
					return m.loadUsage()
				}
				if n == 2 {
					return m.loadStats()
				}
				return nil
			}
			offset = end
		}
	}
	if m.tab == 1 && y >= tokens.HeaderHeight+tokens.PanelFrame+1 {
		row := y - tokens.HeaderHeight - tokens.PanelFrame - 1
		capacity := max(1, m.height-tokens.HeaderHeight-tokens.FooterHeight-tokens.PanelFrame-tokens.InsetY*2)
		start := max(0, m.cursor-capacity+1)
		index := start + row
		if index < len(m.items()) {
			m.cursor = index
		}
	}
	if m.tab == 2 && y >= tokens.HeaderHeight+tokens.PanelFrame+1 {
		row := y - tokens.HeaderHeight - tokens.PanelFrame - 1
		capacity := max(1, m.height-tokens.HeaderHeight-tokens.FooterHeight-tokens.PanelFrame-tokens.InsetY*2)
		start := max(0, m.containerCursor-capacity+1)
		index := start + row
		if index < len(m.containers()) {
			m.containerCursor = index
		}
	}
	if m.tab == 4 && y >= tokens.HeaderHeight+tokens.PanelFrame+1 {
		row := y - tokens.HeaderHeight - tokens.PanelFrame - 1
		capacity := max(1, m.height-tokens.HeaderHeight-tokens.FooterHeight-tokens.PanelFrame-tokens.InsetY*2)
		start := max(0, m.volumeCursor-capacity+1)
		index := start + row
		if index < len(m.volumes()) {
			m.volumeCursor = index
		}
	}
	return nil
}
