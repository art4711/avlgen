package main

import (
	"avlgen/avlgen"
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintf(os.Stderr, "Usage: avlgen <node type name> <outfile>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	nodeT := flag.Arg(0)
	outF := flag.Arg(1)
	c := avlgen.New(nodeT)
	out, err := os.OpenFile(outF, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()
	err = c.Gen(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		os.Exit(1)
	}
}
