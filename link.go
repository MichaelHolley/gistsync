package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/MichaelHolley/gistsync/internal/config"
	"github.com/MichaelHolley/gistsync/internal/gh"
	"github.com/MichaelHolley/gistsync/internal/state"
)

func runLink(args []string) error {
	fs := newFlagSet("link")
	if err := parseFlags(fs, args, "Usage: gistsync link <name> [path]"); err != nil {
		return err
	}
	if fs.NArg() < 1 || fs.NArg() > 2 {
		return errors.New("link: expected a name and an optional path")
	}
	name := fs.Arg(0)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("link: %w", err)
	}
	entry, tracked := cfg.Find(name)

	path := entry.Path
	if fs.NArg() == 2 {
		path, err = resolvePath(fs.Arg(1))
		if err != nil {
			return fmt.Errorf("link: %w", err)
		}
		if tracked && entry.Path != path {
			return fmt.Errorf("link: %q is already tracked at %s — run 'gistsync rm %s' first if you want it somewhere else", name, entry.Path, name)
		}
		if err := checkLinkable(path); err != nil {
			return fmt.Errorf("link: %w", err)
		}
	} else if !tracked {
		return fmt.Errorf("link: %q is not tracked on this device — pass the local path: 'gistsync link %s <path>'", name, name)
	}

	if err := gh.Preflight(); err != nil {
		return fmt.Errorf("link: %w", err)
	}

	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("link: %w", err)
	}
	if record, ok := st.Find(name); ok && record.GistID != "" {
		if err := trackLocally(cfg, tracked, name, path); err != nil {
			return fmt.Errorf("link: %w", err)
		}
		fmt.Printf("%s is already linked to %s — nothing changed\n", name, gistURL(record.GistID))
		return nil
	}

	id, err := gh.FindGistID(name)
	if err != nil {
		return fmt.Errorf("link: %w", err)
	}
	if id == "" {
		return fmt.Errorf("link: no gist described %q — %s was never pushed, so run 'gistsync push %s' on the device that has the file", gh.Description(name), name, name)
	}
	gist, err := gh.GetGist(id)
	if err != nil {
		return fmt.Errorf("link: %w", err)
	}
	content, err := gist.Content(name)
	if err != nil {
		return fmt.Errorf("link: %w", err)
	}
	if err := checkContent(gistURL(gist.ID), content); err != nil {
		return fmt.Errorf("link: %w", err)
	}

	local, readErr := os.ReadFile(path)
	missing := errors.Is(readErr, os.ErrNotExist)
	if readErr != nil && !missing {
		return fmt.Errorf("link: %w", readErr)
	}
	matches := readErr == nil && bytes.Equal(local, content)

	if err := trackLocally(cfg, tracked, name, path); err != nil {
		return fmt.Errorf("link: %w", err)
	}

	// Sync markers are recorded only when both sides already hold the same
	// bytes; inventing them otherwise would make the next pull believe the
	// local file already matches the gist.
	record := state.Record{Name: name, GistID: gist.ID}
	if matches {
		record.LastSyncedHash = hashContent(content)
		record.LastSyncedGistSHA = gist.Version()
	}
	st.Put(record)
	if err := st.Save(); err != nil {
		return fmt.Errorf("link: %w", err)
	}

	fmt.Printf("Linked %s → %s\n", name, gistURL(gist.ID))
	switch {
	case missing:
		fmt.Printf("%s does not exist yet.\nRun 'gistsync pull %s' to fetch it.\n", path, name)
	case matches:
		fmt.Printf("%s already matches the gist — nothing to transfer.\n", path)
	default:
		fmt.Printf(`%s differs from the gist — this device has never synced it.
  local: %s
  gist:  %s
Compare them, then choose a winner:
  gistsync push --force %s   local wins, overwrites the gist
  gistsync pull --force %s   remote wins, overwrites the local file
`, path, path, gistURL(gist.ID), name, name)
	}
	return nil
}

func trackLocally(cfg *config.Config, tracked bool, name, path string) error {
	if tracked {
		return nil
	}
	cfg.Add(config.File{Name: name, Path: path})
	return cfg.Save()
}

// checkLinkable applies the trackable rules only when the file is already
// here; on a second device the path usually points at nothing yet.
func checkLinkable(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return checkTrackable(path)
}
