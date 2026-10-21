// Package engine defines daemon-independent resource and streaming contracts.
package engine

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Image is a local image summary and its container references.
type Image struct {
	ID string
	RepoTags, Digests []string
	Size, Shared int64
	Created time.Time
	Dangling bool
	UsedBy []Container
}

// Name returns a human-readable reference, including untagged images.
func (i Image) Name() string {
	if len(i.RepoTags) > 0 { return i.RepoTags[0] }
	return "<none>:<none>"
}

// ShortID strips the digest algorithm and truncates the identifier safely.
func (i Image) ShortID() string { return ShortID(i.ID) }

// ShortID returns at most twelve characters of a digest.
func ShortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 { return id[:12] }
	return id
}

// Container is the lightweight information needed for counts and image usage.
type Container struct {
	ID, Name, ImageID, Image, State, Status string
	Labels map[string]string
}

// Layer records an image history entry, including metadata-only entries.
type Layer struct { ID, Command string; Size int64 }

// Inspection contains structured image metadata. Services mask values by default.
type Inspection struct {
	ID, Platform string
	Env, Entrypoint, Command, Ports, Digests []string
	Labels map[string]string
	Layers []Layer
}

// Progress is one Docker pull stream update.
type Progress struct { ID, Status string; Current, Total int64 }

// Event reports a resource change or stream connection state.
type Event struct {
	ID, Kind, Action string
	At time.Time
	Err error
}

// Engine abstracts local daemon access. Stream callbacks must not be retained.
type Engine interface {
	ListImages(context.Context) ([]Image, error)
	ListContainers(context.Context) ([]Container, error)
	InspectImage(context.Context, string) (Inspection, error)
	RemoveImage(context.Context, string) error
	Pull(context.Context, string, func(Progress) error) error
	Events(context.Context, func(Event) error) error
	Close() error
}

// ErrNotImplemented identifies deliberately deferred capabilities.
type ErrNotImplemented struct { Requirement string }

func (e ErrNotImplemented) Error() string { return fmt.Sprintf("%s is not implemented yet", e.Requirement) }
