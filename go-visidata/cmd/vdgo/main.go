package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ipstone/visidata/go-visidata/pkg/config"
	"github.com/ipstone/visidata/go-visidata/pkg/display"
	"github.com/ipstone/visidata/go-visidata/pkg/loader"
	"github.com/ipstone/visidata/go-visidata/pkg/tui"
)

func main() {
	opts, inputPath, err := parseCLI()
	if err != nil {
		fmt.Fprintf(os.Stderr, "vdgo: %v\n", err)
		os.Exit(2)
	}

	sh, err := loader.LoadWithOptions(inputPath, loader.Options{
		Format:     opts.filetype,
		Header:     opts.header,
		Table:      opts.table,
		ShowHidden: opts.showHidden,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "vdgo: %v\n", err)
		os.Exit(1)
	}

	if opts.savePath != "" {
		if err := sh.Export(opts.savePath); err != nil {
			fmt.Fprintf(os.Stderr, "vdgo: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "saved %s\n", opts.savePath)
		return
	}

	if opts.limit > 0 {
		fmt.Print(display.RenderPreview(sh, opts.limit))
		return
	}

	if err := tui.RunWithOptions(sh, tui.Options{Theme: opts.theme}); err != nil {
		fmt.Fprintf(os.Stderr, "vdgo: %v\n", err)
		os.Exit(1)
	}
}

type cliOptions struct {
	configPath string
	filetype   string
	header     int
	limit      int
	savePath   string
	showHidden bool
	table      string
	theme      string
}

func parseCLI() (cliOptions, string, error) {
	var filetype string
	var header int
	var limit int
	var savePath string
	var showHidden bool
	var table string
	var theme string
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to a TOML config file (defaults to .vdgorc discovery)")
	flag.StringVar(&filetype, "filetype", "", "force a specific loader (for example: fixed, json, jsonl, delimited, sqlite)")
	flag.IntVar(&header, "header", 1, "number of header lines for fixed-width loading")
	flag.IntVar(&limit, "n", 0, "number of rows to preview instead of launching the interactive viewer")
	flag.StringVar(&savePath, "save", "", "export the loaded sheet to a file and exit")
	flag.BoolVar(&showHidden, "show-hidden", false, "include hidden directory entries when loading directories")
	flag.StringVar(&table, "table", "", "table name to load from a SQLite database or sheet name for XLSX")
	flag.StringVar(&theme, "theme", "", "built-in TUI theme name (default, amber, ocean)")
	flag.Parse()

	if flag.NArg() != 1 {
		return cliOptions{}, "", fmt.Errorf("usage: vdgo [-config path] [-n rows] [-save path] [-filetype fixed] [-header lines] [-table name] <file|->")
	}

	opts := cliOptions{
		configPath: configPath,
		filetype:   filetype,
		header:     header,
		limit:      limit,
		savePath:   savePath,
		showHidden: showHidden,
		table:      table,
		theme:      theme,
	}
	explicit := visitedFlags()
	cfgPath, err := config.Discover(configPath)
	if err != nil {
		return cliOptions{}, "", err
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return cliOptions{}, "", err
	}
	applyConfig(&opts, cfg, explicit)
	return opts, flag.Arg(0), nil
}

func visitedFlags() map[string]bool {
	visited := make(map[string]bool)
	flag.CommandLine.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}

func applyConfig(opts *cliOptions, cfg config.Config, explicit map[string]bool) {
	if !explicit["filetype"] && cfg.Filetype != nil {
		opts.filetype = *cfg.Filetype
	}
	if !explicit["header"] && cfg.Header != nil {
		opts.header = *cfg.Header
	}
	if !explicit["n"] && cfg.PreviewRows != nil {
		opts.limit = *cfg.PreviewRows
	}
	if !explicit["table"] && cfg.Table != nil {
		opts.table = *cfg.Table
	}
	if !explicit["theme"] && cfg.Theme != nil {
		opts.theme = *cfg.Theme
	}
	if !explicit["show-hidden"] && cfg.ShowHidden != nil {
		opts.showHidden = *cfg.ShowHidden
	}
}
