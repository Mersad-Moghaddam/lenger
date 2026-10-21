package docker

import (
	"context"
	"fmt"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/style"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/views/inspector"
	"net/http"
	"strings"
	"testing"
)

func TestResidualOrdinaryDiagnosticSuffix(t *testing.T) {
	message := `{"context":"` + strings.Repeat("request-context ", 100) + `","error":"connection refused"}`
	e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/json") {
			_, _ = fmt.Fprint(w, `{"Config":{"Tty":true}}`)
			return
		}
		_, _ = fmt.Fprint(w, "2026-10-21T12:00:00.123456789Z "+message+"\n")
	})
	view := inspector.LogsViewModel{Follow: true, SearchQuery: "connection refused", LevelFilter: "error"}
	err := e.Logs(context.Background(), "c", engine.LogOptions{}, func(entry engine.LogEntry) error {
		if entry.Text != message || entry.At.Nanosecond() != 123456789 {
			t.Errorf("ordinary diagnostic lost: %q", entry.Text)
		}
		view.Append(entry)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// App copy uses the plain rendered viewport; exercise exactly that content.
	copied := view.Render(style.Sheet{Plain: true}, 2048, 10)
	if !strings.Contains(copied, `"error":"connection refused"`) {
		t.Fatal("search/error filter/copy lost actionable suffix")
	}
}

func TestResidualFittingRecordAndInvalidInterior(t *testing.T) {
	for _, size := range []int{501, 512, 64 * 1024} {
		var entry engine.LogEntry
		w := logWriter{emit: func(e engine.LogEntry) error { entry = e; return nil }}
		text := strings.Repeat("x", size-4) + "tail"
		if _, err := w.Write([]byte(text + "\n")); err != nil {
			t.Fatal(err)
		}
		if entry.Text != text {
			t.Fatalf("fitting %d byte record was truncated: %d", size, len(entry.Text))
		}
	}
	var entry engine.LogEntry
	w := logWriter{emit: func(e engine.LogEntry) error { entry = e; return nil }}
	text := "2026-10-21T12:00:00.123456789Z prefix\xff valid-actionable-error " + strings.Repeat("x", 128*1024)
	if _, err := w.Write([]byte(text + "\n")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(entry.Text, "prefix� valid-actionable-error") || !strings.Contains(entry.Text, "[truncated]") || entry.At.Nanosecond() != 123456789 {
		t.Fatalf("invalid interior erased valid prefix: %q", entry.Text)
	}
}
