package main

import (
	"avlgen/avlgen"
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Parse()
	if flag.NArg() != 6 {
		fmt.Fprintf(os.Stderr, "Usage: avlgen <node type name> <link type name> <link member name> <tree type name> <package name> <outfile>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	c := avlgen.New(flag.Arg(0), flag.Arg(1), flag.Arg(2), flag.Arg(3), flag.Arg(4))
	out, err := os.OpenFile(flag.Arg(5), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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
