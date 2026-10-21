package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/distribution/reference"
	"github.com/moby/moby/api/types/registry"
)

var helperNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

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
	cfg, err := loadCredentialConfig()
	if err != nil {
		return "", err
	}
	auth := cfg.authFor(server)
	helper := cfg.Helpers[server]
	if helper == "" {
		helper = cfg.Store
	}
	if helper != "" {
		auth, err = helperAuth(ctx, helper, server)
	} else {
		auth, err = decodeStoredAuth(auth)
	}
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	auth.ServerAddress = server
	buf, err := json.Marshal(auth)
	if err != nil {
		return "", fmt.Errorf("encode auth config: %w", err)
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

func helperAuth(ctx context.Context, helper, server string) (registry.AuthConfig, error) {
	var auth registry.AuthConfig
	if !helperNamePattern.MatchString(helper) {
		return auth, fmt.Errorf("invalid Docker credential helper name")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker-credential-"+helper, "get")
	cmd.Stdin = strings.NewReader(server + "\n")
	cmd.WaitDelay = time.Second
	output, err := cmd.Output()
	if ctx.Err() != nil {
		return auth, ctx.Err()
	}
	if err != nil {
		// Do not leak helper output: it may contain credentials.
		if strings.TrimSpace(string(output)) == "credentials not found in native keychain" {
			return auth, nil
		}
		return auth, fmt.Errorf("Docker credential helper failed; check docker login")
	}
	var secret struct{ Username, Secret string }
	if err := json.Unmarshal(output, &secret); err != nil {
		return auth, fmt.Errorf("invalid Docker credential helper response")
	}
	if secret.Username == "<token>" {
		auth.IdentityToken = secret.Secret
	} else {
		auth.Username, auth.Password = secret.Username, secret.Secret
	}
	return auth, nil
}
