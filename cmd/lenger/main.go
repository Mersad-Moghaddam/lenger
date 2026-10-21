// Command lenger runs the Docker image manager or its offline component gallery.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/Mersad-Moghaddam/lenger/internal/app"
	"github.com/Mersad-Moghaddam/lenger/internal/config"
	"github.com/Mersad-Moghaddam/lenger/internal/core/services"
	"github.com/Mersad-Moghaddam/lenger/internal/engine"
	"github.com/Mersad-Moghaddam/lenger/internal/engine/docker"
	"github.com/Mersad-Moghaddam/lenger/internal/engine/fake"
	"github.com/Mersad-Moghaddam/lenger/internal/gallery"
	"github.com/Mersad-Moghaddam/lenger/internal/platform/omarchy"
	"github.com/Mersad-Moghaddam/lenger/internal/ui/theme"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "lenger:", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) > 1 && os.Args[1] == "setup-omarchy" {
		apply := len(os.Args) > 2 && (os.Args[2] == "--apply" || os.Args[2] == "-y")
		fmt.Print(omarchy.Setup(apply))
		return nil
	}
	flags := flag.NewFlagSet("lenger", flag.ContinueOnError)
	galleryFlag := flags.Bool("gallery", false, "render offline gallery without Docker")
	themeFlag := flags.String("theme", "", "bundled theme name or omarchy")
	sizeFlag := flags.String("size", "", "terminal viewport, e.g. 120x40")
	debugFlag := flags.Bool("debug", false, "write safe operational diagnostics to the state directory")
	engineDefault := os.Getenv("LENGER_ENGINE")
	if engineDefault == "" {
		engineDefault = "docker"
	}
	engineFlag := flags.String("engine", engineDefault, "docker or fake (fake never contacts Docker)")
	if err := flags.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if *galleryFlag {
		return gallery.Render(ctx, os.Stdout, *themeFlag, *sizeFlag)
	}
	c, err := config.Load()
	if err != nil {
		return err
	}
	if *themeFlag != "" {
		c.UI.Theme = *themeFlag
	}
	if err := validateEngine(*engineFlag, c.Docker.Context); err != nil {
		return err
	}
	p, err := theme.Bundled("midnight")
	if err != nil {
		return err
	}
	if c.UI.Theme == "omarchy" {
		p, err = omarchy.DefaultSource().Load(ctx)
	} else {
		p, err = theme.Bundled(c.UI.Theme)
	}
	if err != nil {
		return err
	}
	var e engine.Engine
	switch *engineFlag {
	case "fake":
		e = fake.New()
	case "docker":
		e, err = docker.New()
	}
	if err != nil {
		return err
	}
	defer func() { _ = e.Close() }()
	if *debugFlag {
		file, err := debugFile()
		if err != nil {
			return err
		}
		defer file.Close() //nolint:errcheck // Closing a diagnostic log must not hide terminal errors.
		logger := log.New(file)
		logger.Info("session started", "engine", *engineFlag, "theme", p.Name)
	}
	m := app.New(ctx, services.New(e), c, p)
	m.SetFixture(*engineFlag == "fake")
	opts := []tea.ProgramOption{tea.WithContext(ctx), tea.WithFPS(30)}
	if *sizeFlag != "" {
		size, err := gallery.ParseSize(*sizeFlag)
		if err != nil {
			return err
		}
		opts = append(opts, tea.WithWindowSize(size.Width, size.Height))
	}
	if _, present := os.LookupEnv("NO_COLOR"); present {
		opts = append(opts, tea.WithColorProfile(colorprofile.Ascii))
	}
	_, err = tea.NewProgram(m, opts...).Run()
	if err != nil {
		return fmt.Errorf("run terminal: %w", err)
	}
	return nil
}

// validateEngine keeps real-daemon restrictions out of offline fixture mode.
func validateEngine(name, configuredContext string) error {
	switch name {
	case "fake":
		return nil
	case "docker":
		if configuredContext != "" || os.Getenv("DOCKER_CONTEXT") != "" {
			return engine.ErrNotImplemented{Requirement: "PLT-04 Docker contexts; use DOCKER_HOST for unix/tcp endpoints"}
		}
		return nil
	default:
		return fmt.Errorf("unknown engine %q: use docker or fake", name)
	}
}

func debugFile() (*os.File, error) {
	dir := os.Getenv("XDG_STATE_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		dir = filepath.Join(home, ".local", "state")
	}
	dir = filepath.Join(dir, "lenger")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(dir, "debug.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("open debug log: %w", err)
	}
	return file, nil
}
