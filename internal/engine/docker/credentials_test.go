package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/moby/moby/api/types/registry"
)

func TestRegistryAuthUsesExistingDockerConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOCKER_CONFIG", dir)
	data := `{"auths":{"https://index.docker.io/v1/":{"auth":"dXNlcjpwYXNz"}}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	encoded, err := registryAuth(context.Background(), "redis:7")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatal(err)
	}
	var auth registry.AuthConfig
	if err := json.Unmarshal(decoded, &auth); err != nil {
		t.Fatal(err)
	}
	if auth.Username != "user" || auth.Password != "pass" {
		t.Fatal("Docker auth was not decoded")
	}
}

func TestRegistryAuthRejectsUnsafeHelperName(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOCKER_CONFIG", dir)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"credsStore":"../../unsafe"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := registryAuth(context.Background(), "redis:7"); err == nil {
		t.Fatal("unsafe credential helper accepted")
	}
}
