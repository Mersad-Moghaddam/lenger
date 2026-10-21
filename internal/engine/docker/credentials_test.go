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

func TestCredentialSelectionPrefersExactThenStableLegacyKeys(t *testing.T) {
	cfg := credentialConfig{Auths: map[string]registry.AuthConfig{
		"registry.example":             {Username: "exact"},
		"https://registry.example/v1/": {Username: "legacy"},
		"http://registry.example/v1/":  {Username: "first-legacy"},
	}}
	if got := cfg.authFor("registry.example").Username; got != "exact" {
		t.Fatalf("exact registry key lost precedence: %q", got)
	}
	delete(cfg.Auths, "registry.example")
	if got := cfg.authFor("registry.example").Username; got != "first-legacy" {
		t.Fatalf("legacy credential selection is not deterministic: %q", got)
	}
}

func TestCredentialSelectionPreservesEmptyExactEntry(t *testing.T) {
	cfg := credentialConfig{Auths: map[string]registry.AuthConfig{
		"registry.example":             {},
		"https://registry.example/v1/": {Username: "stale"},
	}}
	if got := cfg.authFor("registry.example"); got.Username != "" {
		t.Fatal("an empty exact entry reused stale legacy credentials")
	}
}

func TestDecodeStoredAuthRejectsMalformedCredentials(t *testing.T) {
	for _, encoded := range []string{"not-base64!", "bm8tY29sb24="} {
		if _, err := decodeStoredAuth(registry.AuthConfig{Auth: encoded}); err == nil {
			t.Fatal("malformed stored credentials were accepted")
		}
	}
}

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
