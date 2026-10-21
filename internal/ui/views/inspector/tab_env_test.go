package inspector

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
)

func TestEnvSensitiveKeys(t *testing.T) {
	for _, key := range []string{"AWS_SECRET_ACCESS_KEY", "db_password", "TOKEN", "auth_bearer", "Api_Key", "PRIVATE_CERT", "PASSWD", "PASS", "Auth_Token", "API_Secret", "CREDENTIALS"} {
		if !IsSensitiveKey(key) {
			t.Errorf("not sensitive: %s", key)
		}
	}
	for _, key := range []string{"HOST", "PORT", "PATH", "LOG_LEVEL"} {
		if IsSensitiveKey(key) {
			t.Errorf("unexpected sensitive: %s", key)
		}
	}
}

func TestEnvParseAndDefaultMask(t *testing.T) {
	items := ParseEnv([]string{"TOKEN=secret=a=b", "HOST=example", "EMPTY=", "BARE"})
	if len(items) != 4 || items[0].Value != "secret=a=b" || items[3].Key != "BARE" || items[3].Value != "" {
		t.Fatalf("incorrect parse")
	}
	m := &EnvViewModel{EnvItems: items}
	out := RenderEnvTab(style.Sheet{Plain: true}, m, nil, 100, 15)
	if strings.Contains(out, "secret=a=b") || !strings.Contains(out, "••••••••••••") || !strings.Contains(out, "HOST=example") {
		t.Fatalf("incorrect masked render: %s", out)
	}
}

func TestEnvRevealAndTimeout(t *testing.T) {
	now := time.Unix(100, 0)
	m := &EnvViewModel{EnvItems: ParseEnv([]string{"TOKEN=hidden", "PASSWD=other"})}
	if !m.RevealItem(0, now) || !m.EnvItems[0].Revealed || m.RevealedAll {
		t.Fatal("individual reveal failed")
	}
	if m.RevealItem(-1, now) || m.RevealItem(10, now) {
		t.Fatal("invalid index accepted")
	}
	if m.CheckAutoRemask(now.Add(60*time.Second - time.Nanosecond)) {
		t.Fatal("early remask")
	}
	if !m.CheckAutoRemask(now.Add(60*time.Second)) || m.EnvItems[0].Revealed {
		t.Fatal("60s remask failed")
	}
	if m.CheckAutoRemask(now.Add(time.Minute * 2)) {
		t.Fatal("reported nonexistent change")
	}
	m.RevealAll(now)
	if !m.RevealedAll || !m.EnvItems[1].Revealed {
		t.Fatal("bulk reveal failed")
	}
	if !strings.Contains(RenderEnvTab(style.Sheet{Plain: true}, m, nil, 100, 12), "TOKEN=hidden") {
		t.Fatal("reveal not rendered")
	}
	m.RecordActivity(now.Add(30 * time.Second))
	if m.CheckAutoRemask(now.Add(60 * time.Second)) {
		t.Fatal("interaction did not extend timeout")
	}
	if !m.RemaskItem(0) || m.RevealedAll || m.EnvItems[0].Revealed {
		t.Fatal("individual remask failed")
	}
	if !m.RemaskAll() || m.EnvItems[1].Revealed {
		t.Fatal("drawer/container remask failed")
	}
}

func TestEnvSafeExports(t *testing.T) {
	m := &EnvViewModel{EnvItems: ParseEnv([]string{"TOKEN=topsecret", "HOST=example"})}
	m.RevealAll(time.Now())
	exported := m.RedactedEnv()
	if strings.Contains(strings.Join(exported, "\n"), "topsecret") || exported[0] != "TOKEN=••••••••••••" {
		t.Fatal("export leaked")
	}
	if strings.Contains(fmt.Sprint(m), "topsecret") || strings.Contains(fmt.Sprintf("%+v", m.EnvItems[0]), "topsecret") {
		t.Fatal("default formatting leaked")
	}
	encoded, err := json.Marshal(m)
	if err != nil || strings.Contains(string(encoded), "topsecret") || strings.Contains(fmt.Sprintf("%#v", m), "topsecret") {
		t.Fatal("diagnostic serialization leaked")
	}
	if m.EnvItems[0].Value != "topsecret" {
		t.Fatal("export mutated source")
	}
}

func TestEnvPortsPreview(t *testing.T) {
	for _, tt := range []struct {
		mapping, url string
		warning      bool
	}{
		{"0.0.0.0:8080 -> 80/tcp", "http://127.0.0.1:8080", true},
		{"127.0.0.1:8080 -> 80/tcp", "http://127.0.0.1:8080", false},
		{"192.0.2.1:443 -> 443/tcp", "http://192.0.2.1:443", true},
		{"[::]:8080 -> 80/tcp", "http://[::1]:8080", true},
		{"[::1]:8080 -> 80/tcp", "http://[::1]:8080", false},
	} {
		p, err := PreviewPort(tt.mapping)
		if err != nil || p.URL != tt.url || (p.Warning != "") != tt.warning {
			t.Errorf("preview %s: %+v %v", tt.mapping, p, err)
		}
		out := RenderEnvTab(style.Sheet{Plain: true}, nil, []string{tt.mapping}, 120, 15)
		if !strings.Contains(out, tt.mapping) || !strings.Contains(out, tt.url) || !strings.Contains(out, "Confirm") {
			t.Errorf("missing preview: %s", out)
		}
		if tt.warning && !strings.Contains(out, "Security") {
			t.Error("missing security note")
		}
	}
	for _, raw := range []string{"80/tcp", "127.0.0.1:53 -> 53/udp", "javascript:alert(1)", "http://evil:8080 -> 80/tcp", "127.0.0.1:0 -> 80/tcp", "127.0.0.1:65536 -> 80/tcp", "127.0.0.1:8080;sh -> 80/tcp", "127.0.0.1:8080 -> secret", "127.0.0.1:8080\n -> 80/tcp"} {
		p, err := PreviewPort(raw)
		if err == nil || p.URL != "" || strings.Contains(err.Error(), raw) {
			t.Errorf("unsafe input accepted/leaked: %q", raw)
		}
	}
}

func TestEnvRenderBoundsAndControls(t *testing.T) {
	m := &EnvViewModel{EnvItems: ParseEnv([]string{"HOST=abc\x1b[31m\nnext"})}
	out := RenderEnvTab(style.Sheet{Plain: true}, m, nil, 40, 10)
	if strings.Contains(out, "\x1b") {
		t.Fatal("terminal control survived")
	}
	if RenderEnvTab(style.Sheet{}, m, nil, 0, 10) != "" {
		t.Fatal("zero width")
	}
}

func TestEnvUDPExposureWarning(t *testing.T) {
	out := RenderEnvTab(style.Sheet{Plain: true}, nil, []string{"0.0.0.0:53 -> 53/udp", "80/tcp"}, 120, 15)
	if !strings.Contains(out, "Security: bound to a non-loopback") || strings.Contains(out, "http://") || !strings.Contains(out, "80/tcp") {
		t.Fatalf("UDP/exposed port presentation: %s", out)
	}
}
