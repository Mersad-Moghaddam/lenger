package views

import (
	"strings"
	"testing"
	"time"

	"github.com/Mersad-Moghaddam/lenger/internal/core/store"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

func TestProjectsViewRendersServicesAndStreamLogs(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	project := engine.ComposeProject{
		Name:        "storefront",
		WorkingDir:  "/opt/storefront",
		ConfigFiles: []string{"/opt/storefront/docker-compose.yml"},
		Services:    []string{"db", "web"},
		Status:      "running",
		Containers: []engine.Container{
			{Name: "storefront-web-1", Service: "web", State: "running", Ports: []string{"8080:80"}},
			{Name: "storefront-db-1", Service: "db", State: "running", Ports: []string{"5432:5432"}},
		},
	}
	logs := []string{
		"[+] Running 2/2",
		" Container storefront-web-1  Started",
		" Container storefront-db-1   Started",
	}

	text := Projects(plainSheet(), []engine.ComposeProject{project}, 0, "", logs, 120, 30, now)
	for _, want := range []string{
		"storefront",
		"running",
		"2 svcs",
		"/opt/storefront",
		"SERVICES (2)",
		"web",
		"db",
		"COMPOSE STREAM LOGS",
		"Container storefront-web-1  Started",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("projects view missing %q:\n%s", want, text)
		}
	}
}

func TestNetworksViewRendersIPAMAndEndpoints(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	net := engine.Network{
		ID:         "net-0123456789ab",
		Name:       "storefront_default",
		Driver:     "bridge",
		Scope:      "local",
		Attachable: true,
		Subnets:    []string{"172.28.0.0/16"},
		Gateways:   []string{"172.28.0.1"},
		Created:    now.Add(-2 * time.Hour),
		Containers: []engine.NetworkEndpoint{
			{ContainerID: "cid1", Name: "storefront-web-1", IPv4: "172.28.0.2/16"},
		},
	}

	text := Networks(plainSheet(), []engine.Network{net}, 0, "", 120, 30, now)
	for _, want := range []string{
		"storefront_default",
		"bridge",
		"1 endpoints",
		"IPAM CONFIGURATION",
		"172.28.0.0/16",
		"172.28.0.1",
		"CONNECTED CONTAINERS (1)",
		"storefront-web-1",
		"172.28.0.2/16",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("networks view missing %q:\n%s", want, text)
		}
	}
}

func TestContainerHealthProbeDiagnostics(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	item := engine.Container{ID: "web1", Name: "web-1", State: "running", Health: "healthy"}
	inspect := &engine.ContainerInspection{
		ID:     item.ID,
		Name:   item.Name,
		State:  item.State,
		Health: item.Health,
		HealthCheck: &engine.HealthProbe{
			Status:        "healthy",
			FailingStreak: 0,
			Start:         now.Add(-15 * time.Second),
			End:           now.Add(-14 * time.Second),
			ExitCode:      0,
			Output:        "HTTP/1.1 200 OK\r\n{\"status\":\"pass\"}",
		},
	}

	text := Containers(plainSheet(), []engine.Container{item}, 0, nil, "", inspect, true, false, nil, nil, nil, 120, 30, now)
	for _, want := range []string{
		"HEALTHCHECK PROBE",
		"Status: healthy",
		"Exit code: 0",
		"Failing streak: 0",
		"Last checked: 14s ago",
		"HTTP/1.1 200 OK",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("container health probe view missing %q:\n%s", want, text)
		}
	}
}

func TestDashboardReflectsDaemonInfoAndStore(t *testing.T) {
	st := store.Store{
		Containers: []engine.Container{
			{ID: "c1", Name: "c-1", State: "running"},
		},
		Daemon: engine.DaemonInfo{
			NCPU:          16,
			ServerVersion: "27.3.1",
			MemTotal:      34_359_738_368,
			CgroupVersion: "2",
		},
	}
	text := Dashboard(plainSheet(), st, nil, 120, 30, false)
	for _, want := range []string{
		"16 host CPUs",
		"Engine 27.3.1",
		"34.4 GB total RAM",
		"Cgroup v2",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("dashboard missing daemon info telemetry %q:\n%s", want, text)
		}
	}
}
