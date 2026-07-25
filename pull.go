package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MichaelHolley/gistsync/internal/gh"
	"github.com/MichaelHolley/gistsync/internal/state"
	"github.com/MichaelHolley/gistsync/internal/store"
)

func runPull(args []string) error {
	fs := newFlagSet("pull")
	force := fs.Bool("force", false, "overwrite the local file with gist content")
	if err := parseFlags(fs, args, "Usage: gistsync pull <name> [--force]"); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("pull: expected exactly one name")
	}
	name := fs.Arg(0)

	entry, err := trackedFile("pull", name)
	if err != nil {
		return err
	}

	if err := gh.Preflight(); err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	record, _ := st.Find(name)
	if record.GistID == "" {
		return fmt.Errorf("pull: %q has no gist yet — run 'gistsync link %s' to adopt an existing one, or 'gistsync push %s' to create it", name, name, name)
	}

	gist, err := gh.GetGist(record.GistID)
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	content, err := gist.Content(name)
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	if err := checkContent(gistURL(gist.ID), content); err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	local, err := os.ReadFile(entry.Path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		// A missing local file has nothing to lose, so it is always written.
	case err != nil:
		return fmt.Errorf("pull: %w", err)
	case !*force:
		switch s := classify(record, hashContent(local), gist.Version()); s {
		case stateClean:
			fmt.Printf("%s is already up to date with %s\n", entry.Path, gistURL(gist.ID))
			return nil
		case stateAhead, stateConflict:
			return fmt.Errorf("pull: %w", refuse("pull", entry, gist.ID, s))
		}
	}

	if err := writeLocal(entry.Path, content); err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	st.Put(state.Record{
		Name:              name,
		GistID:            gist.ID,
		LastSyncedHash:    hashContent(content),
		LastSyncedGistSHA: gist.Version(),
	})
	if err := st.Save(); err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	fmt.Printf("Pulled %s → %s\n", gistURL(gist.ID), entry.Path)
	return nil
}

func writeLocal(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	perm := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}
	return store.WriteFileAtomic(path, content, perm)
}
