package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ipstone/visidata/go-visidata/pkg/display"
	"github.com/ipstone/visidata/go-visidata/pkg/loader"
	"github.com/ipstone/visidata/go-visidata/pkg/tui"
)

func main() {
	var filetype string
	var header int
	var limit int
	var savePath string
	var table string
	flag.StringVar(&filetype, "filetype", "", "force a specific loader (for example: fixed, json, jsonl, delimited, sqlite)")
	flag.IntVar(&header, "header", 1, "number of header lines for fixed-width loading")
	flag.IntVar(&limit, "n", 0, "number of rows to preview instead of launching the interactive viewer")
	flag.StringVar(&savePath, "save", "", "export the loaded sheet to a file and exit")
	flag.StringVar(&table, "table", "", "table name to load from a SQLite database")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: vdgo [-n rows] [-save path] [-filetype fixed] [-header lines] [-table name] <file|->")
		os.Exit(2)
	}

	sh, err := loader.LoadWithOptions(flag.Arg(0), loader.Options{
		Format: filetype,
		Header: header,
		Table:  table,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "vdgo: %v\n", err)
		os.Exit(1)
	}

	if savePath != "" {
		if err := sh.Export(savePath); err != nil {
			fmt.Fprintf(os.Stderr, "vdgo: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "saved %s\n", savePath)
		return
	}

	if limit > 0 {
		fmt.Print(display.RenderPreview(sh, limit))
		return
	}

	if err := tui.Run(sh); err != nil {
		fmt.Fprintf(os.Stderr, "vdgo: %v\n", err)
		os.Exit(1)
	}
}
