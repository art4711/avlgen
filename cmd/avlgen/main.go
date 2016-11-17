package main

import (
	"avlgen/avlgen"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func parseFile(fs *token.FileSet, fname string, trees *avlgen.Trees) {
	f, err := parser.ParseFile(fs, fname, nil, 0)
	if err != nil {
		log.Fatalf("parser.ParseFile(%s): %v", fname, err)
	}
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
				log.Fatalf("unquote: %v", err)
			}
			tag := reflect.StructTag(ts)
			tv, ok := tag.Lookup("avlgen")
			if !ok {
				continue
			}
			fType, ok := f.Type.(*ast.Ident)
			if !ok {
				log.Fatalf("Apparently things aren't as simple as I understand them: %v", f.Type)
			}
			if len(f.Names) != 1 {
				log.Fatalf("%s embed field name problem: %v", typ.Name.Name, f.Names)
			}
			err = trees.AddTree(typ.Name.Name, fType.Name, f.Names[0].Name, "", tv)
			if err != nil {
				log.Fatal(err)
			}
		}
		return true
	})
}

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Usage: avlgen <package dir>\n")
		flag.PrintDefaults()
		os.Exit(1)

	}
	dname := flag.Arg(0)

	fs := token.NewFileSet()
	pkg, err := build.Default.ImportDir(dname, 0)
	if err != nil {
		log.Fatalf("cannot process directory '%s': %v", dname, err)
	}
	trees := avlgen.New(pkg.Name)
	for _, fname := range pkg.GoFiles {
		parseFile(fs, fname, trees)
	}
	for _, fname := range pkg.TestGoFiles {
		parseFile(fs, fname, trees)
	}
	n := strings.TrimSuffix(pkg.Name, ".go") + "_trees.go"
	out, err := os.OpenFile(n, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("open(%s): %v", n, err)
	}
	defer out.Close()
	err = trees.Gen(out)
	if err != nil {
		os.Remove(n)
		log.Fatalf("gen: %v\n", err)
	}
}
