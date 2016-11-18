package avlgen

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/template"
)

type Trees struct {
	trees []*conf
	// Name of the package.
	Pkg     string
	Imports map[string]string
}

type conf struct {
	// Name of the link type.
	LinkT string
	// Name of the tree type.
	TreeT string
	// Name of the node type.
	NodeT string
	// Name of the node member that is our link.
	LinkN string
	// How to compare two nodes...
	CmpF string
	// Compare node to value.
	CmpVal     string
	CmpValType string
	// Optional functions to generate
	Foreach bool
	Check   bool
}

func (c *conf) parseTag(tag string) error {
	s := strings.Split(tag, ",")
	// The first element is always the name of the tree type.
	c.TreeT = s[0]
	s = s[1:]
	// The rest of the elements are split key:value pairs
	for i := range s {
		kv := strings.SplitN(s[i], ":", 2)
		if len(kv) == 1 {
			switch kv[0] {
			case "foreach":
				c.Foreach = true
			case "debug":
				c.Foreach = true
				c.Check = true
			default:
				return fmt.Errorf("unknown tag value: %s", s[i])
			}
		} else {
			k, v := kv[0], kv[1]
			switch k {
			case "cmp":
				c.CmpF = v
			case "cmpval":
				m := regexp.MustCompile("(.*)\\((.*)\\)").FindStringSubmatch(v)
				if len(m) != 3 {
					return fmt.Errorf("invalid cmpval, expected 'cmpval:<fn>(<type>)', got 'cmpval:%s'", v)
				}
				c.CmpVal = m[1]
				c.CmpValType = m[2]
			default:
				return fmt.Errorf("unknown tag value: %s", s[i])
			}
		}
	}
	return nil
}

func New(pkg string) *Trees {
	return &Trees{Pkg: pkg, Imports: make(map[string]string)}
}

func (t *Trees) AddTree(nodeT, linkT, linkN, treeT, tag string) error {
	c := &conf{
		LinkT: linkT,
		TreeT: treeT,
		NodeT: nodeT,
		LinkN: linkN,
		CmpF:  "cmp",
	}
	if tag != "" {
		err := c.parseTag(tag)
		if err != nil {
			return err
		}
	}
	if c.Check {
		t.Imports["fmt"] = "fmt"
	}
	t.trees = append(t.trees, c)
	return nil
}

func (t *Trees) Gen(out io.Writer) error {
	err := prologueTmpl.Execute(out, t)
	if err != nil {
		return err
	}
	for _, c := range t.trees {
		err := treeTmpl.Execute(out, c)
		if err != nil {
			return err
		}
	}
	return nil
}

var prologueTmpl = template.Must(template.New("prologue").Parse(`// Code generated by "avlgen", do not edit.

package {{.Pkg}}
{{range .Imports}}
import "{{.}}"
{{end}}

func btoi(a bool) int {
	// See: https://github.com/golang/go/issues/6011#issuecomment-254303032
	//
	// Hopefully this will make the generated code suck less in the
	// future because currently it's comically bad. false is branch
	// predicted by the compiler to be unlikely, it's put after the
	// function, jumps back, then the compiler forgets that it just
	// loaded 0 or 1 into a register and does a bounds check on a 2
	// element array.
	x := 0
	if a {
		x = 1
	}
	// By performing this '& 1' we add one useless instruction but
	// we eliminate the bounds check which is a branch.
	// Yes, it is worth it.
	return x & 1
}
`))

