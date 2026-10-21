package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Mersad-Moghaddam/lenger/internal/core/services"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

func TestRunFormBuildsAValidatedSpec(t *testing.T) {
	form := runForm{
		name:    "api-1",
		ports:   "8080:80, 127.0.0.1:5432:5432",
		env:     "MODE=test, TOKEN=local",
		volumes: "scratch-data:/data, /tmp/input:/input:ro",
		network: "bridge",
		command: `sh -c "echo hello"`,
		restart: 1,
	}
	spec, err := form.spec("nginx:1.27-alpine")
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := services.NormalizeRunSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.Name != "api-1" || normalized.Image != "nginx:1.27-alpine" {
		t.Fatalf("unexpected identity: %#v", normalized)
	}
	if len(normalized.Ports) != 2 || len(normalized.Env) != 2 || len(normalized.Volumes) != 2 {
		t.Fatalf("form lists were not split: %#v", normalized)
	}
	if normalized.Restart != "unless-stopped" {
		t.Fatalf("restart policy = %q", normalized.Restart)
	}
	if got := services.RunCommand(normalized); !strings.Contains(got, "--name api-1") || !strings.Contains(got, "nginx:1.27-alpine") {
		t.Fatalf("preview command did not describe the form: %q", got)
	}
}

func TestRunWizardRoutesFieldsWithoutLeakingInput(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 1
	_, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if m.overlay != "run" || m.runImage == "" {
		t.Fatal("run key did not open the wizard")
	}
	if m.runField != 0 {
		t.Fatalf("initial field = %d", m.runField)
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.runField != 1 {
		t.Fatalf("tab moved to field %d", m.runField)
	}
	m.input.SetValue("8080:80")
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.runField != 2 || m.runForm.ports != "8080:80" {
		t.Fatal("field change did not save the previous value")
	}
}

func TestContainerLogsDrawerPausesFollowWithoutStoppingCollection(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'l', Text: "l"})
	if cmd == nil || !m.focus.DrawerOpen || m.focus.ActiveTab != 1 || m.logsID == "" {
		t.Fatal("logs key did not open a bounded stream")
	}
	m.focus.Area = FocusDrawerControls
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.drawerLogs.Follow || m.logsCancel == nil {
		t.Fatal("scroll must pause follow, not detach collection")
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	if !m.drawerLogs.Follow {
		t.Fatal("follow did not restart the stream")
	}
	m.closeDrawer()
}

func TestCleanupRequiresTypedVolumesConfirmation(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	if cmd == nil || m.overlay != "cleanup" {
		t.Fatal("cleanup key did not open a preview")
	}
	m.cleanupLoaded = true
	m.cleanupLoading = false
	m.cleanupSelected = defaultCleanupSelection()
	m.cleanupSelected[services.CleanupUnusedVolumes] = true
	m.overlay = "cleanup-confirm"
	m.input.SetValue("wrong")
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || m.overlay != "cleanup-confirm" || m.cleanupLoading {
		t.Fatal("wrong confirmation must be rejected without mutation")
	}
	m.input.SetValue("volumes")
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || !m.cleanupLoading {
		t.Fatal("correct confirmation did not start the cleanup command")
	}
}

func TestVolumeRemovalRejectsUsedVolumesAndWrongConfirmation(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 2
	items := m.volumes()
	for n, volume := range items {
		if volume.Name == "postgres-data" {
			m.volumeCursor = n
			break
		}
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if m.overlay == "remove-volume" {
		t.Fatal("a referenced volume must not open removal")
	}
	for n, volume := range items {
		if volume.RefCount == 0 && len(volume.UsedBy) == 0 {
			m.volumeCursor = n
			break
		}
	}
	_, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if m.overlay != "remove-volume" {
		t.Fatal("unused volume did not open confirmation")
	}
	m.input.SetValue("nope")
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil || m.overlay != "remove-volume" {
		t.Fatal("wrong volume confirmation must not close the dialog")
	}
}

func TestCopyEquivalentUsesTheActiveResource(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 1
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd == nil || !strings.HasPrefix(m.toast, "Copied: docker image inspect ") {
		t.Fatalf("image copy toast = %q", m.toast)
	}
	m.tab = 2
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd == nil || !strings.HasPrefix(m.toast, "Copied: docker volume inspect ") {
		t.Fatalf("volume copy toast = %q", m.toast)
	}
}

func TestStatsMessageDerivesCPUFromCumulativeTotals(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	var id string
	for _, c := range m.store.Containers {
		if c.Running() {
			id = c.ID
			break
		}
	}
	if id == "" {
		t.Fatal("fixture has no running container")
	}
	m.stats[id] = engine.ContainerStats{SystemTotal: 1_000_000_000, CPUTotal: 25_000_000, OnlineCPUs: 4}
	m.statHistory[id] = nil
	_, cmd := m.Update(statsMsg{data: map[string]engine.ContainerStats{id: {
		SystemTotal: 2_000_000_000, CPUTotal: 75_000_000, OnlineCPUs: 4, Memory: 96_000_000,
	}}})
	if cmd == nil {
		t.Fatal("a live sample did not schedule the next stats tick")
	}
	if got := m.statCPU[id]; got < 19.9 || got > 20.1 {
		t.Fatalf("derived CPU percent = %.2f, want 20", got)
	}
	if len(m.statHistory[id]) != 1 {
		t.Fatalf("history length = %d", len(m.statHistory[id]))
	}
}

func TestContainerActionConfirmationCarriesSelectedIDs(t *testing.T) {
	m := fixtureModel(t)
	m.tab = 0
	items := m.containers()
	if len(items) == 0 {
		t.Fatal("fixture has no containers")
	}
	m.containerCursor = 0
	m.selectedContain[items[0].ID] = true
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if cmd != nil || m.overlay != "container-action" || m.actionName != engine.ActionRemove {
		t.Fatal("destructive container action did not request confirmation")
	}
	if len(m.actionIDs) != 1 || m.actionIDs[0] != items[0].ID {
		t.Fatalf("confirmation IDs = %v", m.actionIDs)
	}
}
