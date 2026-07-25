package main

import (
	"errors"
	"fmt"

	"github.com/MichaelHolley/gistsync/internal/config"
	"github.com/MichaelHolley/gistsync/internal/gh"
	"github.com/MichaelHolley/gistsync/internal/state"
)

func runLink(args []string) error {
	fs := newFlagSet("link")
	if err := parseFlags(fs, args, "Usage: gistsync link <name>"); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("link: expected exactly one name")
	}
	name := fs.Arg(0)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("link: %w", err)
	}
	if _, ok := cfg.Find(name); !ok {
		return fmt.Errorf("link: %q is not in this device's config.toml — config is authored per device, so add a [[file]] entry with this machine's path first, then run 'gistsync link %s' again", name, name)
	}

	if err := gh.Preflight(); err != nil {
		return fmt.Errorf("link: %w", err)
	}

	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("link: %w", err)
	}
	if record, ok := st.Find(name); ok && record.GistID != "" {
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

	// Only gist_id is recorded: inventing sync markers here would make the next
	// pull believe the local file already matches the gist.
	st.Put(state.Record{Name: name, GistID: id})
	if err := st.Save(); err != nil {
		return fmt.Errorf("link: %w", err)
	}

	fmt.Printf("Linked %s → %s\nRun 'gistsync pull %s' to fetch it.\n", name, gistURL(id), name)
	return nil
}
