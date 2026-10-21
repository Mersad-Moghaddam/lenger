package docker

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/moby/moby/api/types/registry"
)

// credentialConfig models only the Docker fields needed for read-only pulls.
type credentialConfig struct {
	Auths   map[string]registry.AuthConfig `json:"auths"`
	Helpers map[string]string              `json:"credHelpers"`
	Store   string                         `json:"credsStore"`
}

func loadCredentialConfig() (credentialConfig, error) {
	var cfg credentialConfig
	dir := os.Getenv("DOCKER_CONFIG")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return cfg, fmt.Errorf("find Docker configuration: %w", err)
		}
		dir = filepath.Join(home, ".docker")
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read Docker configuration: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("decode Docker configuration: %w", err)
	}
	return cfg, nil
}

func (c credentialConfig) authFor(server string) registry.AuthConfig {
	if auth, ok := c.Auths[server]; ok {
		return auth
	}
	// Prefer exact keys, then a stable ordering of legacy URL-shaped keys.
	// Never choose between credentials using randomized map iteration order.
	keys := make([]string, 0, len(c.Auths))
	for key := range c.Auths {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		if registryHost(key) == registryHost(server) {
			return c.Auths[key]
		}
	}
	return registry.AuthConfig{}
}

func decodeStoredAuth(auth registry.AuthConfig) (registry.AuthConfig, error) {
	if auth.Auth == "" || auth.Username != "" {
		return auth, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(auth.Auth)
	if err != nil {
		return registry.AuthConfig{}, fmt.Errorf("invalid Docker auth encoding")
	}
	username, password, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return registry.AuthConfig{}, fmt.Errorf("invalid Docker auth entry")
	}
	auth.Username, auth.Password = username, strings.TrimRight(password, "\x00")
	auth.Auth = ""
	return auth, nil
}

func registryHost(value string) string {
	value = strings.TrimPrefix(strings.TrimPrefix(value, "https://"), "http://")
	host, _, _ := strings.Cut(value, "/")
	return host
}
