package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/distribution/reference"
	"github.com/moby/moby/client"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// SearchRegistry queries Docker registries for images matching the query term (IMG-06).
func (e *Engine) SearchRegistry(ctx context.Context, query string) ([]engine.RegistrySearchResult, error) {
	auth, _ := registryAuth(ctx, query)
	results, err := e.client.ImageSearch(ctx, query, client.ImageSearchOptions{
		Limit:        30,
		RegistryAuth: auth,
	})
	if err != nil {
		return nil, engine.Classify(fmt.Errorf("search registry: %w", err))
	}
	out := make([]engine.RegistrySearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, engine.RegistrySearchResult{
			Name:        r.Name,
			Description: r.Description,
			Stars:       r.StarCount,
			Official:    r.IsOfficial,
			Automated:   r.IsAutomated,
		})
	}
	return out, nil
}

// ListRemoteTags fetches published tags for a repository via OCI/Docker Hub APIs (IMG-06).
func (e *Engine) ListRemoteTags(ctx context.Context, repo string) ([]engine.RemoteTag, error) {
	named, err := reference.ParseNormalizedNamed(repo)
	if err != nil {
		return nil, fmt.Errorf("invalid repository: %w", err)
	}
	domain := reference.Domain(named)
	path := reference.Path(named)

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if domain == "docker.io" {
		return listDockerHubTags(reqCtx, path)
	}
	return listOCITags(reqCtx, domain, path)
}

func listDockerHubTags(ctx context.Context, repoPath string) ([]engine.RemoteTag, error) {
	url := fmt.Sprintf("https://registry.hub.docker.com/v2/repositories/%s/tags?page_size=50", repoPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, engine.Classify(fmt.Errorf("hub tag query: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, engine.Classify(fmt.Errorf("hub tag query failed with HTTP %d", resp.StatusCode))
	}

	var payload struct {
		Results []struct {
			Name        string    `json:"name"`
			FullSize    int64     `json:"full_size"`
			LastUpdated time.Time `json:"last_updated"`
			Digest      string    `json:"digest"`
			Images      []struct {
				Digest string `json:"digest"`
				Size   int64  `json:"size"`
			} `json:"images"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode hub tags: %w", err)
	}

	tags := make([]engine.RemoteTag, 0, len(payload.Results))
	for _, item := range payload.Results {
		digest := item.Digest
		if digest == "" && len(item.Images) > 0 {
			digest = item.Images[0].Digest
		}
		tags = append(tags, engine.RemoteTag{
			Name:      item.Name,
			Digest:    digest,
			Size:      item.FullSize,
			UpdatedAt: item.LastUpdated,
		})
	}
	return tags, nil
}

func listOCITags(ctx context.Context, domain, path string) ([]engine.RemoteTag, error) {
	url := fmt.Sprintf("https://%s/v2/%s/tags/list?n=50", domain, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, engine.Classify(fmt.Errorf("oci tag query: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, engine.Classify(fmt.Errorf("oci tag query returned HTTP %d", resp.StatusCode))
	}

	var payload struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode oci tags: %w", err)
	}

	tags := make([]engine.RemoteTag, 0, len(payload.Tags))
	for _, t := range payload.Tags {
		tags = append(tags, engine.RemoteTag{
			Name:      t,
			UpdatedAt: time.Now(),
		})
	}
	return tags, nil
}

// CheckImageUpdate checks remote distribution manifest to compare with local digest (IMG-09).
func (e *Engine) CheckImageUpdate(ctx context.Context, ref, localDigest string) (engine.ImageUpdateStatus, error) {
	auth, _ := registryAuth(ctx, ref)
	dist, err := e.client.DistributionInspect(ctx, ref, auth)
	if err != nil {
		return engine.ImageUpdateStatus{
			Ref:         ref,
			LocalDigest: localDigest,
			CheckedAt:   time.Now(),
		}, engine.Classify(fmt.Errorf("check image update: %w", err))
	}

	remoteDigest := string(dist.Descriptor.Digest)
	hasUpdate := false
	if remoteDigest != "" && localDigest != "" {
		hasUpdate = !strings.Contains(localDigest, remoteDigest) && !strings.Contains(remoteDigest, localDigest)
	}

	return engine.ImageUpdateStatus{
		Ref:          ref,
		LocalDigest:  localDigest,
		RemoteDigest: remoteDigest,
		HasUpdate:    hasUpdate,
		CheckedAt:    time.Now(),
	}, nil
}
