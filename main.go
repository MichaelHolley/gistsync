package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/MichaelHolley/gistsync/internal/store"
)

const usage = `gistsync — sync individual files across devices via GitHub Gists

Usage:
  gistsync <command> [arguments]

Commands:
  init                          create ~/.gistsync/ with config.toml and state.json
  add <path> [--name <name>]    track a file locally
  link <name>                   adopt an existing gist for a name in config.toml
  rm <name>                     stop tracking a file on this device
  status                        show clean / ahead / behind / conflict per file
  list                          show tracked names, local paths, and gist URLs
  push <name> [--force]         upload local file to its gist
  pull <name> [--force]         write gist content to the local file

Run 'gistsync <command> --help' for command-specific flags.
`

var plannedCommands = []string{"add", "link", "rm", "status", "list", "push", "pull"}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "gistsync:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}

	switch cmd := args[0]; cmd {
	case "help", "--help", "-h":
		fmt.Print(usage)
		return nil
	case "init":
		return runInit(args[1:])
	default:
		for _, planned := range plannedCommands {
			if cmd == planned {
				return fmt.Errorf("%s: not implemented yet", cmd)
			}
		}
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Println("Usage: gistsync init")
			return nil
		}
		return fmt.Errorf("init: %w", err)
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("init: unexpected argument %q", fs.Arg(0))
	}

	dir, created, err := store.Init()
	if err != nil {
		return fmt.Errorf("init: %w", err)
	}
	if created {
		fmt.Printf("Initialised %s\n", dir)
	} else {
		fmt.Printf("%s already exists — nothing changed\n", dir)
	}
	return nil
}
