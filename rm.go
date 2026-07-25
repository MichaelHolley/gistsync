package main

import (
	"errors"
	"fmt"

	"github.com/MichaelHolley/gistsync/internal/config"
	"github.com/MichaelHolley/gistsync/internal/state"
)

func runRm(args []string) error {
	fs := newFlagSet("rm")
	if err := parseFlags(fs, args, "Usage: gistsync rm <name>"); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("rm: expected exactly one name")
	}
	name := fs.Arg(0)

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("rm: %w", err)
	}
	entry, ok := cfg.Find(name)
	if !ok {
		return fmt.Errorf("rm: %q is not tracked — see 'gistsync list'", name)
	}
	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("rm: %w", err)
	}
	record, hadRecord := st.Find(name)

	cfg.Remove(name)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("rm: %w", err)
	}
	st.Remove(name)
	if err := st.Save(); err != nil {
		return fmt.Errorf("rm: %w", err)
	}

	fmt.Printf("Untracked %q. %s is untouched on disk.\n", name, entry.Path)
	if hadRecord && record.GistID != "" {
		fmt.Printf("The gist was left in place: %s\n", gistURL(record.GistID))
		fmt.Println("Delete it yourself on github.com if you no longer want it.")
	} else {
		fmt.Println("No gist existed for this name, so nothing remote was affected.")
	}
	return nil
}
