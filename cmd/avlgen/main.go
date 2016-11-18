// Avlgen is a tool to automatically generate code for embedded AVL
// trees in structs. The purpose is to be able to store structs in an
// ordered tree without any additional memory allocation. All the
// data needed to store a struct in the tree is part of the struct.
//
// Avlgen is invoked on a package directory, examines the package and
// finds structs with fields that contain a special tag and generates
// the necessary code for those structs in the same package.
//
// For example:
//
//	package foo
//
//	//go:generate avlgen .
//	type str struct {
//		key string
//		tl  tlink `avlgen:"strTree"`
//	}
//
//	func (a *str)cmp(b *str) (bool, bool) {
//		return a.key == b.key, a.key < b.key
//	}
//
// This will generate a file called "foo_trees.go" with the type
// "strTree", and some access methods. You can now manage str structs
// like this:
//
//	st := strTree{}
//	st.insert(&str{ key: "foobar" })
//	s := st.lookup(&str{ key: "foobar" })
//	st.delete(s)
//
// The "tlink" part is a type name of the type that avlgen will
// generate and must be unique for each tree, but otherwise can be any
// string. The same goes for its name in the struct. This is the part
// that embeds the tree in the struct.
//
// This is the simplest possible tree and is already relatively useful
// especially since the struct "str" can contain any data. Of course, things
// can be improved with some configuration. Insert, lookup and delete won't
// perform any memory allocations.
//
// The default comparison function is a receiver of the type of the tree
// elements called "cmp". We can change its name by just adding a bit more
// information to the tag:
//
//	type str struct {
//		key string
//		tl tlink `avlgen:"strTree,cmp:cmpKeys"`
//	}
//
// Now we expect there to be a "cmpKeys" function instead of
// "cmp". The compare function returns two booleans: if the two values
// are equal or if the first value is "less" than the other
// value. You're free to define "less" in whatever way you wish as
// long as it is transitive (if a > b and b > c then a > c).
//
// The next useful feature is obvious from the above "lookup"
// example. It's quite wasteful to allocate a fake struct to perform
// lookups just because our compare function only understands how to
// compare two element structs.
//
//	type str struct {
//		key string
//		tl tlink `avlgen:"strTree,cmpval:cmpk(string)"`
//	}
//	func (a *str)cmpk(b string) (bool, bool) {
//		return a.key == b, a.key < b
//	}
//
// This allows us to generate special functions lookupVal and
// deleteVal:
//
//	s := lookupVal("foobar")
//	tr.deleteVal("foobar")
//
// The type (string in this case) can be anything, of course. It's
// specified in the tag and the code is generated correctly for any
// key types (it shouldn't be too hard to add multiple arguments to
// the cmpk/lookupVal functions in case of more complex keys, but this
// isn't implemented yet).
//
// There is obviously no "insertVal" function since it is expected
// that structs are much more complex than this example.
//
// When `cmpval` is specified we also implement two more functions:
// searchValLEQ and searchValGEQ. They behave like lookup, but in case
// there's no equal element, they return the nearest less than (LEQ)
// or greater than (GEQ) node.
//
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

var outFname = flag.String("o", "", "output file name")

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

	outName := strings.TrimSuffix(pkg.Name, ".go") + "_trees.go"
	if *outFname != "" {
		outName = *outFname
	}

	trees := avlgen.New(pkg.Name)
	for _, fname := range pkg.GoFiles {
		if fname == outName {
			continue // Skip the generated file (mostly for my testing)
		}
		parseFile(fs, fname, trees)
	}
	for _, fname := range pkg.TestGoFiles {
		parseFile(fs, fname, trees)
	}
	out, err := os.OpenFile(outName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("open(%s): %v", outName, err)
	}
	defer out.Close()
	err = trees.Gen(out)
	if err != nil {
		os.Remove(outName)
		log.Fatalf("gen: %v\n", err)
	}
}
