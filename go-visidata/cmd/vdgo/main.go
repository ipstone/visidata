package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ipstone/visidata/go-visidata/pkg/display"
	"github.com/ipstone/visidata/go-visidata/pkg/loader"
)

func main() {
	var limit int
	flag.IntVar(&limit, "n", 10, "number of rows to preview")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: vdgo [-n rows] <file|->")
		os.Exit(2)
	}

	sh, err := loader.Load(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "vdgo: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(display.RenderPreview(sh, limit))
}
