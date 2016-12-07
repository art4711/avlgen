# avl trees [![godoc reference](https://godoc.org/github.com/art4711/avlgen/cmd/avlgen?status.png)](https://godoc.org/github.com/art4711/avlgen/cmd/avlgen)

Who needs generics when you have go generate.

## What?

This is a Go implementation of an embedded/inline AVL tree very
heavily based on [my C implementation](https://github.com/art4711/stuff/tree/master/avl).

Since we don't have generics and there's no pre-processor and we can't
commit pointer atrocities we `go generate` the necessary code instead.

## Why?

I needed something like a map, but ordered. And I don't like my
data structures slow and inefficient. So I made this.

Embedded data structures have the advantage that they require no
additional memory allocation on most operations. Everything that's
needed to keep track of the data is part of the data. This allows an
implementation with little overhead. Fast, memory efficient and
self-contained.

A common criticism of embedded data structures is that they lead to
lifetime management issues. It's easy to forget to remove a struct
from a list/tree before freeing it. It's not really a problem here
because garbage collection. We can't get dangling pointers.

## How?

Read the [documentation](https://godoc.org/github.com/art4711/avlgen/cmd/avlgen).

## Wait, what?

Just `cd tests; go generate && go test .` and read the generated file.

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
implementation of AVL doesn't perform the same or faster than RB.
None. I'm happy to be proven wrong.

I suspect this is because AVL trees are always shallower than RB
trees. So the extra cost of more rebalancing is paid off, with
interest, by using less cache. 20 years ago maybe the balance was
different and instructions heavy to execute, today cache is king.

Also. The code for AVL trees is trivial. Especially the brutally
optimized version I have here. In C it's almost branchless (can't be
branchless, but it's as close as we can get). Why?  Because instead of
copying `left` and `right` from textbooks we have an array with two
elements and we index it with booleans. That removes most
branches. And allows us to replace a handful of rotation functions
with one that's branchless. Branches matter on modern CPUs. A lot. All
this is a lie in this implementation because `btoi` ruins everything
in Go.

## wtf is `btoi`?

Unfortunately, everything I said in the previous paragraph only
applies to this implementation if it's compiled with a good C
compiler. The Go compiler is quite bad at generating good code
here. I agree that the language shouldn't define numerical values for
booleans, but in return the compiler should understand how to optimize
this:

    var arr [2]*Foo
    var b bool
    b = cmp(...)
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

Update: I managed to remove the bounds check by doing a useless `& 1`
in a few strategic places, replacing a branch with one useless
instruction, but we still get the expensive branch for converting the
boolean to int.

## Use 1.8

The compiler in Go 1.8beta1 generates good code here. The speedups are
significant, up to 30% on certain benchmarks.

As a minor surprise, the 1.7.3 compiled (branching) version was faster
than 1.8beta1 when doing lookups in the tree in the same order as the
elements were inserted (which is a pretty unrealistic scenario, the
benchmark was terribly stupid so I fixed it). If I had to speculate it
is because we were accessing the elements in the same order they were
allocated and the allocator probably allocates them pretty much
linearly, so everything comes in a nice cache order. The branches
probably triggered speculative execution which caused prefetching of
cache lines we'd need soon anyway. The 1.8 code on the other hand
probably makes the CPU wait for the address to be fully computed
before fetching things from memory, so we didn't get the accidental
prefetch. The accidental prefetch helped linear access (since we'd
need the cache lines) while it hurt the non-linear access (since we'd
pollute cache with unnecessary stuff).

## Benchmarketing

If you read this section before it looked much better. I've
implemented a less stupid benchmark now.

With a million int,int pairs in a tree vs. `map[int]int` the map is
somewhere around 5-10x faster. As it should be, we should be touching
around 20x more cache lines for every operation. The map starts
outperforming the tree somewhere between 100 and 1000 elements.

The big win here is that the tree allocates approximately half the
memory of map in all tested cases. And it's ordered.

## Should this tool generate other embedded data structures?

It's pretty trivial to add other data structures to this tool. The tag
would need to change just a little. The biggest change would be
renaming the tool. I've decided against this because I currently have
no particular need for anything other than ordered sets. Even though
I miss TAILQs sometimes.
