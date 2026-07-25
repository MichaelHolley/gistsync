package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/MichaelHolley/gistsync/internal/config"
	"github.com/MichaelHolley/gistsync/internal/gh"
)

func runAdd(args []string) error {
	fs := newFlagSet("add")
	name := fs.String("name", "", "logical name (defaults to the file's basename)")
	if err := parseFlags(fs, args, "Usage: gistsync add <path> [--name <name>]"); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("add: expected exactly one path")
	}

	path, err := filepath.Abs(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("add: %w", err)
	}
	if err := checkTrackable(path); err != nil {
		return fmt.Errorf("add: %w", err)
	}

	logicalName := *name
	derived := logicalName == ""
	if derived {
		logicalName = deriveName(path)
		if logicalName == "" {
			return fmt.Errorf("add: cannot derive a name from %s — pass --name", path)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("add: %w", err)
	}
	if existing, ok := cfg.Find(logicalName); ok {
		if derived {
			return fmt.Errorf("add: name %q is already tracked (%s) — pass --name to choose a different logical name", logicalName, existing.Path)
		}
		return fmt.Errorf("add: name %q is already tracked (%s)", logicalName, existing.Path)
	}

	if err := gh.Preflight(); err != nil {
		return fmt.Errorf("add: %w", err)
	}
	existing, err := gh.FindGistID(logicalName)
	if err != nil {
		return fmt.Errorf("add: %w", err)
	}
	if existing != "" {
		return fmt.Errorf("add: a gist for %q already exists (%s) — add the entry to config.toml and run 'gistsync link %s' instead", logicalName, gistURL(existing), logicalName)
	}

	cfg.Add(config.File{Name: logicalName, Path: path})
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("add: %w", err)
	}

	fmt.Printf("Tracking %s as %q\n", path, logicalName)
	fmt.Printf("Run 'gistsync push %s' to create its gist.\n", logicalName)
	return nil
}

// deriveName drops leading dots so ~/.vimrc becomes the logical name "vimrc".
func deriveName(path string) string {
	return strings.TrimLeft(filepath.Base(path), ".")
}

func checkTrackable(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s: no such file", path)
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s: is a directory — gistsync tracks individual files", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return checkContent(path, data)
}

// checkContent enforces the text-only guarantee. Invalid UTF-8 is rejected
// because the gist API is JSON, which cannot carry those bytes unaltered.
func checkContent(path string, data []byte) error {
	if bytes.IndexByte(data, 0) >= 0 {
		return fmt.Errorf("%s: looks binary (contains a NUL byte) — gistsync tracks text files only", path)
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("%s: not valid UTF-8 — gistsync cannot transfer it byte-exact", path)
	}
	return nil
}
