package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/distribution/reference"
	"github.com/moby/moby/api/types/registry"
)

// registryAuth reads Docker's existing credential configuration without writing it.
// Helpers run with a deadline so cancellation does not leave a pull stuck forever.
func registryAuth(ctx context.Context, ref string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	named, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return "", fmt.Errorf("invalid image reference: %w", err)
	}
	server := reference.Domain(named)
	if server == "docker.io" {
		server = "https://index.docker.io/v1/"
	}
	dir := os.Getenv("DOCKER_CONFIG")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find Docker configuration: %w", err)
		}
		dir = filepath.Join(home, ".docker")
	}
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read Docker configuration: %w", err)
	}
	var cfg struct {
		Auths   map[string]registry.AuthConfig `json:"auths"`
		Helpers map[string]string              `json:"credHelpers"`
		Store   string                         `json:"credsStore"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("decode Docker configuration: %w", err)
	}
	auth := cfg.Auths[server]
	// Docker permits auth keys with a legacy URL prefix and /v1/ suffix.
	if auth == (registry.AuthConfig{}) {
		for key, value := range cfg.Auths {
			if registryHost(key) == registryHost(server) {
				auth = value
				break
			}
		}
	}
	helper := cfg.Helpers[server]
	if helper == "" {
		helper = cfg.Store
	}
	if helper != "" {
		if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`).MatchString(helper) {
			return "", fmt.Errorf("invalid Docker credential helper name")
		}
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "docker-credential-"+helper, "get")
		cmd.Stdin = strings.NewReader(server + "\n")
		cmd.WaitDelay = time.Second
		output, err := cmd.Output()
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if err != nil {
			// This standard sentinel means an anonymous pull is appropriate.
			if strings.TrimSpace(string(output)) == "credentials not found in native keychain" {
				return "", nil
			}
			return "", fmt.Errorf("Docker credential helper failed; check docker login")
		}
		var secret struct{ Username, Secret string }
		if err := json.Unmarshal(output, &secret); err != nil {
			return "", fmt.Errorf("invalid Docker credential helper response")
		}
		auth = registry.AuthConfig{ServerAddress: server}
		if secret.Username == "<token>" {
			auth.IdentityToken = secret.Secret
		} else {
			auth.Username, auth.Password = secret.Username, secret.Secret
		}
	} else if auth.Auth != "" && auth.Username == "" {
		decoded, err := base64.StdEncoding.DecodeString(auth.Auth)
		if err != nil {
			return "", fmt.Errorf("invalid Docker auth encoding")
		}
		username, password, ok := strings.Cut(string(decoded), ":")
		if !ok {
			return "", fmt.Errorf("invalid Docker auth entry")
		}
		auth.Username, auth.Password = username, strings.TrimRight(password, "\x00")
		auth.Auth = ""
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	auth.ServerAddress = server
	return registry.EncodeAuthConfig(auth)
}

func registryHost(value string) string {
	value = strings.TrimPrefix(strings.TrimPrefix(value, "https://"), "http://")
	host, _, _ := strings.Cut(value, "/")
	return host
}
