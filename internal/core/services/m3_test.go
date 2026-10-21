package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/engine/fake"
)

func TestParseTrivyOutput(t *testing.T) {
	trivyJSON := []byte(`{
		"Results": [
			{
				"Vulnerabilities": [
					{"Severity": "CRITICAL", "FixedVersion": "1.2.3"},
					{"Severity": "HIGH", "FixedVersion": ""},
					{"Severity": "HIGH", "FixedVersion": "2.0.0"},
					{"Severity": "MEDIUM", "FixedVersion": ""},
					{"Severity": "LOW", "FixedVersion": "0.1"}
				]
			}
		]
	}`)

	report, err := parseTrivyOutput(trivyJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Scanner != "trivy" {
		t.Errorf("expected scanner trivy, got %s", report.Scanner)
	}
	if report.Critical != 1 {
		t.Errorf("expected 1 critical, got %d", report.Critical)
	}
	if report.High != 2 {
		t.Errorf("expected 2 high, got %d", report.High)
	}
	if report.Medium != 1 {
		t.Errorf("expected 1 medium, got %d", report.Medium)
	}
	if report.Low != 1 {
		t.Errorf("expected 1 low, got %d", report.Low)
	}
	if report.Fixable != 3 {
		t.Errorf("expected 3 fixable, got %d", report.Fixable)
	}
}

func TestParseGrypeOutput(t *testing.T) {
	grypeJSON := []byte(`{
		"matches": [
			{
				"vulnerability": {
					"severity": "Critical"
				},
				"fix": {
					"state": "fixed"
				}
			},
			{
				"vulnerability": {
					"severity": "Medium"
				},
				"fix": {
					"state": "not-fixed"
				}
			}
		]
	}`)

	report, err := parseGrypeOutput(grypeJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Scanner != "grype" {
		t.Errorf("expected scanner grype, got %s", report.Scanner)
	}
	if report.Critical != 1 {
		t.Errorf("expected 1 critical, got %d", report.Critical)
	}
	if report.Medium != 1 {
		t.Errorf("expected 1 medium, got %d", report.Medium)
	}
	if report.Fixable != 1 {
		t.Errorf("expected 1 fixable, got %d", report.Fixable)
	}
}

func TestRewriteMirrorReference(t *testing.T) {
	tests := []struct {
		ref      string
		mirror   string
		expected string
	}{
		{
			ref:      "nginx:1.27",
			mirror:   "mirror.gcr.io",
			expected: "mirror.gcr.io/library/nginx:1.27",
		},
		{
			ref:      "docker.io/library/redis:latest",
			mirror:   "https://mirror.baidubce.com/",
			expected: "mirror.baidubce.com/library/redis:latest",
		},
		{
			ref:      "ghcr.io/owner/repo:v1.0",
			mirror:   "internal-mirror:5000",
			expected: "internal-mirror:5000/owner/repo:v1.0",
		},
	}

	for _, tt := range tests {
		rewritten, err := RewriteMirrorReference(tt.ref, tt.mirror)
		if err != nil {
			t.Fatalf("failed rewriting %s: %v", tt.ref, err)
		}
		if rewritten != tt.expected {
			t.Errorf("RewriteMirrorReference(%q, %q) = %q, want %q", tt.ref, tt.mirror, rewritten, tt.expected)
		}
	}
}

func TestGenerateDaemonMirrorsDiff(t *testing.T) {
	existing := `{"log-driver": "json-file"}`
	mirrors := []string{"mirror.gcr.io", "https://registry-1.docker.io"}

	diff, newCfg, err := GenerateDaemonMirrorsDiff(existing, mirrors)
	if err != nil {
		t.Fatalf("failed generating diff: %v", err)
	}
	if !strings.Contains(diff, "--- /etc/docker/daemon.json") {
		t.Errorf("diff missing header: %s", diff)
	}
	if !strings.Contains(diff, "+") {
		t.Errorf("diff missing additions: %s", diff)
	}
	if !strings.Contains(newCfg, "registry-mirrors") {
		t.Errorf("new config missing registry-mirrors key: %s", newCfg)
	}
}

func TestRegistryServicesWithFakeEngine(t *testing.T) {
	ctx := context.Background()
	e := fake.New()
	svc := New(e)

	// Search
	results, err := svc.SearchRegistry(ctx, "nginx")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results for nginx")
	}
	if results[0].Name != "nginx" {
		t.Errorf("expected first result to be nginx, got %s", results[0].Name)
	}

	// Tags
	tags, err := svc.ListRemoteTags(ctx, "nginx")
	if err != nil {
		t.Fatalf("list tags failed: %v", err)
	}
	if len(tags) == 0 {
		t.Fatal("expected tags for nginx")
	}

	// Update Check
	status, err := svc.CheckUpdate(ctx, "nginx:latest", "sha256:old")
	if err != nil {
		t.Fatalf("check update failed: %v", err)
	}
	if !status.HasUpdate {
		t.Error("expected update available for nginx:latest")
	}

	// Tag image
	snap, err := svc.Snapshot(ctx)
	if err != nil || len(snap.Images) == 0 {
		t.Fatalf("snapshot failed: %v", err)
	}
	sourceImg := snap.Images[0]
	if err := svc.TagImage(ctx, sourceImg.ID, "custom-tag:v1"); err != nil {
		t.Fatalf("tag failed: %v", err)
	}

	// Push image
	var progressCount int
	err = svc.PushImage(ctx, "custom-tag:v1", func(p engine.Progress) error {
		progressCount++
		return nil
	})
	if err != nil {
		t.Fatalf("push failed: %v", err)
	}
	if progressCount == 0 {
		t.Error("expected push progress emissions")
	}

	// Save and Load image
	tmpFile := filepath.Join(os.TempDir(), "test-save-load.tar")
	defer os.Remove(tmpFile)

	if err := svc.SaveImage(ctx, sourceImg.ID, tmpFile); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if err := svc.LoadImage(ctx, tmpFile); err != nil {
		t.Fatalf("load failed: %v", err)
	}
}
