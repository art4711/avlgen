# avl trees [![godoc reference](https://godoc.org/github.com/art4711/avlgen/cmd/avlgen?status.png)](https://godoc.org/github.com/art4711/avlgen/cmd/avlgen)

Who needs templates when you have go generate.

## What?

This is a Go implementation of an inline (some people call it embedded)
AVL tree very heavily based on [my C implementation](https://github.com/art4711/stuff/tree/master/avl).

Since we don't have templates and there's no pre-processor and we
can't commit pointer atrocities we `go generate` the necessary code
instead.

## How?

In short you need something like this in `anyfile.go`:

    //go:generate go run <figure out path later>/cmd/avlgen/main.go .
    type anyStruct struct {
        anyFieldName anyTypeName `avlgen:"nameOfTreeType"`
    }

    func (a *anyStruct)cmp(b *anyStruct) (bool, bool) {
        return a.Equals(b), a.Less(b)
    }

Running `go generate` will create `<package name>_trees.go` that
contains the implementation of the type `nameOfTreeType` which is an
avl tree of those structs order by the `cmp` function. This will be
applied to all files in the package so you have have as many trees
as you'd like for the same package.

`anyFieldName anyTypeName` is the linkage we need into the tree. This
is the part that embeds the data structure into your structure.

The two interesting functions that will be generated will be
`(*nameOfTreeType).insert` and `(*nameOfTreeType).lookup` (delete,
search and a few special ones coming soon). The functions/type aren't
exported on purpose. If you want to export them, wrap the generated
struct or bug me and I'll add a flag.

## the struct tag

The tag can contain other configuration for the generated code.
Currently this is possible:

    `avlgen:"name,cmp:cmpfun,cmpval:cmpvalfun(int)"`

This will generate the tree type `name` using `cmpfun` as the standard
compare function and `cmvalfun(int)` as a special optimized lookup compare
function. The default name for `cmp` is a function named `cmp`.

If `cmpval` is specified it is just like the compare function, but
instead of taking a node struct pointer as an argument, you can pass a
key type directly. Removing the necessity of creating a fake struct
for the lookup funciton. The type of the argument is whatever is
inside the parentheses in the tag. The generator will then generate a
`(*nameOfTreeType).lookupVal(<key type>)` function that can be used to
more efficiently look up values in the tree.

## Wait, what?

Just `cd tests; go generate && go test .` and read the generated
code. I don't want to document too much since everything will change
soon, this is the scaffolding I need to actually work on this code.

## Is this really useful?

I'm generally happy with a small selection of primitives to work with.
Arrays, maps and convoluted structs with pointers are sufficient for
almost all problems.

But I now need a data structure where I add and remove elements often
and it needs to be ordered. I can't get away from the ordering
requirement and I can't afford to dump a map into an array and sort
it every time I need the data ordered, because it's all the time.

## Why don't you implement RB trees instead? RB trees are webscale.

To be fair, people don't say that RB trees are webscale, they instead
say "they need fewer rebalance operations". But this bloody argument
is repeated just as often and as mindlessly as in that famous video
about mongodb being webscale.

I have not seen any benchmark in the past 10-15 years where a good
implementation of AVL isn't the same or faster than RB. None. I'm
happy to be proven wrong.

I suspect this is because AVL trees are always shallower than RB
trees. So the extra cost of more rebalancing is paid off, with
interest, by using less cache. 20 years ago memory writes were
probably expensive, today cache is king. Also. The code for AVL trees
is trivial. Especially the brutally optimized version I have here. In
C it's almost branchless (can't be branchless, but it's close). Why?
Because instead of copying `left` and `right` from textbooks we have
an array with two elements and we index it with booleans. That removes
most branches. And allows us to replace 4 rotation functions with one
that's branchless. Branches matter on modern CPUs. A lot. All this is
a lie in this implementation because `btoi` ruins everything in Go.

## wtf is btoi?

Unfortunately, everything I said in the last paragraph only applies to
this implementation if it's compiled with a good C compiler. The Go
compiler is comically bad at generating good code here. I agree that
the language shouldn't define numerical values for booleans, but in
return the compiler should understand how to optimize this:

    var arr [2]*Foo
    var b bool
    b := cmp(...)
    x := 0
    if b {
        x = 1
    }
    a := arr[x]

We repeat this pattern all over the code to do something on the left
or right sides of the tree.

A C compiler generates something like this (the exact code wasn't like
above, but it was close enough): (we are just after the comparison)

    // Set al to 1 if the result of the last compare was greater.
    setg   %al
    // zero-extend to a full 64 bit value
    movzbl %al,%eax
    // index the array and put result in rbx
    mov    (%rbx,%rax,8),%rbx

The Go compiler (1.7.3) manages:

    // If the result of the last compare was greater jump way out.
    46e90c: jge    46e94d
    // Set bx to 1
    46e90e: mov    $0x1,%rbx
    // Quickly forget that we just set bx to 0 or 1 and check if it
    // by some accident isn't bigger than 2.
    46e915: cmp    $0x2,%rbx
    // I guess bounds check elimination isn't part of our business
    // strategy. Check those bounds, check them hard.
    46e919: jae    46e946
    // finally.
    46e91b: lea    0x10(%rcx,%rbx,8),%rcx
    [... rest of the function here ...]
    
    // Back of the function. We were predicted to be unlikely so the
    // code for false was put at the back of the function.
    // set bx to 0
    46e94d: xor    %ebx,%ebx
    // jump back to the action, just past the instruction that
    // set bx to 1
    46e94f: jmp    46e915

Let's do three branches instead of three instructions.

I still haven't found a way to fix this. There is a mention in an
issue that the compiler in tip (1.8?) should be able to deal with
this better: https://github.com/golang/go/issues/6011#issuecomment-254303032

Update: I managed to remove the bounds check by doing a useless `& 1`
in a few strategic places, replacing a branch with one useless
instruction, but we still get the expensive branch for converting the
boolean to int.

## Benchmarketing

I did a quick benchmark of inserting a million int,int pairs into the
AVL tree (one of the ints is the key) and compared it to map[int]int.
The results are pretty meaningless (because of btoi and because of
different use cases), but AVL trees are 35% slower at insertions and
30% faster at lookups. Comparing a struct with string,string pair to a
map[string]string shows marginally faster (within 5%) insert speed for
insertions and around 20% slower lookup speed.

The big win here is that the tree allocates approximately half the
memory in all tested cases.
