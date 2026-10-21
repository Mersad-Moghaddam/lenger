package engine

import "testing"

func TestDiagnosticAttachmentDetached(t *testing.T) {
	memory := uint64(0)
	source := DiagnosticEvidence{ConfiguredMemory: &memory, HealthConfig: &HealthCheckConfig{Command: []string{"original"}}, HealthHistory: []HealthProbe{{Output: "original"}}}
	i := (ContainerInspection{}).WithDiagnosticEvidence(source)
	memory = 10
	source.HealthConfig.Command[0] = "changed"
	source.HealthHistory[0].Output = "changed"
	d := i.DiagnosticEvidence()
	if *d.ConfiguredMemory != 0 || d.HealthConfig.Command[0] != "original" || d.HealthHistory[0].Output != "original" {
		t.Fatal("attachment retained mutable input")
	}
}
