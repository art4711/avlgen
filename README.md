# avl trees

Who needs templates when you have go generate.

## What?

This is an implementation of an inline (some people call it embedded)
AVL tree very heavily based on [my C implementation](https://github.com/art4711/stuff/tree/master/avl).

Except in Go.

Since we don't have templates and there's no pre-processor and we
can't commit pointer atrocities we `go generate` the necessary code
instead.

## How?

In short you need something like this in `anyfile.go`:

    //go:generate go run <figure out path later>/cmd/avlgen/main.go -- anyfile.go
    type anyStruct struct {
        anyFieldName anyTypeName `avlgen:"nameOfTreeType"`
    }

    func (a *anyStruct)cmp(b *anyStruct) (bool, bool) {
        return a.Equals(b), a.Less(b)
    }

Running `go generate` will create `anyfile_trees.go` that contains the
implementation of the type `nameOfTreeType` which is an avl tree of
those structs order by the `cmp` function.

I should figure out the cmp function better so that we can actually
have the same struct in multiple trees. Soon.

## Wait, what?

Just "cd tests; go generate && go test ." and read the generated
code. I don't want to document too much since everything will change
soon, this is the scaffolding I need to actually work on this code.
