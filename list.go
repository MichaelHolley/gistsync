package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/MichaelHolley/gistsync/internal/config"
	"github.com/MichaelHolley/gistsync/internal/gh"
	"github.com/MichaelHolley/gistsync/internal/state"
)

func runList(args []string) error {
	fs := newFlagSet("list")
	remote := fs.Bool("remote", false, "list the gistsync gists on your GitHub account")
	if err := parseFlags(fs, args, "Usage: gistsync list [--remote]"); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("list: unexpected argument %q", fs.Arg(0))
	}
	if *remote {
		return listRemote()
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

func listRemote() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	if err := gh.Preflight(); err != nil {
		return fmt.Errorf("list: %w", err)
	}
	gists, err := gh.ListGists()
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	if len(gists) == 0 {
		fmt.Println("No gistsync gists on this account — push one from the device that has the file.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTRACKED HERE\tGIST")
	for _, g := range gists {
		tracked := "no"
		if f, ok := cfg.Find(g.Name); ok {
			tracked = f.Path
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", g.Name, tracked, gistURL(g.ID))
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
