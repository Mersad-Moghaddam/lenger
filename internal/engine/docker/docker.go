// Package docker adapts the official Moby client to Lenger's engine interface.
package docker

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/docker/go-connections/sockets"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/moby/moby/client"
)

// Engine owns one API-negotiating Moby client.
type Engine struct {
	client    *client.Client
	transport *failedResponseTransport
}

// New respects DOCKER_HOST, TLS settings and the standard proxy environment.
func New() (*Engine, error) {
	host := os.Getenv(client.EnvOverrideHost)
	if host == "" {
		host = client.DefaultDockerHost
	}
	u, err := client.ParseHostURL(host)
	if err != nil {
		return nil, engine.Classify(err)
	}
	transport := &http.Transport{}
	if err := sockets.ConfigureTransport(transport, u.Scheme, u.Host); err != nil {
		return nil, engine.Classify(err)
	}
	if certPath := os.Getenv(client.EnvOverrideCertPath); certPath != "" {
		config, err := tlsconfig.Client(tlsconfig.Options{
			CAFile: filepath.Join(certPath, "ca.pem"), CertFile: filepath.Join(certPath, "cert.pem"), KeyFile: filepath.Join(certPath, "key.pem"),
			InsecureSkipVerify: os.Getenv(client.EnvTLSVerify) == "", MinVersion: tls.VersionTLS12,
		})
		if err != nil {
			return nil, engine.Classify(err)
		}
		transport.TLSClientConfig = config
	}
	wrapped := &failedResponseTransport{base: transport}
	opts := []client.Opt{client.WithHost(host), client.WithAPIVersionFromEnv(), client.WithHTTPClient(&http.Client{Transport: wrapped, CheckRedirect: client.CheckRedirect})}
	if transport.TLSClientConfig != nil {
		opts = append(opts, client.WithScheme("https"))
	}
	c, err := client.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("create Docker client: %w", err)
	}
	return &Engine{client: c, transport: wrapped}, nil
}

// Moby consumes failed HTTP response bodies internally without returning them
// to the adapter. Close those bodies when its error decoder finishes reading.
type failedResponseTransport struct{ base http.RoundTripper }

func (t *failedResponseTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.base.RoundTrip(request)
	if err != nil {
		return response, err
	}
	if response.StatusCode >= http.StatusBadRequest && response.Body != nil {
		response.Body = &failedResponseBody{ReadCloser: response.Body}
	}
	return response, nil
}

func (t *failedResponseTransport) CloseIdleConnections() {
	if closer, ok := t.base.(interface{ CloseIdleConnections() }); ok {
		closer.CloseIdleConnections()
	}
}

type failedResponseBody struct {
	io.ReadCloser
	read   int
	closed bool
}

func (b *failedResponseBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.read += n
	// Moby limits error decoding to 1 MiB; it may not read EOF at that limit.
	if err != nil || b.read >= 1<<20 {
		_ = b.Close()
	}
	return n, err
}

func (b *failedResponseBody) Close() error {
	if b.closed {
		return nil
	}
	b.closed = true
	return b.ReadCloser.Close()
}

// ListImages reads all local image summaries.
func (e *Engine) ListImages(ctx context.Context) ([]engine.Image, error) {
	result, err := e.client.ImageList(ctx, client.ImageListOptions{All: true, SharedSize: true})
	if err != nil {
		return nil, fmt.Errorf("list images: %w", err)
	}
	items := make([]engine.Image, 0, len(result.Items))
	for _, i := range result.Items {
		tags := i.RepoTags
		if len(tags) == 1 && tags[0] == "<none>:<none>" {
			tags = nil
		}
		items = append(items, engine.Image{ID: i.ID, RepoTags: tags, Digests: i.RepoDigests, Size: i.Size, Shared: i.SharedSize, Created: time.Unix(i.Created, 0), Dangling: len(tags) == 0})
	}
	return items, nil
}

