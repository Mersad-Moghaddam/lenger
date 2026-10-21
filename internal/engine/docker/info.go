package docker

import (
	"context"
	"fmt"
	"slices"

	"github.com/moby/moby/client"

	"github.com/Mersad-Moghaddam/lenger/internal/engine"
)

// DaemonInfo returns host and Docker engine metrics and capabilities (SYS-03).
func (e *Engine) DaemonInfo(ctx context.Context) (engine.DaemonInfo, error) {
	infoRes, err := e.client.Info(ctx, client.InfoOptions{})
	if err != nil {
		return engine.DaemonInfo{}, fmt.Errorf("read daemon info: %w", err)
	}
	info := infoRes.Info
	out := engine.DaemonInfo{
		ID:              info.ID,
		Name:            info.Name,
		ServerVersion:   info.ServerVersion,
		OS:              info.OperatingSystem,
		OSType:          info.OSType,
		Arch:            info.Architecture,
		KernelVersion:   info.KernelVersion,
		Driver:          info.Driver,
		LoggingDriver:   info.LoggingDriver,
		CgroupDriver:    info.CgroupDriver,
		CgroupVersion:   info.CgroupVersion,
		RootDir:         info.DockerRootDir,
		NCPU:            info.NCPU,
		MemTotal:        info.MemTotal,
		SecurityOptions: slices.Clone(info.SecurityOptions),
		Warnings:        slices.Clone(info.Warnings),
	}

	verRes, err := e.client.ServerVersion(ctx, client.ServerVersionOptions{})
	if err == nil {
		out.APIVersion = verRes.APIVersion
		out.MinAPIVersion = verRes.MinAPIVersion
	}
	return out, nil
}
