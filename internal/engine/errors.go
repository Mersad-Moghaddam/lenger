package engine

import (
	"context"
	"errors"
	"net"
	"strings"
)

// ClassifiedError provides safe, actionable failure text without credentials.
type ClassifiedError struct {
	Category, Message, Suggestion string
	Cause                         error
}

func (e *ClassifiedError) Error() string { return e.Message }

// Unwrap preserves cancellation and SDK error matching.
func (e *ClassifiedError) Unwrap() error { return e.Cause }

// Classify maps daemon/registry errors to REG-01 categories.
func Classify(err error) *ClassifiedError {
	if err == nil {
		return nil
	}
	var known *ClassifiedError
	if errors.As(err, &known) {
		return known
	}
	text := strings.ToLower(err.Error())
	category, message, next := "unknown", "The operation failed.", "Retry, then check Docker's daemon logs."
	var dns *net.DNSError
	switch {
	case errors.Is(err, context.Canceled):
		category, message, next = "cancelled", "Operation cancelled.", "Retry when ready."
	case strings.Contains(text, "permission denied"):
		category, message, next = "permission", "Docker socket access was denied.", "sudo usermod -aG docker \"$USER\"; then log out and back in. The docker group grants root-equivalent access."
	case strings.Contains(text, "no such container"), strings.Contains(text, "container not found"):
		category, message, next = "not-found", "The container was not found.", "Refresh the list; it may have been removed by another client."
	case strings.Contains(text, "no such volume"), strings.Contains(text, "volume not found"):
		category, message, next = "not-found", "The volume was not found.", "Refresh the list; it may have been removed by another client."
	case strings.Contains(text, "no such network"), strings.Contains(text, "network not found"):
		category, message, next = "not-found", "The network was not found.", "Refresh the list; it may have been removed by another client."
	case strings.Contains(text, "has active endpoints"), strings.Contains(text, "network has active endpoints"):
		category, message, next = "in-use", "The network still has active container endpoints attached.", "Disconnect all containers before removing this network."
	case strings.Contains(text, "docker compose"), strings.Contains(text, "compose executable"), strings.Contains(text, "executable file not found in $path"):
		category, message, next = "compose", "Docker Compose CLI is not installed or accessible in PATH.", "Install the docker-compose-plugin package or ensure 'docker compose' is installed."
	case strings.Contains(text, "already in use"), strings.Contains(text, "already allocated"), strings.Contains(text, "address already in use"):
		category, message, next = "conflict", "That name or port is already in use.", "Choose another name or host port, or remove the existing resource."
	case strings.Contains(text, "is in use by"), strings.Contains(text, "volume is in use"), strings.Contains(text, "cannot remove a running container"):
		category, message, next = "in-use", "The resource is still in use.", "Stop or remove the dependent resource first."
	case strings.Contains(text, "is not running"), strings.Contains(text, "container is paused"):
		category, message, next = "state", "The container is not in a state that allows this action.", "Refresh the list and choose an action valid for its current state."
	case strings.Contains(text, "429"), strings.Contains(text, "toomanyrequests"), strings.Contains(text, "rate limit"):
		category, message, next = "rate-limit", "The registry rate limit was reached.", "Wait before retrying or use docker login."
	case strings.Contains(text, "unauthorized"), strings.Contains(text, "authentication required"), strings.Contains(text, "401"):
		category, message, next = "auth", "Registry authentication is required.", "Use docker login for the image registry, then retry."
	case strings.Contains(text, "403"), strings.Contains(text, "forbidden"), strings.Contains(text, "blocked"):
		category, message, next = "blocked", "The registry refused this request.", "Check registry policy, proxy settings and network access."
	case strings.Contains(text, "manifest unknown"), strings.Contains(text, "not found"), strings.Contains(text, "404"):
		category, message, next = "not-found", "The image or tag was not found.", "Check the repository name and tag."
	case strings.Contains(text, "x509"), strings.Contains(text, "tls"):
		category, message, next = "tls", "The registry TLS connection failed.", "Check the system clock and trusted CA certificates; do not disable TLS verification."
	case errors.As(err, &dns), strings.Contains(text, "no such host"):
		category, message, next = "dns", "The registry hostname could not be resolved.", "Check DNS and network connectivity."
	case errors.Is(err, context.DeadlineExceeded), strings.Contains(text, "timeout"), strings.Contains(text, "deadline exceeded"):
		category, message, next = "timeout", "The registry request timed out.", "Check connectivity and HTTP_PROXY / HTTPS_PROXY, then retry."
	case strings.Contains(text, "cannot connect"), strings.Contains(text, "connection refused"), strings.Contains(text, "is the docker daemon running"):
		category, message, next = "daemon", "Docker is not reachable.", "sudo systemctl start docker; then press R to reconnect."
	}
	return &ClassifiedError{Category: category, Message: message, Suggestion: next, Cause: err}
}
