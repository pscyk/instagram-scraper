// instagram-scraper browses saved raw profile snapshots without making network requests.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"

	"github.com/pscyk/instagram-scraper/internal/snapshot"
	"github.com/pscyk/instagram-scraper/internal/tui"
)

type options struct {
	demo, json bool
	path       string
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, output, diagnostic io.Writer) int {
	settings, err := parseOptions(args, output)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err == nil {
		err = execute(ctx, settings, output)
	}
	if err != nil {
		_, _ = fmt.Fprintln(diagnostic, tui.SafeText(err.Error(), 500))
		return 1
	}
	return 0
}

func parseOptions(args []string, output io.Writer) (options, error) {
	settings := options{}
	flags := flag.NewFlagSet("instagram-scraper", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.BoolVar(&settings.demo, "demo", false, "open the example profile")
	flags.BoolVar(&settings.json, "json", false, "print loaded snapshots as JSON")
	flags.StringVar(&settings.path, "input", "", "snapshot file or directory")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprint(output, help)
			return settings, flag.ErrHelp
		}
		return settings, errors.New("invalid options; use --help")
	}
	if flags.NArg() > 1 || (settings.path != "" && flags.NArg() > 0) ||
		(settings.demo && (flags.NArg() > 0 || settings.path != "")) {
		return settings, errors.New("choose --demo or one snapshot file/directory")
	}
	if flags.NArg() == 1 {
		settings.path = flags.Arg(0)
	}
	if settings.path == "" && !settings.demo {
		return settings, errors.New("choose a snapshot file/directory, or --demo; use --help")
	}
	return settings, nil
}

func execute(ctx context.Context, settings options, output io.Writer) error {
	var items []snapshot.Snapshot
	var err error
	if settings.demo {
		items, err = snapshot.Demo()
	} else {
		items, err = snapshot.Load(settings.path)
	}
	if err != nil {
		return err
	}
	if settings.json {
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		if encoder.Encode(items) != nil {
			return errors.New("cannot write snapshot JSON")
		}
		return nil
	}
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return errors.New("interactive mode requires a terminal; use --json for scripts")
	}
	_, err = tea.NewProgram(tui.New(items).WithContext(ctx), tea.WithContext(ctx), tea.WithOutput(output)).Run()
	if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
		return nil
	}
	return err
}

const help = `Instagram Scraper

Usage:
  instagram-scraper --demo
  instagram-scraper results/
  instagram-scraper --input results/
  instagram-scraper --json results/

Options:
  --demo        Open the example profile
  --input PATH  JSON file or directory from lookup.py
  --json        Print JSON
  --help        Show help

Keys: ↑↓/jk select, enter profile, / search, f filter, s sort, c clear,
[ ] adjacent profile, esc back, ? help, q quit.
Images: i picture, n/p reel, t thumbnail. Requires Kitty and kitten.
Enter / Esc to return from an image.
`
