package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

func TestInspectDiagnosticEvidenceContract(t *testing.T) {
	e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"Config":{"Cmd":["command-secret"],"Entrypoint":["command-secret"]},"State":{"Health":{"Status":"healthy","Log":[{"Start":"2026-10-21T01:00:00Z","End":"2026-10-21T01:00:01Z","ExitCode":0,"Output":"probe-secret"}]}}}`)
	})
	i, err := e.InspectContainer(context.Background(), "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.ValueOf(i).MethodByName("DiagnosticEvidence").IsValid() {
		t.Error("inspect lacks explicit DiagnosticEvidence accessor")
	}
	for _, value := range []any{i, i.HealthCheck} {
		for _, format := range []string{"%v", "%+v", "%#v"} {
			if strings.Contains(fmt.Sprintf(format, value), "probe-secret") {
				t.Errorf("%s leaks raw probe output", format)
			}
		}
		b, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "probe-secret") || strings.Contains(string(b), "command-secret") {
			t.Error("JSON leaks raw probe output")
		}
	}
}

func TestInspectDiagnosticBoundedHistory(t *testing.T) {
	var entries []string
	for n := 0; n < engine.MaxHealthHistory+3; n++ {
		entries = append(entries, fmt.Sprintf(`{"Start":"2026-10-21T01:00:00Z","ExitCode":%d}`, n))
	}
	e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"State":{"Health":{"Log":[%s]}}}`, strings.Join(entries, ","))
	})
	i, err := e.InspectContainer(context.Background(), "fixture")
	if err != nil {
		t.Fatal(err)
	}
	d := i.DiagnosticEvidence()
	if len(d.HealthHistory) != engine.MaxHealthHistory || d.HealthHistory[0].ExitCode != 3 || i.HealthCheck.ExitCode != engine.MaxHealthHistory+2 {
		t.Fatal("history is not bounded latest daemon entries")
	}
}

func TestInspectDiagnosticErrorSafe(t *testing.T) {
	e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		fmt.Fprint(w, `{"message":"daemon-secret command-secret"}`)
	})
	_, err := e.InspectContainer(context.Background(), "fixture")
	if err == nil {
		t.Fatal("expected inspect failure")
	}
	for _, format := range []string{"%v", "%+v", "%#v"} {
		if strings.Contains(fmt.Sprintf(format, err), "secret") {
			t.Error("inspect error exposes raw daemon response")
		}
	}
}

func TestInspectDiagnosticMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, payload                            string
		memory, state, config, history, executed bool
		count                                    int
	}{
		{"nil objects", `{}`, false, false, false, false, false, 0},
		{"nil nested", `{"State":null,"HostConfig":null,"Config":null}`, false, false, false, false, false, 0},
		{"configured never executed", `{"HostConfig":{"Memory":0},"Config":{"Healthcheck":{"Test":["CMD-SHELL","command-secret"],"Interval":1000000000,"Timeout":2000000000,"StartPeriod":3000000000,"Retries":4}},"State":{"Error":"daemon-secret","Health":{"Status":"starting","Log":[]}}}`, true, true, true, true, false, 0},
		{"executed zero", `{"HostConfig":{"Memory":1024},"State":{"Error":"","Health":{"Status":"healthy","Log":[{"Start":"2026-10-21T01:00:00Z","End":"2026-10-21T01:00:01Z","ExitCode":0,"Output":"probe-secret"}]}}}`, true, true, false, true, true, 1},
		{"multiple chronological", `{"State":{"Health":{"Status":"healthy","Log":[null,{"Start":"2026-10-21T01:00:00Z","End":"2026-10-21T01:00:01Z","ExitCode":1,"Output":"probe-secret"},{"Start":"2026-10-21T01:01:00Z","End":"2026-10-21T01:01:01Z","ExitCode":0}]}}}`, false, true, false, true, true, 2},
		{"nil history", `{"State":{"Health":{"Status":"starting","Log":null}}}`, false, true, false, true, false, 0},
		{"zero probe without timestamps", `{"State":{"Health":{"Status":"starting","Log":[{"ExitCode":0}]}}}`, false, true, false, true, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := testAdapter(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, tc.payload)
			})
			before := time.Now()
			i, err := e.InspectContainer(context.Background(), "fixture")
			if err != nil {
				t.Fatal(err)
			}
			after := time.Now()
			d := i.DiagnosticEvidence()
			if i.InspectedAt.Before(before) || i.InspectedAt.After(after) {
				t.Error("not genuine inspect completion")
			}
			if (d.ConfiguredMemory != nil) != tc.memory || d.DaemonErrorKnown != tc.state || (d.HealthConfig != nil) != tc.config || d.HealthHistoryKnown != tc.history || d.ProbeExecutionObserved != tc.executed || len(d.HealthHistory) != tc.count {
				t.Fatalf("invalid evidence flags: memory=%v state=%v config=%v history=%v executed=%v count=%d", d.ConfiguredMemory != nil, d.DaemonErrorKnown, d.HealthConfig != nil, d.HealthHistoryKnown, d.ProbeExecutionObserved, len(d.HealthHistory))
			}
			if tc.config {
				if *d.ConfiguredMemory != 0 || d.DaemonError != "daemon-secret" || d.HealthConfig.Command[1] != "command-secret" || d.HealthConfig.Interval != time.Second || d.HealthConfig.Timeout != 2*time.Second || d.HealthConfig.StartPeriod != 3*time.Second || d.HealthConfig.Retries != 4 {
					t.Error("lost configured values")
				}
				d.HealthConfig.Command[0] = "mutated"
				*d.ConfiguredMemory = 99
				if i.DiagnosticEvidence().HealthConfig.Command[0] != "CMD-SHELL" || *i.DiagnosticEvidence().ConfiguredMemory != 0 {
					t.Error("accessor aliases config")
				}
			}
			if tc.count > 0 && tc.executed {
				v := reflect.ValueOf(d.HealthHistory[0]).FieldByName("ExecutionObserved")
				if !v.IsValid() || !v.Bool() {
					t.Error("recorded probe lacks execution validity")
				}
				if !d.HealthHistory[0].Start.Equal(time.Date(2026, 10, 21, 1, 0, 0, 0, time.UTC)) || d.HealthHistory[0].Output != "probe-secret" {
					t.Error("lost daemon probe")
				}
				last := d.HealthHistory[tc.count-1]
				if i.HealthCheck == nil || i.HealthCheck.ExitCode != last.ExitCode || !i.HealthCheck.Start.Equal(last.Start) {
					t.Error("latest compatibility lost")
				}
				d.HealthHistory[0].Output = "mutated"
				if i.DiagnosticEvidence().HealthHistory[0].Output != "probe-secret" {
					t.Error("history aliases")
				}
				i.HealthCheck.Output = "compatibility mutated"
				if i.DiagnosticEvidence().HealthHistory[tc.count-1].Output == "compatibility mutated" {
					t.Error("latest aliases history")
				}
			}
			for _, value := range []any{i, d, d.HealthConfig, d.HealthHistory} {
				for _, format := range []string{"%v", "%+v", "%#v"} {
					s := fmt.Sprintf(format, value)
					for _, secret := range []string{"daemon-secret", "command-secret", "probe-secret"} {
						if strings.Contains(s, secret) {
							t.Errorf("format %s leaked %s", format, secret)
						}
					}
				}
				b, _ := json.Marshal(value)
				for _, secret := range []string{"daemon-secret", "command-secret", "probe-secret"} {
					if strings.Contains(string(b), secret) {
						t.Error("JSON leaked raw evidence")
					}
				}
			}
		})
	}
}
