package main

import "testing"

func TestValidateEngineIgnoresDockerContextInFakeMode(t *testing.T) {
	t.Setenv("DOCKER_CONTEXT", "production")
	if err := validateEngine("fake", "configured-context"); err != nil {
		t.Fatalf("offline fake mode depends on Docker configuration: %v", err)
	}
	if err := validateEngine("docker", ""); err == nil {
		t.Fatal("Docker context was silently ignored")
	}
}
