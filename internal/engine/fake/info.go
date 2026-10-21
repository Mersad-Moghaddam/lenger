package fake

import (
	"context"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// DaemonInfo returns deterministic host and engine telemetry (SYS-03).
func (e *Engine) DaemonInfo(ctx context.Context) (engine.DaemonInfo, error) {
	if err := ctx.Err(); err != nil {
		return engine.DaemonInfo{}, err
	}
	return engine.DaemonInfo{
		ID:              "fake-daemon-01",
		Name:            "harbor-station-01",
		ServerVersion:   "27.3.1",
		APIVersion:      "1.47",
		MinAPIVersion:   "1.24",
		OS:              "Ubuntu 24.04.1 LTS",
		OSType:          "linux",
		Arch:            "x86_64",
		KernelVersion:   "6.11.0-17-generic",
		Driver:          "overlay2",
		LoggingDriver:   "json-file",
		CgroupDriver:    "systemd",
		CgroupVersion:   "2",
		RootDir:         "/var/lib/docker",
		NCPU:            16,
		MemTotal:        34_359_738_368, // 32 GiB
		SecurityOptions: []string{"name=seccomp,profile=builtin", "name=cgroupns"},
		Warnings:        nil,
	}, nil
}
