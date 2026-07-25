package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

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

var plannedCommands = []string{"link", "pull"}

func main() {
	err := run(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gistsync:", err)
		os.Exit(1)
	}
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

// parseFlags returns flag.ErrHelp after printing usageLine when help was
// requested; main treats that as a successful exit.
func parseFlags(fs *flag.FlagSet, args []string, usageLine string) error {
	err := fs.Parse(flagsFirst(fs, args))
	if errors.Is(err, flag.ErrHelp) {
		fmt.Println(usageLine)
		return err
	}
	if err != nil {
		return fmt.Errorf("%s: %w", fs.Name(), err)
	}
	return nil
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
	case "add":
		return runAdd(args[1:])
	case "list":
		return runList(args[1:])
	case "rm":
		return runRm(args[1:])
	case "push":
		return runPush(args[1:])
	case "status":
		return runStatus(args[1:])
	default:
		for _, planned := range plannedCommands {
			if cmd == planned {
				return fmt.Errorf("%s: not implemented yet", cmd)
			}
		}
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
}

// flagsFirst reorders args so flags precede positional arguments, letting
// 'gistsync add <path> --name x' work as documented — the flag package
// otherwise stops parsing at the first positional argument.
func flagsFirst(fs *flag.FlagSet, args []string) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			positional = append(positional, args[i+1:]...)
			return append(flags, positional...)
		case strings.HasPrefix(arg, "-") && arg != "-":
			flags = append(flags, arg)
			if !strings.Contains(arg, "=") && takesValue(fs, arg) && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		default:
			positional = append(positional, arg)
		}
	}
	return append(flags, positional...)
}

func takesValue(fs *flag.FlagSet, arg string) bool {
	f := fs.Lookup(strings.TrimLeft(arg, "-"))
	if f == nil {
		return false
	}
	boolFlag, ok := f.Value.(interface{ IsBoolFlag() bool })
	return !ok || !boolFlag.IsBoolFlag()
}

func runInit(args []string) error {
	fs := newFlagSet("init")
	if err := parseFlags(fs, args, "Usage: gistsync init"); err != nil {
		return err
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
