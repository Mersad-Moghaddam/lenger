package docker

import (
	"errors"
	"strings"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// inspectionError preserves cancellation/error matching while keeping raw daemon
// response text out of ordinary logging, debug formatting, and JSON exports.
type inspectionError struct {
	cause          error
	classification engine.ClassifiedError
}

func newInspectionError(cause error) *inspectionError {
	// Classify before redacting the public message. Retain only safe metadata in
	// the classification; the raw response remains privately available to Unwrap.
	c := *engine.Classify(cause)
	// Classify's legacy HTTP-code substring checks can mistake an endpoint's
	// ephemeral port (e.g. 40453) for a response status. Resolve explicit daemon
	// connectivity failures before those text heuristics, using only safe text.
	text := strings.ToLower(cause.Error())
	if c.Category != "permission" && c.Category != "cancelled" && c.Category != "timeout" && c.Category != "dns" &&
		(strings.Contains(text, "cannot connect") || strings.Contains(text, "connection refused") || strings.Contains(text, "is the docker daemon running")) {
		c = *engine.Classify(errDaemonUnavailable)
	}
	c.Cause = nil
	return &inspectionError{cause: cause, classification: c}
}

var errDaemonUnavailable = errors.New("cannot connect to Docker daemon")

// As preserves the existing Classify contract without exposing the raw cause in
// the classified error's public fields or ordinary diagnostic serialization.
func (e *inspectionError) As(target any) bool {
	if p, ok := target.(**engine.ClassifiedError); ok {
		c := e.classification
		c.Cause = e
		*p = &c
		return true
	}
	return false
}

func (*inspectionError) Error() string      { return "Could not inspect container." }
func (e *inspectionError) String() string   { return e.Error() }
func (e *inspectionError) GoString() string { return e.Error() }
func (e *inspectionError) Unwrap() error    { return e.cause }
