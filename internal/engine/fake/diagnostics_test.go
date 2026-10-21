package fake

import (
	"context"
	"testing"
)

func TestInspectDiagnosticEvidence(t *testing.T) {
	e := New()
	containers, _ := e.ListContainers(context.Background())
	for _, c := range containers {
		i, err := e.InspectContainer(context.Background(), c.ID)
		if err != nil {
			t.Fatal(err)
		}
		d := i.DiagnosticEvidence()
		if i.InspectedAt.IsZero() || !d.DaemonErrorKnown || d.ConfiguredMemory == nil {
			t.Fatal("missing fake inspect evidence")
		}
		if c.Health != "" {
			if !d.HealthHistoryKnown || !d.ProbeExecutionObserved || len(d.HealthHistory) != 1 || d.HealthConfig == nil {
				t.Fatal("missing fake health evidence")
			}
			if d.HealthHistory[0].Status != "" || d.HealthHistory[0].FailingStreak != 0 || i.HealthCheck.Status != c.Health {
				t.Fatal("fake history invents aggregate health")
			}
			d.HealthConfig.Command[0] = "changed"
			d.HealthHistory[0].Output = "changed"
			if i.DiagnosticEvidence().HealthConfig.Command[0] == "changed" || i.DiagnosticEvidence().HealthHistory[0].Output == "changed" {
				t.Fatal("aliased fake evidence")
			}
		}
	}
}
