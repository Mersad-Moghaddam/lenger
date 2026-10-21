package containers

import (
	"reflect"
	"testing"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// Catches sorting, grouping by service/name, and conflating standalone with a real project.
func TestGroupByComposeStableIdentity(t *testing.T) {
	in := []engine.Container{
		{ID: "b1", Project: "stale", Labels: map[string]string{"com.docker.compose.project": "beta"}},
		{ID: "solo"},
		{ID: "a1", Project: "alpha"},
		{ID: "b2", Labels: map[string]string{"com.docker.compose.project": "beta"}},
		{ID: "named", Project: "standalone"},
		{ID: "solo2", Labels: map[string]string{}},
	}
	wantProjects := []string{"beta", "", "alpha", "standalone"}
	wantIDs := [][]string{{"b1", "b2"}, {"solo", "solo2"}, {"a1"}, {"named"}}
	for range 20 {
		groups := GroupByCompose(in)
		if len(groups) != len(wantProjects) {
			t.Fatalf("groups = %#v", groups)
		}
		for i, g := range groups {
			var ids []string
			for _, c := range g.Services {
				ids = append(ids, c.ID)
			}
			if g.Project != wantProjects[i] || !reflect.DeepEqual(ids, wantIDs[i]) || g.Collapsed {
				t.Fatalf("group %d = %#v, IDs %v", i, g, ids)
			}
		}
		groups[0].Services[0].ID = "changed"
	}
	if in[0].ID != "b1" || in[0].Project != "stale" || in[0].Labels["com.docker.compose.project"] != "beta" || in[1].ID != "solo" {
		t.Fatalf("input mutated: %#v", in)
	}
}

func TestGroupByComposeEmpty(t *testing.T) {
	if got := GroupByCompose(nil); len(got) != 0 {
		t.Fatalf("empty input = %#v", got)
	}
}
