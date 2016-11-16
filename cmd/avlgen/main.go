package main

import (
	"avlgen/avlgen"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: avlgen <infile.go>\n")
		flag.PrintDefaults()
		os.Exit(1)

	}
	fname := flag.Arg(0)

	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, fname, nil, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ParseFile: %v\n", err)
		os.Exit(1)
	}
	packN := f.Name.Name
	confs := []*avlgen.Conf{}
	first := true
	ast.Inspect(f, func(n ast.Node) bool {

		typ, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := typ.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, f := range st.Fields.List {
			if f.Tag == nil {
				continue
			}
			ts, err := strconv.Unquote(f.Tag.Value)
			if err != nil {
				log.Fatal(err)
			}
			tag := reflect.StructTag(ts)
			tv, ok := tag.Lookup("avlgen")
			if !ok {
				continue
			}
			fType, ok := f.Type.(*ast.Ident)
			if !ok {
				panic("I understand nothing")
			}
			if len(f.Names) != 1 {
				panic("Make my life easier, give the struct field one name and one name only, please.")
			}
			c, err := avlgen.New(typ.Name.Name, fType.Name, f.Names[0].Name, "", packN, first, tv)
			if err != nil {
				log.Fatal(err)
			}
			confs = append(confs, c)
			first = false
		}
		return true
	})
	n := strings.TrimSuffix(fname, ".go") + "_trees.go"
	out, err := os.OpenFile(n, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open: %v\n", err)
		os.Exit(1)
	}
	defer out.Close()
	for _, c := range confs {
		err = c.Gen(out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "gen: %v\n", err)
			os.Exit(1)
		}
	}
}
