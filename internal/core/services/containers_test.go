package services

import (
	"context"
	"strings"
	"testing"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/engine/fake"
)

func TestRunContainerLifecycleAndMaskedInspection(t *testing.T) {
	ctx := context.Background()
	s := New(fake.New())
	id, err := s.Run(ctx, engine.RunSpec{
		Name: "api-1", Image: "nginx:1.27-alpine", Network: "bridge",
		Ports: []string{"8080:80"}, Env: []string{"MODE=test"},
		Volumes: []string{"scratch-data:/data"}, Restart: "unless-stopped",
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := s.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range snapshot.Containers {
		if c.ID == id {
			found = c.State == "running" && c.Name == "api-1" && len(c.Ports) == 1
		}
	}
	if !found {
		t.Fatal("started container was not present in the snapshot")
	}
	volumeFound := false
	for _, v := range snapshot.Volumes {
		if v.Name == "scratch-data" {
			volumeFound = v.RefCount == 1 && len(v.UsedBy) == 1
		}
	}
	if !volumeFound {
		t.Fatal("volume reference index did not include the new container")
	}
	inspection, err := s.InspectContainer(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range inspection.Env {
		if strings.Contains(value, "fixture-secret") || !strings.HasSuffix(value, "=••••") {
			t.Fatalf("unmasked container environment: %s", value)
		}
	}
	if len(inspection.Command) != 1 || !strings.Contains(inspection.Command[0], "hidden") {
		t.Fatal("container command was not masked")
	}
	var logLines int
	if err := s.Logs(ctx, id, engine.LogOptions{Tail: 3}, func(entry engine.LogEntry) error {
		if entry.At.IsZero() {
			t.Fatal("fixture log lost its timestamp")
		}
		logLines++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if logLines != 3 {
		t.Fatalf("got %d log lines, want 3", logLines)
	}
	if err := s.ActContainers(ctx, []string{id}, engine.ActionRemove)[id]; err == nil {
		t.Fatal("running container removal was allowed")
	}
	if err := s.ActContainers(ctx, []string{id}, engine.ActionStop)[id]; err != nil {
		t.Fatal(err)
	}
	if err := s.ActContainers(ctx, []string{id}, engine.ActionRemove)[id]; err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsUnsafeOrInvalidForms(t *testing.T) {
	for _, tc := range []struct {
		name string
		spec engine.RunSpec
	}{
		{"empty image", engine.RunSpec{}},
		{"bad name", engine.RunSpec{Image: "redis:7", Name: "bad name"}},
		{"bad port", engine.RunSpec{Image: "redis:7", Ports: []string{"70000:6379"}}},
		{"bad env", engine.RunSpec{Image: "redis:7", Env: []string{"NO_EQUALS"}}},
		{"bad mount", engine.RunSpec{Image: "redis:7", Volumes: []string{"relative:/data"}}},
		{"bad restart", engine.RunSpec{Image: "redis:7", Restart: "sometimes"}},
		{"bad network", engine.RunSpec{Image: "redis:7", Network: "bad network"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NormalizeRunSpec(tc.spec); err == nil {
				t.Fatal("invalid form was accepted")
			}
		})
	}
}

func TestRunCommandPreviewAndCommandParsing(t *testing.T) {
	args, err := ParseCommandLine(`sh -c "echo hello" 'two words'`)
	if err != nil || len(args) != 4 || args[2] != "echo hello" || args[3] != "two words" {
		t.Fatalf("unexpected parsed command: %#v, %v", args, err)
	}
	command := RunCommand(engine.RunSpec{
		Name: "api-1", Image: "nginx:1.27", Restart: "unless-stopped",
		Ports: []string{"8080:80"}, Env: []string{"GREETING=hello world"},
		Volumes: []string{"scratch-data:/data:ro"}, Network: "bridge",
		Command: []string{"sh", "-c", "echo hello"},
	})
	for _, want := range []string{
		"docker run -d", "--name api-1", "-p 8080:80", "'GREETING=hello world'",
		"-v scratch-data:/data:ro", "--restart unless-stopped", "--network bridge",
		"nginx:1.27", "sh -c 'echo hello'",
	} {
		if !strings.Contains(command, want) {
			t.Fatalf("preview %q is missing %q", command, want)
		}
	}
	if _, err := ParseCommandLine(`sh "unfinished`); err == nil {
		t.Fatal("unfinished quote was accepted")
	}
}
