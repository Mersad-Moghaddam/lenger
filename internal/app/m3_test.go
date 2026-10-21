package app

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

func TestRegistrySearchOverlayAndNavigation(t *testing.T) {
	m := fixtureModel(t)

	// Press S to open registry search overlay
	_, _ = m.Update(tea.KeyPressMsg{Code: 'S', Text: "S"})
	if m.overlay != "registry-search" {
		t.Fatalf("expected registry-search overlay, got %s", m.overlay)
	}

	view := m.View()
	if !strings.Contains(view, "REGISTRY SEARCH") {
		t.Fatalf("expected search header in view: %s", view)
	}

	// Deliver simulated search results
	results := []engine.RegistrySearchResult{
		{Name: "nginx", Description: "Official build of Nginx.", Stars: 20100, Official: true},
		{Name: "bitnami/nginx", Description: "Bitnami container image for NGINX.", Stars: 430, Official: false, Automated: true},
	}
	_, _ = m.Update(registrySearchMsg{query: "nginx", results: results})

	view = m.View()
	if !strings.Contains(view, "nginx") || !strings.Contains(view, "OFFICIAL") {
		t.Fatalf("expected nginx and OFFICIAL badge in view: %s", view)
	}

	// Press t to browse tags
	_, _ = m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})

	// Deliver simulated remote tags
	tags := []engine.RemoteTag{
		{Name: "latest", Digest: "sha256:latestdigest", Size: 48000000, UpdatedAt: time.Now()},
		{Name: "1.27-alpine", Digest: "sha256:alpinedigest", Size: 44000000, UpdatedAt: time.Now()},
	}
	_, _ = m.Update(remoteTagsMsg{repo: "nginx", tags: tags})

	view = m.View()
	if !strings.Contains(view, "TAGS FOR nginx") || !strings.Contains(view, "1.27-alpine") {
		t.Fatalf("expected remote tags view: %s", view)
	}

	// Press Esc to return to search results
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !m.registryBrowsingTags && m.overlay == "registry-search" {
		// stepped back to search
	} else {
		t.Fatalf("expected back to search results, got browsingTags=%v overlay=%s", m.registryBrowsingTags, m.overlay)
	}

	// Press Esc again to close overlay
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.overlay != "" {
		t.Fatalf("expected overlay closed, got %s", m.overlay)
	}
}

func TestTagAndPushImageWorkflows(t *testing.T) {
	m := fixtureModel(t)
	// Go to Images tab
	_, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	if m.tab != 1 {
		t.Fatalf("expected images tab 1, got %d", m.tab)
	}

	// Press t to open tag overlay
	_, _ = m.Update(tea.KeyPressMsg{Code: 't', Text: "t"})
	if m.overlay != "tag-image" {
		t.Fatalf("expected tag-image overlay, got %s", m.overlay)
	}
	if !strings.Contains(m.View(), "TAG IMAGE") {
		t.Fatalf("expected TAG IMAGE header in view: %s", m.View())
	}

	// Simulate tag success
	_, _ = m.Update(imageTaggedMsg{sourceID: m.tagSourceID, targetRef: "myreg.io/app:v1"})
	if m.overlay != "" {
		t.Fatalf("expected overlay closed after tagging, got %s", m.overlay)
	}

	// Press P to open push overlay
	_, _ = m.Update(tea.KeyPressMsg{Code: 'P', Text: "P"})
	if m.overlay != "push-image" {
		t.Fatalf("expected push-image overlay, got %s", m.overlay)
	}

	// Deliver push progress
	_, _ = m.Update(pushProgressMsg{progress: engine.Progress{ID: "layer-1", Status: "Pushing", Current: 50, Total: 100}})
	if !strings.Contains(m.View(), "PUSH IMAGE TO REGISTRY") {
		t.Fatalf("expected push view: %s", m.View())
	}

	// Deliver push done
	_, _ = m.Update(pushDoneMsg{ref: "myreg.io/app:v1"})
	if m.overlay != "" {
		t.Fatalf("expected push overlay closed on completion, got %s", m.overlay)
	}
}

func TestExportAndImportTarballWorkflows(t *testing.T) {
	m := fixtureModel(t)
	_, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})

	// Press E for export
	_, _ = m.Update(tea.KeyPressMsg{Code: 'E', Text: "E"})
	if m.overlay != "export-image" {
		t.Fatalf("expected export-image overlay, got %s", m.overlay)
	}
	if !strings.Contains(m.View(), "EXPORT IMAGE ARCHIVE") {
		t.Fatalf("expected export view: %s", m.View())
	}

	// Cancel export
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.overlay != "" {
		t.Fatalf("expected overlay closed, got %s", m.overlay)
	}

	// Press I for import
	_, _ = m.Update(tea.KeyPressMsg{Code: 'I', Text: "I"})
	if m.overlay != "import-image" {
		t.Fatalf("expected import-image overlay, got %s", m.overlay)
	}
	if !strings.Contains(m.View(), "IMPORT IMAGE ARCHIVE") {
		t.Fatalf("expected import view: %s", m.View())
	}
}

func TestDaemonMirrorsDiffOverlay(t *testing.T) {
	m := fixtureModel(t)

	// Press M to open daemon mirrors
	_, _ = m.Update(tea.KeyPressMsg{Code: 'M', Text: "M"})

	// Deliver simulated diff
	diffText := "--- /etc/docker/daemon.json\n+++ /etc/docker/daemon.json\n+ \"registry-mirrors\": [\"https://mirror.gcr.io\"]"
	newCfg := "{\n  \"registry-mirrors\": [\n    \"https://mirror.gcr.io\"\n  ]\n}"
	_, _ = m.Update(daemonMirrorsMsg{diff: diffText, newConfig: newCfg})

	if m.overlay != "daemon-mirrors" {
		t.Fatalf("expected daemon-mirrors overlay, got %s", m.overlay)
	}
	view := m.View()
	if !strings.Contains(view, "DAEMON REGISTRY MIRRORS CONFIGURATION") {
		t.Fatalf("expected daemon mirrors header: %s", view)
	}
	if !strings.Contains(view, "https://mirror.gcr.io") {
		t.Fatalf("expected mirror URL in diff preview: %s", view)
	}
}

func TestUpdateAndVulnerabilityScanMessages(t *testing.T) {
	m := fixtureModel(t)
	_, _ = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})

	items := m.items()
	if len(items) == 0 {
		t.Fatal("expected items in fixture")
	}
	targetID := items[0].ID

	// Deliver update check message
	updateStatus := engine.ImageUpdateStatus{
		Ref:          items[0].Name(),
		LocalDigest:  "sha256:localdigest",
		RemoteDigest: "sha256:remotenewer",
		HasUpdate:    true,
		CheckedAt:    time.Now(),
	}
	_, _ = m.Update(imageUpdateMsg{id: targetID, status: updateStatus})

	if ins, ok := m.inspections[targetID]; !ok || ins.UpdateStatus == nil || !ins.UpdateStatus.HasUpdate {
		t.Fatal("expected inspection to have update status with HasUpdate=true")
	}

	// Deliver vulnerability scan message
	report := engine.VulnerabilityReport{
		Scanner:   "trivy",
		Critical:  2,
		High:      4,
		Medium:    6,
		Low:       10,
		Fixable:   5,
		ScannedAt: time.Now(),
	}
	_, _ = m.Update(vulnScanMsg{id: targetID, report: report})

	if ins, ok := m.inspections[targetID]; !ok || ins.Vulnerabilities == nil || ins.Vulnerabilities.Critical != 2 {
		t.Fatal("expected inspection to have vulnerability report with Critical=2")
	}
}
