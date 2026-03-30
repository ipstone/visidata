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
	var limit int
	var table string
	flag.IntVar(&limit, "n", 0, "number of rows to preview instead of launching the interactive viewer")
	flag.StringVar(&table, "table", "", "table name to load from a SQLite database")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: vdgo [-n rows] <file|->")
		os.Exit(2)
	}

	sh, err := loader.LoadWithOptions(flag.Arg(0), loader.Options{Table: table})
	if err != nil {
		fmt.Fprintf(os.Stderr, "vdgo: %v\n", err)
		os.Exit(1)
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
