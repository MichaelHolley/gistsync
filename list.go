package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/MichaelHolley/gistsync/internal/config"
	"github.com/MichaelHolley/gistsync/internal/state"
)

func runList(args []string) error {
	fs := newFlagSet("list")
	if err := parseFlags(fs, args, "Usage: gistsync list"); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("list: unexpected argument %q", fs.Arg(0))
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	st, err := state.Load()
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}

	if len(cfg.Files) == 0 {
		fmt.Println("No files tracked — add one with 'gistsync add <path>'.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tPATH\tGIST")
	for _, f := range cfg.Files {
		fmt.Fprintf(w, "%s\t%s\t%s\n", f.Name, f.Path, gistColumn(st, f.Name))
	}
	return w.Flush()
}

func gistColumn(st *state.State, name string) string {
	if r, ok := st.Find(name); ok && r.GistID != "" {
		return gistURL(r.GistID)
	}
	return "not pushed"
}

func gistURL(id string) string {
	return "https://gist.github.com/" + id
}
