// Package inspector contains the inspector drawer's presentation models.
package inspector

import (
	"encoding/json"
	"errors"
	"net"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
)

const envMask = "••••••••••••"

var sensitiveKey = regexp.MustCompile(`(?i)(PASSWORD|PASSWD|PASS|SECRET|KEY|TOKEN|AUTH|CREDENTIAL|PRIVATE|API_KEY)`)

// EnvItem retains the raw value only for explicit on-screen reveal. Diagnostic
// formatting and JSON serialization always redact sensitive values.
type EnvItem struct {
	Key         string
	Value       string
	IsSensitive bool
	Revealed    bool
}

type EnvViewModel struct {
	EnvItems     []EnvItem
	RevealedAll  bool
	LastActivity time.Time
}

// ParseEnv preserves order, duplicates, empty values, and embedded equals signs.
// A bare key is represented with an empty value.
func ParseEnv(raw []string) []EnvItem {
	items := make([]EnvItem, 0, len(raw))
	for _, entry := range raw {
		key, value, _ := strings.Cut(entry, "=")
		items = append(items, EnvItem{Key: key, Value: value, IsSensitive: IsSensitiveKey(key)})
	}
	return items
}

func IsSensitiveKey(key string) bool { return sensitiveKey.MatchString(key) }

func (e EnvItem) sensitive() bool { return e.IsSensitive || IsSensitiveKey(e.Key) }

func (e EnvItem) redactedValue() string {
	if e.sensitive() {
		return envMask
	}
	return style.Clean(e.Value)
}

func (e EnvItem) String() string   { return style.Clean(e.Key) + "=" + e.redactedValue() }
func (e EnvItem) GoString() string { return e.String() }

func (e EnvItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Key         string
		Value       string
		IsSensitive bool
	}{style.Clean(e.Key), e.redactedValue(), e.sensitive()})
}

// RedactedEnv is safe for diagnostic exports even while values are revealed.
// The returned slice is independent of the model.
func (m *EnvViewModel) RedactedEnv() []string {
	if m == nil {
		return nil
	}
	items := make([]string, len(m.EnvItems))
	for i, item := range m.EnvItems {
		items[i] = item.String()
	}
	return items
}

func (m EnvViewModel) String() string   { return strings.Join(m.RedactedEnv(), "\n") }
func (m EnvViewModel) GoString() string { return m.String() }

// RecordActivity should be called by the app for interactions within this tab.
func (m *EnvViewModel) RecordActivity(now time.Time) {
	if m != nil {
		m.LastActivity = now
	}
}

// RevealItem handles Space/click reveal; invalid indexes do not change activity.
func (m *EnvViewModel) RevealItem(index int, now time.Time) bool {
	if m == nil || index < 0 || index >= len(m.EnvItems) {
		return false
	}
	m.EnvItems[index].Revealed = true
	m.RecordActivity(now)
	m.updateRevealedAll()
	return true
}

func (m *EnvViewModel) RevealAll(now time.Time) {
	if m == nil {
		return
	}
	for i := range m.EnvItems {
		m.EnvItems[i].Revealed = true
	}
	m.RecordActivity(now)
	m.updateRevealedAll()
}

func (m *EnvViewModel) updateRevealedAll() {
	m.RevealedAll = len(m.EnvItems) > 0
	for _, item := range m.EnvItems {
		m.RevealedAll = m.RevealedAll && item.Revealed
	}
}

func (m *EnvViewModel) RemaskItem(index int) bool {
	if m == nil || index < 0 || index >= len(m.EnvItems) {
		return false
	}
	changed := m.EnvItems[index].Revealed
	m.EnvItems[index].Revealed = false
	m.updateRevealedAll()
	return changed
}

// RemaskAll MUST be called before closing the drawer or changing containers.
// It also clears activity so a reused model cannot retain the previous session.
func (m *EnvViewModel) RemaskAll() bool {
	if m == nil {
		return false
	}
	changed := m.RevealedAll
	for i := range m.EnvItems {
		changed = changed || m.EnvItems[i].Revealed
		m.EnvItems[i].Revealed = false
	}
	m.RevealedAll = false
	m.LastActivity = time.Time{}
	return changed
}

