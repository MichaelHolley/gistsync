package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"

	"github.com/MichaelHolley/gistsync/internal/config"
	"github.com/MichaelHolley/gistsync/internal/gh"
	"github.com/MichaelHolley/gistsync/internal/state"
)

func runPush(args []string) error {
	fs := newFlagSet("push")
	force := fs.Bool("force", false, "overwrite the gist with local content")
	if err := parseFlags(fs, args, "Usage: gistsync push <name> [--force]"); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("push: expected exactly one name")
	}
	name := fs.Arg(0)

	entry, err := trackedFile("push", name)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(entry.Path)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	if err := checkContent(entry.Path, content); err != nil {
		return fmt.Errorf("push: %w", err)
	}

	if err := gh.Preflight(); err != nil {
		return fmt.Errorf("push: %w", err)
	}

	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	record, _ := st.Find(name)

	gist, err := pushContent(entry, record, content, *force)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	if gist.ID == "" {
		fmt.Printf("%s is already up to date with %s\n", entry.Path, gistURL(record.GistID))
		return nil
	}

	st.Put(state.Record{
		Name:              name,
		GistID:            gist.ID,
		LastSyncedHash:    hashContent(content),
		LastSyncedGistSHA: gist.Version(),
	})
	if err := st.Save(); err != nil {
		return fmt.Errorf("push: %w", err)
	}

	fmt.Printf("Pushed %s → %s\n", entry.Path, gistURL(gist.ID))
	return nil
}

// pushContent returns a zero Gist when the gist already matches local content
// and there is nothing to upload.
func pushContent(entry config.File, record state.Record, content []byte, force bool) (gh.Gist, error) {
	if record.GistID == "" {
		existing, err := gh.FindGistID(entry.Name)
		if err != nil {
			return gh.Gist{}, err
		}
		if existing != "" {
			return gh.Gist{}, fmt.Errorf("a gist for %q already exists (%s) — run 'gistsync link %s' to adopt it", entry.Name, gistURL(existing), entry.Name)
		}
		return gh.CreateGist(entry.Name, content)
	}

	gist, err := gh.GetGist(record.GistID)
	if err != nil {
		return gh.Gist{}, err
	}

	if !force {
		switch s := classify(record, hashContent(content), gist.Version()); s {
		case stateClean:
			return gh.Gist{}, nil
		case stateBehind, stateConflict:
			return gh.Gist{}, refuse("push", entry, record.GistID, s)
		}
	}
	return gh.UpdateGist(record.GistID, entry.Name, content)
}

func hashContent(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
