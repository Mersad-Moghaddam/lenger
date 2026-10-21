package views

import (
	"strings"
	"testing"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/theme"
)

func testSheet() style.Sheet {
	p, _ := theme.Bundled("omarchy")
	return style.Sheet{Palette: p, Plain: false, ASCII: false}
}

func TestCleanLayerCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "/bin/sh -c #(nop)  CMD [\"nginx\" \"-g\" \"daemon off;\"]",
			expected: "CMD [\"nginx\" \"-g\" \"daemon off;\"]",
		},
		{
			input:    "/bin/sh -c apk add --no-cache curl",
			expected: "apk add --no-cache curl",
		},
		{
			input:    "#(nop) ENV PATH=/usr/local/bin:/usr/bin",
			expected: "ENV PATH=/usr/local/bin:/usr/bin",
		},
	}

	for _, tt := range tests {
		got := CleanLayerCommand(tt.input)
		if got != tt.expected {
			t.Errorf("CleanLayerCommand(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestLayerBreakdown(t *testing.T) {
	s := testSheet()
	layers := []engine.Layer{
		{ID: "layer1", Command: "FROM alpine", Size: 5 * 1024 * 1024},
		{ID: "layer2", Command: "RUN apk add --no-cache curl", Size: 12 * 1024 * 1024},
		{ID: "layer3", Command: "COPY app /app", Size: 45 * 1024 * 1024},
		{ID: "layer4", Command: "CMD /app", Size: 0},
	}

	out := LayerBreakdown(s, layers, 80)
	if !strings.Contains(out, "BIGGEST") {
		t.Errorf("expected biggest layer callout, got:\n%s", out)
	}
	if !strings.Contains(out, "45.0 MB") {
		t.Errorf("expected size 45.0 MB in output, got:\n%s", out)
	}
}

func TestReverseDependencies(t *testing.T) {
	s := testSheet()
	img := engine.Image{
		ID:       "sha256:1234567890abcdef",
		RepoTags: []string{"nginx:latest"},
	}

	containers := []engine.Container{
		{
			ID:      "c1",
			Name:    "web-server",
			ImageID: img.ID,
			Image:   "nginx:latest",
			State:   "running",
			Project: "storefront",
			Service: "web",
		},
	}

	projects := []engine.ComposeProject{
		{
			Name: "storefront",
		},
	}

	out := ReverseDependencies(s, img, containers, projects, 80)
	if !strings.Contains(out, "storefront") {
		t.Errorf("expected Compose project storefront in dependencies, got:\n%s", out)
	}
	if !strings.Contains(out, "web-server") {
		t.Errorf("expected container web-server in dependencies, got:\n%s", out)
	}

	// Test unreferenced image
	unreferencedOut := ReverseDependencies(s, engine.Image{ID: "sha256:unused"}, nil, nil, 80)
	if !strings.Contains(unreferencedOut, "Safe to remove") {
		t.Errorf("expected Safe to remove callout for unreferenced image, got:\n%s", unreferencedOut)
	}
}

func TestImageDetailWithM3Soundings(t *testing.T) {
	s := testSheet()
	img := engine.Image{
		ID:       "sha256:1234567890abcdef",
		RepoTags: []string{"nginx:1.27"},
		Size:     48 * 1024 * 1024,
		Created:  time.Now().Add(-2 * time.Hour),
	}

	inspect := engine.Inspection{
		ID:       img.ID,
		Platform: "linux/amd64",
		Layers: []engine.Layer{
			{ID: "base", Command: "FROM alpine", Size: 8 * 1024 * 1024},
			{ID: "nginx", Command: "RUN apk add nginx", Size: 40 * 1024 * 1024},
		},
		UpdateStatus: &engine.ImageUpdateStatus{
			Ref:          "nginx:1.27",
			HasUpdate:    true,
			RemoteDigest: "sha256:remote-digest",
		},
		Vulnerabilities: &engine.VulnerabilityReport{
			Scanner:  "trivy",
			Critical: 2,
			High:     5,
			Medium:   8,
			Low:      15,
			Fixable:  7,
		},
	}

	out := ImageDetail(s, img, &inspect, 100, 30, time.Now())
	if !strings.Contains(out, "UPDATE AVAILABLE") {
		t.Errorf("expected UPDATE AVAILABLE in detail view, got:\n%s", out)
	}
	if !strings.Contains(out, "SECURITY SCAN") {
		t.Errorf("expected SECURITY SCAN in detail view, got:\n%s", out)
	}
	if !strings.Contains(out, "CRIT:2") {
		t.Errorf("expected CRIT:2 in detail view, got:\n%s", out)
	}
	if !strings.Contains(out, "BIGGEST") {
		t.Errorf("expected BIGGEST layer badge in detail view, got:\n%s", out)
	}
}
