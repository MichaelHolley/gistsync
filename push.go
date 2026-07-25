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
	if err := parseFlags(fs, args, "Usage: gistsync push <name>"); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("push: expected exactly one name")
	}
	name := fs.Arg(0)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	entry, ok := cfg.Find(name)
	if !ok {
		return fmt.Errorf("push: %q is not tracked — see 'gistsync list'", name)
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

	gist, err := pushContent(name, record.GistID, content)
	if err != nil {
		return fmt.Errorf("push: %w", err)
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

func pushContent(name, gistID string, content []byte) (gh.Gist, error) {
	if gistID != "" {
		return gh.UpdateGist(gistID, name, content)
	}

	existing, err := gh.FindGistID(name)
	if err != nil {
		return gh.Gist{}, err
	}
	if existing != "" {
		return gh.Gist{}, fmt.Errorf("a gist for %q already exists (%s) — run 'gistsync link %s' to adopt it", name, gistURL(existing), name)
	}
	return gh.CreateGist(name, content)
}

func hashContent(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