// ListContainers includes stopped containers when computing image usage.
func (e *Engine) ListContainers(ctx context.Context) ([]engine.Container, error) {
	result, err := e.client.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	items := make([]engine.Container, 0, len(result.Items))
	for _, c := range result.Items {
		name := engine.ShortID(c.ID)
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		health := ""
		if c.Health != nil {
			health = string(c.Health.Status)
		}
		ports := make([]string, 0, len(c.Ports))
		for _, p := range c.Ports {
			ports = append(ports, formatPortSummary(p))
		}
		mounts := make([]engine.Mount, 0, len(c.Mounts))
		for _, m := range c.Mounts {
			mounts = append(mounts, engine.Mount{Type: string(m.Type), Name: m.Name, Source: m.Source, Destination: m.Destination, Mode: m.Mode, RW: m.RW})
		}
		items = append(items, engine.Container{
			ID: c.ID, Name: name, ImageID: c.ImageID, Image: c.Image,
			State: string(c.State), Status: c.Status, Health: health,
			Created: time.Unix(c.Created, 0), Ports: ports, Mounts: mounts,
			Project: c.Labels["com.docker.compose.project"], Service: c.Labels["com.docker.compose.service"],
			ProjectDir:         c.Labels["com.docker.compose.project.working_dir"],
			ProjectConfigFiles: splitCommaOrEmpty(c.Labels["com.docker.compose.project.config_files"]),
			Labels:             c.Labels,
		})
	}
	return items, nil
}

func splitCommaOrEmpty(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}

// InspectImage joins config and history for the image details pane.
func (e *Engine) InspectImage(ctx context.Context, id string) (engine.Inspection, error) {
	i, err := e.client.ImageInspect(ctx, id)
	if err != nil {
		return engine.Inspection{}, fmt.Errorf("inspect image: %w", err)
	}
	out := engine.Inspection{ID: i.ID, Platform: i.Os + "/" + i.Architecture, Digests: i.RepoDigests}
	if i.Config != nil {
		out.Env = i.Config.Env
		out.Entrypoint = i.Config.Entrypoint
		out.Command = i.Config.Cmd
		out.Labels = i.Config.Labels
		for port := range i.Config.ExposedPorts {
			out.Ports = append(out.Ports, string(port))
		}
	}
	h, err := e.client.ImageHistory(ctx, id)
	if err != nil {
		return engine.Inspection{}, fmt.Errorf("image history: %w", err)
	}
	for _, l := range h.Items {
		out.Layers = append(out.Layers, engine.Layer{ID: l.ID, Command: l.CreatedBy, Size: l.Size})
	}
	return out, nil
}

// RemoveImage never forces deletion or prunes parent images implicitly.
func (e *Engine) RemoveImage(ctx context.Context, id string) error {
	_, err := e.client.ImageRemove(ctx, id, client.ImageRemoveOptions{Force: false, PruneChildren: false})
	if err != nil {
		return fmt.Errorf("remove image: %w", err)
	}
	return nil
}

// Pull streams progress without retaining a response body after completion.
func (e *Engine) Pull(ctx context.Context, ref string, emit func(engine.Progress) error) error {
	auth, err := registryAuth(ctx, ref)
	if err != nil {
		return fmt.Errorf("resolve registry credentials: %w", err)
	}
	body, err := e.client.ImagePull(ctx, ref, client.ImagePullOptions{RegistryAuth: auth})
	if err != nil {
		return fmt.Errorf("pull image: %w", err)
	}
	defer body.Close() //nolint:errcheck // The decoder/context error is authoritative.
	return DecodePull(ctx, body, emit)
}

// Events reconnects with cancellable exponential backoff and a replay cursor.
func (e *Engine) Events(ctx context.Context, emit func(engine.Event) error) error {
	backoff := time.Second
	since := ""
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		streamCtx, cancel := context.WithCancel(ctx)
		stream := e.client.Events(streamCtx, client.EventsListOptions{Since: since})
		if err := emit(engine.Event{Kind: "connection", Action: "connecting", At: time.Now()}); err != nil {
			cancel()
			return err
		}
		var streamErr error
	loop:
		for {
			select {
			case <-ctx.Done():
				cancel()
				return ctx.Err()
			case ev, ok := <-stream.Messages:
				if !ok {
					streamErr = fmt.Errorf("event stream closed")
					break loop
				}
				backoff = time.Second
				at := time.Unix(0, ev.TimeNano)
				since = at.Format(time.RFC3339Nano)
				if err := emit(engine.Event{ID: ev.Actor.ID, Kind: string(ev.Type), Action: string(ev.Action), At: at}); err != nil {
					cancel()
					return err
				}
			case err, ok := <-stream.Err:
				if !ok || err == nil {
					streamErr = fmt.Errorf("event stream ended")
				} else {
					streamErr = err
				}
				break loop
			}
		}
		cancel()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := emit(engine.Event{Kind: "connection", Action: "reconnecting", At: time.Now(), Err: engine.Classify(streamErr)}); err != nil {
			return err
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		backoff = min(30*time.Second, backoff*2)
	}
}

// Close closes idle transport connections.
func (e *Engine) Close() error {
	if e.transport != nil {
		e.transport.CloseIdleConnections()
	}
	return e.client.Close()
}