var treeTmpl = template.Must(template.New("code").Parse(`
type {{.LinkT}} struct {
	nodes  [2]{{.TreeT}}
	height int
}

type {{.TreeT}} struct {
	n *{{.NodeT}}
}

func (tr *{{.TreeT}}) height() int {
	if tr.n == nil {
		return 0
	}
	return tr.n.{{.LinkN}}.height
}

func (tr *{{.TreeT}}) reheight() {
	l := tr.n.{{.LinkN}}.nodes[0].height()
	r := tr.n.{{.LinkN}}.nodes[1].height()
	if l > r {
		tr.n.{{.LinkN}}.height = l + 1
	} else {
		tr.n.{{.LinkN}}.height = r + 1
	}
}

func (tr *{{.TreeT}}) balance() int {
	return tr.n.{{.LinkN}}.nodes[0].height() - tr.n.{{.LinkN}}.nodes[1].height()
}

func (tr *{{.TreeT}}) rotate(d int) {
	n := tr.n
	d &= 1			// Remind the compiler that d is 0 or 1 to eliminate bounds checks
	notd := (d ^ 1) & 1
	pivot := n.{{.LinkN}}.nodes[notd].n
	n.{{.LinkN}}.nodes[notd].n = pivot.{{.LinkN}}.nodes[d].n
	pivot.{{.LinkN}}.nodes[d].n = n
	pivot.{{.LinkN}}.nodes[d].reheight()
	tr.n = pivot
	tr.reheight()
}

func (tr *{{.TreeT}}) rebalance() {
	n := tr.n
	tr.reheight()
	bal := tr.balance()
	bl, br := bal < -1, bal > 1
	if !(bl || br) {
		return
	}
	d := btoi(bl)
	bd := n.{{.LinkN}}.nodes[d].balance()
	if (bl && bd >= 0) || (br && bd <= 0) {
		n.{{.LinkN}}.nodes[d].rotate(d)
	}
	tr.rotate((d^1)&1)
}

func (tr *{{.TreeT}}) insert(x *{{.NodeT}}) {
	if tr.n == nil {
		x.{{.LinkN}}.nodes[0].n = nil
		x.{{.LinkN}}.nodes[1].n = nil
		x.{{.LinkN}}.height = 1
		tr.n = x
		return
	}
	_, less := x.{{.CmpF}}(tr.n)

	/*
	 * We need to decide how to handle equality.
	 *
	 * Four options:
	 * 1. Silently assume it doesn't happen, just insert
	 *    duplicate elements. It's your foot and your
	 *    trigger. (current choice)
	 * 2. Silently ignore and don't insert.
	 * 3. Refuse to insert, return boolean for success.
	 * 4. Replace, return old element.
	 *
	 * The _ in the statement above is for equality.
	 */
	tr.n.{{.LinkN}}.nodes[btoi(less)].insert(x)
	tr.rebalance()
}

func (tr *{{.TreeT}}) delete(x *{{.NodeT}}) {
	/*
	 * We silently ignore deletions of elements that are
	 * not in the tree. The options here are to return
	 * something or panic or do nothing. All three equally
	 * valid.
	 */
	if tr.n == nil {
		return
	}

	if tr.n == x {
		if tr.n.{{.LinkN}}.nodes[0].n == nil {
			tr.n = tr.n.{{.LinkN}}.nodes[1].n
		} else if tr.n.{{.LinkN}}.nodes[1].n == nil {
			tr.n = tr.n.{{.LinkN}}.nodes[0].n
		} else {
			r := tr.n.{{.LinkN}}.nodes[0].n
			for r.{{.LinkN}}.nodes[1].n != nil {
				r = r.{{.LinkN}}.nodes[1].n
			}
			tr.n.{{.LinkN}}.nodes[0].delete(r)
			r.{{.LinkN}}.nodes = tr.n.{{.LinkN}}.nodes
			tr.n = r
			tr.reheight()
		}
	} else {
		_, less := x.{{.CmpF}}(tr.n)
		tr.n.{{.LinkN}}.nodes[btoi(less)].delete(x)
		tr.rebalance()
	}
}

func (tr *{{.TreeT}}) lookup(x *{{.NodeT}}) *{{.NodeT}} {
	n := tr.n

	for n != nil {
		eq, less := x.{{.CmpF}}(n)
		if eq {
			break
		}
		n = n.{{.LinkN}}.nodes[btoi(less)].n
	}
	return n
}

{{if .CmpVal}}
func (tr *{{.TreeT}})lookupVal(x {{.CmpValType}}) *{{.NodeT}} {
	n := tr.n
	for n != nil {
		// notice that the compare order is reversed to how lookup does, so less is more.
		eq, more := n.{{.CmpVal}}(x)
		if eq {
			break
		}
		n = n.{{.LinkN}}.nodes[btoi(!more)].n
	}
	return n
}

// Find nearest value greater than or equal to x
func (tr *{{.TreeT}})searchValGEQ(x {{.CmpValType}}) *{{.NodeT}} {
	// Empty tree can't match.
	if tr.n == nil {
		return nil
	}
	eq, more := tr.n.{{.CmpVal}}(x)
	if eq {
		return tr.n
	}
	if !more {
		l := tr.n.{{.LinkN}}.nodes[1].searchValGEQ(x)
		if l != nil {
			_, less := l.{{.CmpF}}(tr.n)
			if less {
				return l
			}
		}
		return tr.n
	}
	return tr.n.{{.LinkN}}.nodes[0].searchValGEQ(x)
}

// Find nearest value less than or equal to x
func (tr *{{.TreeT}})searchValLEQ(x {{.CmpValType}}) *{{.NodeT}} {
	// Empty tree can't match.
	if tr.n == nil {
		return nil
	}
	eq, more := tr.n.{{.CmpVal}}(x)
	if eq {
		return tr.n
	}
	if more {
		l := tr.n.{{.LinkN}}.nodes[0].searchValLEQ(x)
		if l != nil {
			_, less := l.{{.CmpF}}(tr.n)
			if !less {
				return l
			}
		}
		return tr.n
	}
	return tr.n.{{.LinkN}}.nodes[1].searchValLEQ(x)
}

func (tr *{{.TreeT}}) deleteVal(x {{.CmpValType}}) {
	/*
	 * We silently ignore deletions of elements that are
	 * not in the tree. The options here are to return
	 * something or panic or do nothing. All three equally
	 * valid.
	 */
	if tr.n == nil {
		return
	}

	eq, more := tr.n.{{.CmpVal}}(x)
	if eq {
		if tr.n.{{.LinkN}}.nodes[0].n == nil {
			tr.n = tr.n.{{.LinkN}}.nodes[1].n
		} else if tr.n.{{.LinkN}}.nodes[1].n == nil {
			tr.n = tr.n.{{.LinkN}}.nodes[0].n
		} else {
			r := tr.n.{{.LinkN}}.nodes[0].n
			for r.{{.LinkN}}.nodes[1].n != nil {
				r = r.{{.LinkN}}.nodes[1].n
			}
			tr.n.{{.LinkN}}.nodes[0].delete(r)
			r.{{.LinkN}}.nodes = tr.n.{{.LinkN}}.nodes
			tr.n = r
			tr.reheight()
		}
	} else {
		tr.n.{{.LinkN}}.nodes[btoi(!more)].deleteVal(x)
		tr.rebalance()
	}
}
{{end}}
{{if .Foreach}}
func (tr *{{.TreeT}}) foreach(b, m, a func(*{{.NodeT}})) {
	if tr.n == nil {
		return
	}
	if b != nil {
		b(tr.n)
	}
	tr.n.{{.LinkN}}.nodes[0].foreach(b, m, a)
	if m != nil {
		m(tr.n)
	}
	tr.n.{{.LinkN}}.nodes[1].foreach(b, m, a)
	if a != nil {
		a(tr.n)
	}
}
{{end}}
{{if .Check}}
// This function has a bit wonky prototype, but it's
// more natural to foreach on nodes and we want to
// be able to plug this into foreach.
func (tr *{{.TreeT}}) check(n *{{.NodeT}}) error {
	lh := n.{{.LinkN}}.nodes[0].height()
	rh := n.{{.LinkN}}.nodes[1].height()
	nh := n.{{.LinkN}}.height
	// Verify height invariants
	if lh > rh {
		if nh != lh +1 {
			return fmt.Errorf("nodes[0].height %d + 1 != n.height %d", lh, nh)
		}
	} else {
		if nh != rh +1 {
			return fmt.Errorf("nodes[1] height %d + 1 != n.height %d", rh, nh)
		}
	}
	balance := lh - rh
	if balance < -1 || balance > 1 {
		return fmt.Errorf("bad balance: %d %d %d", nh, rh, lh)
	}
	ln := n.{{.LinkN}}.nodes[0].n
	rn := n.{{.LinkN}}.nodes[1].n
	if ln != nil {
		_, less := ln.{{.CmpF}}(n)
		if less {
			return fmt.Errorf("left %v < %v", ln, n)
		}
	}
	if rn != nil {
		_, less := rn.{{.CmpF}}(n)
		if !less {
			return fmt.Errorf("right %v < %v", rn, n)
		}
	}
	return nil
}
{{end}}
`))