// CheckAutoRemask is driven by the app's timer, not rendering. It expires at
// exactly 60 seconds without activity, including models with no activity time.
func (m *EnvViewModel) CheckAutoRemask(now time.Time) bool {
	if m == nil || now.Sub(m.LastActivity) < 60*time.Second {
		return false
	}
	return m.RemaskAll()
}

// PortPreview is confirmation data only; no browser or command is executed.
// The app must obtain explicit confirmation before using URL with a browser API.
type PortPreview struct {
	URL     string
	Warning string
}

const nonLoopbackWarning = "Security: bound to a non-loopback address; this port may be reachable from other hosts."

// portExposureWarning applies even to published UDP ports with no browser URL.
func portExposureWarning(mapping string) string {
	bound, _, ok := strings.Cut(mapping, "->")
	if !ok || style.Clean(mapping) != mapping {
		return ""
	}
	host, port, err := net.SplitHostPort(strings.TrimSpace(bound))
	if err != nil || !validPort(port) {
		return ""
	}
	ip, err := netip.ParseAddr(host)
	if err == nil && !ip.Unmap().IsLoopback() {
		return nonLoopbackWarning
	}
	return ""
}

// PreviewPort accepts only Docker-style IP:port -> port/tcp mappings. Exposed
// ports and UDP are display-only. It never accepts a URL, hostname, or command.
func PreviewPort(mapping string) (PortPreview, error) {
	invalid := errors.New("port mapping has no safe HTTP preview")
	if style.Clean(mapping) != mapping {
		return PortPreview{}, invalid
	}
	bound, target, ok := strings.Cut(mapping, "->")
	if !ok {
		return PortPreview{}, invalid
	}
	portText, protocol, ok := strings.Cut(strings.TrimSpace(target), "/")
	if !ok || protocol != "tcp" || !validPort(portText) {
		return PortPreview{}, invalid
	}
	host, published, err := net.SplitHostPort(strings.TrimSpace(bound))
	if err != nil || !validPort(published) {
		return PortPreview{}, invalid
	}
	ip, err := netip.ParseAddr(host)
	if err != nil || ip.Zone() != "" || ip.IsMulticast() {
		return PortPreview{}, invalid
	}
	ip = ip.Unmap()
	p := PortPreview{}
	if !ip.IsLoopback() {
		p.Warning = nonLoopbackWarning
	}
	if ip.IsUnspecified() {
		if ip.Is4() {
			ip = netip.MustParseAddr("127.0.0.1")
		} else {
			ip = netip.IPv6Loopback()
		}
	}
	p.URL = "http://" + net.JoinHostPort(ip.String(), published)
	return p, nil
}

func validPort(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	n, err := strconv.Atoi(value)
	return err == nil && n > 0 && n <= 65535
}

func RenderEnvTab(s style.Sheet, m *EnvViewModel, ports []string, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	lines := []string{s.Heading("Environment"), s.Muted("Space: reveal row · u: reveal all · auto-remask after 60s idle")}
	if m == nil || len(m.EnvItems) == 0 {
		lines = append(lines, s.Muted("No environment variables"))
	}
	if m != nil {
		for _, item := range m.EnvItems {
			value := item.redactedValue()
			if item.Revealed {
				value = style.Clean(item.Value)
			}
			lines = append(lines, style.Clean(item.Key)+"="+value)
		}
	}
	lines = append(lines, "", s.Heading("Ports"))
	if len(ports) == 0 {
		lines = append(lines, s.Muted("No port mappings"))
	}
	for _, mapping := range ports {
		lines = append(lines, style.Clean(mapping))
		if preview, err := PreviewPort(mapping); err == nil {
			lines = append(lines, s.Muted("Confirm before opening: "+preview.URL))
			if preview.Warning != "" {
				lines = append(lines, s.Text(preview.Warning, s.Palette.Warning))
			}
		} else {
			lines = append(lines, s.Muted("Display only: no safe HTTP preview"))
			if warning := portExposureWarning(mapping); warning != "" {
				lines = append(lines, s.Text(warning, s.Palette.Warning))
			}
		}
	}
	return style.Fit(strings.Join(lines, "\n"), w, h)
}
