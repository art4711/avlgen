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
	// Name of the iterator type.
	IterT string

	// Generated function names
	F map[string]string
}

func (c *conf) parseTag(tag string) error {
	s := strings.Split(tag, ",")
	// The first element is always the name of the tree type.
	c.TreeT = s[0]
	s = s[1:]
	export := false
	// The rest of the elements are split key:value pairs
	for i := range s {
		kv := strings.SplitN(s[i], ":", 2)
		if len(kv) == 1 {
			switch kv[0] {
			case "foreach":
				c.F["foreach"] = "foreach"
			case "debug":
				c.F["foreach"] = "foreach"
				c.F["check"] = "check"
			case "iter":
				c.IterT = c.TreeT + "Iter"
			case "export":
				export = true
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
			case "no":
				delete(c.F, v)
			default:
				return fmt.Errorf("unknown tag value: %s", s[i])
			}
		}
	}
	if export {
		for k, v := range c.F {
			c.F[k] = strings.Title(v)
		}
	}
	return nil
}

func New(pkg string) *Trees {
	return &Trees{Pkg: pkg, Imports: make(map[string]string)}
}

var defaultFuncs = map[string]string{
	"insert":       "insert",
	"delete":       "delete",
	"lookup":       "lookup",
	"last":         "last",
	"first":        "first",
	"lookupVal":    "lookupVal",
	"searchValGEQ": "searchValGEQ",
	"searchValLEQ": "searchValLEQ",
	"deleteVal":    "deleteVal",
	"iter":         "iter",
	"iterVal":      "iterVal",
}

func (t *Trees) AddTree(nodeT, linkT, linkN, treeT, tag string) error {
	c := &conf{
		LinkT: linkT,
		TreeT: treeT,
		NodeT: nodeT,
		LinkN: linkN,
		CmpF:  "cmp",
		F:     make(map[string]string),
	}
	for k, v := range defaultFuncs {
		c.F[k] = v
	}
	if tag != "" {
		err := c.parseTag(tag)
		if err != nil {
			return err
		}
	}
	if c.F["check"] != "" {
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
	d &= 1 // Remind the compiler that d is 0 or 1 to eliminate bounds checks
	notd := (d ^ 1) & 1

	pivot := n.{{.LinkN}}.nodes[notd].n
	n.{{.LinkN}}.nodes[notd].n = pivot.{{.LinkN}}.nodes[d].n
	pivot.{{.LinkN}}.nodes[d].n = n
	pivot.{{.LinkN}}.nodes[d].reheight()
	tr.n = pivot
	tr.reheight()
}

func (tr *{{.TreeT}}) rebalance() {
	tr.reheight()
	bal := tr.balance()
	bl, br := bal < -1, bal > 1
	if !(bl || br) {
		return
	}
	d := btoi(bl)
	bd := tr.n.{{.LinkN}}.nodes[d].balance()
	if (bl && bd > 0) || (br && bd < 0) {
		tr.n.{{.LinkN}}.nodes[d].rotate(d)
	}
	tr.rotate(d ^ 1)
}
{{- if .F.insert}}

func (tr *{{.TreeT}}) {{.F.insert}}(x *{{.NodeT}}) {
	path := [64]*{{.TreeT}}{}
	depth := 0
	for tr.n != nil {
		path[depth] = tr
		depth++
		_, less := tr.n.{{.CmpF}}(x)
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
		 */
		tr = &tr.n.{{.LinkN}}.nodes[btoi(!less)]
	}
	x.{{.LinkN}}.nodes[0].n = nil
	x.{{.LinkN}}.nodes[1].n = nil
	x.{{.LinkN}}.height = 1
	tr.n = x

	for i := depth-1; i >= 0; i-- {
		path[i].rebalance()
	}
}
{{- end -}}
{{- if .F.delete}}

func (tr *{{.TreeT}}) {{.F.delete}}(x *{{.NodeT}}) {
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
			next := tr.n.{{.LinkN}}.nodes[0].{{.F.first}}()
			tr.n.{{.LinkN}}.nodes[0].{{.F.delete}}(next)
			next.{{.LinkN}} = tr.n.{{.LinkN}}
			tr.n = next
			tr.rebalance()
		}
	} else {
		_, less := tr.n.{{.CmpF}}(x)
		tr.n.{{.LinkN}}.nodes[btoi(!less)].{{.F.delete}}(x)
		tr.rebalance()
	}
}
{{- end -}}
{{- if .F.lookup}}

func (tr *{{.TreeT}}) {{.F.lookup}}(x *{{.NodeT}}) *{{.NodeT}} {
	n := tr.n

	for n != nil {
		eq, less := n.{{.CmpF}}(x)
		if eq {
			break
		}
		n = n.{{.LinkN}}.nodes[btoi(!less)].n
	}
	return n
}
{{- end -}}
{{- if .F.last}}

func (tr *{{.TreeT}}) {{.F.last}}() (ret *{{.NodeT}}) {
	for n := tr.n; n != nil; n = n.{{.LinkN}}.nodes[0].n {
		ret = n
	}
	return
}
{{- end -}}
{{- if .F.first}}

func (tr *{{.TreeT}}) {{.F.first}}() (ret *{{.NodeT}}) {
	for n := tr.n; n != nil; n = n.{{.LinkN}}.nodes[1].n {
		ret = n
	}
	return
}
{{- end -}}
{{- if .CmpVal -}}
{{- if .F.lookupVal}}

func (tr *{{.TreeT}}) {{.F.lookupVal}}(x {{.CmpValType}}) *{{.NodeT}} {
	n := tr.n
	for n != nil {
		// notice that the compare order is reversed to how lookup does, so less is more.
		eq, less := n.{{.CmpVal}}(x)
		if eq {
			break
		}
		n = n.{{.LinkN}}.nodes[btoi(!less)].n
	}
	return n
}
{{- end -}}
{{- if .F.searchValGEQ}}

// Find nearest value greater than or equal to x
func (tr *{{.TreeT}}) {{.F.searchValGEQ}}(x {{.CmpValType}}) *{{.NodeT}} {
	// Empty tree can't match.
	if tr.n == nil {
		return nil
	}
	eq, less := tr.n.{{.CmpVal}}(x)
	if eq {
		return tr.n
	}
	if !less {
		l := tr.n.{{.LinkN}}.nodes[1].{{.F.searchValGEQ}}(x)
		if l != nil {
			_, less := tr.n.{{.CmpF}}(l)
			if !less {
				return l
			}
		}
		return tr.n
	}
	return tr.n.{{.LinkN}}.nodes[0].{{.F.searchValGEQ}}(x)
}
{{- end -}}
{{- if .F.searchValLEQ}}

// Find nearest value less than or equal to x
func (tr *{{.TreeT}}) {{.F.searchValLEQ}}(x {{.CmpValType}}) *{{.NodeT}} {
	// Empty tree can't match.
	if tr.n == nil {
		return nil
	}
	eq, less := tr.n.{{.CmpVal}}(x)
	if eq {
		return tr.n
	}
	if less {
		l := tr.n.{{.LinkN}}.nodes[0].{{.F.searchValLEQ}}(x)
		if l != nil {
			_, less := tr.n.{{.CmpF}}(l)
			if less {
				return l
			}
		}
		return tr.n
	}
	return tr.n.{{.LinkN}}.nodes[1].{{.F.searchValLEQ}}(x)
}
{{- end -}}
{{- if .F.deleteVal}}

func (tr *{{.TreeT}}) {{.F.deleteVal}}(x {{.CmpValType}}) {
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
			next := tr.n.{{.LinkN}}.nodes[0].{{.F.first}}()
			tr.n.{{.LinkN}}.nodes[0].{{.F.delete}}(next)
			next.{{.LinkN}} = tr.n.{{.LinkN}}
			tr.n = next
			tr.rebalance()
		}
	} else {
		tr.n.{{.LinkN}}.nodes[btoi(!more)].{{.F.deleteVal}}(x)
		tr.rebalance()
	}
}
{{- end -}}
{{- end -}}
{{- if .IterT}}

type {{.IterT}} struct {
	// First and last elements of the iterator
	start, end *{{.NodeT}}
	// Should start and end elements be included in the iteration?
	incs, ince, rev bool
	// The path we took to reach the previous element.
	path []*{{.TreeT}}
}
{{- if .F.iter}}

func (tr *{{.TreeT}}) {{.F.iter}}(start, end *{{.NodeT}}, incs, ince bool) *{{.IterT}} {
	it := &{{.IterT}}{start: start, end: end, incs: incs, ince: ince, path: make([]*{{.TreeT}}, 0, tr.height())}
	if start != nil {
		it.findStartPath(tr)
	} else {
		it.diveDown(tr)
	}
	if end == nil {
		it.end = tr.{{.F.last}}()
	}
	// Explicitly handle start == end.
	if it.start == it.end && it.incs != it.ince {
		// one false means both false
		it.incs = false
		it.ince = false
	}
	eq, less := it.start.{{.CmpF}}(it.end)
	it.rev = !less && !eq
	return it
}
{{- end -}}
{{- if .CmpVal }}
{{- if .F.iterVal}}

// start, end - start and end values of iteration.
// edgeStart,edgeEnd - ignore start/end and start/end the iteration at the edge of the tree.
// incs, ince - include the start/end value in the iteration.
func (tr *{{.TreeT}}) {{.F.iterVal}}(start, end {{.CmpValType}}, edgeStart, edgeEnd, incs, ince bool) *{{.IterT}} {
	var s, e *{{.NodeT}}
	if !edgeStart {
		s = tr.{{.F.searchValLEQ}}(start)
		if eq, _ := s.{{.CmpVal}}(start); !eq {
			// If we got a value less than start,
			// force incs to false since we don't
			// want to include it.
			incs = false
		}
	}
	if !edgeEnd {
		e = tr.{{.F.searchValGEQ}}(end)
		if eq, _ := e.{{.CmpVal}}(end); !eq {
			// If we got a value greater than end,
			// force ince to false since we don't
			// want to include it.
			ince = false
		}
	}
	return tr.{{.F.iter}}(s, e, incs, ince)
}
{{- end -}}
{{- end}}

// Helper function, don't use.
func (it *{{.IterT}}) diveDown(t *{{.TreeT}}) {
	for t.n != nil {
		it.path = append(it.path, t)
		it.start = t.n // lazy, should just be done once.
		t = &t.n.{{.LinkN}}.nodes[btoi(!it.rev)]
	}
}

// Helper function, don't use.
func (it *{{.IterT}}) findStartPath(t *{{.TreeT}}) {
	for {
		it.path = append(it.path, t)
		eq, less := t.n.{{.CmpF}}(it.start)
		if eq {
			break
		}
		t = &t.n.{{.LinkN}}.nodes[btoi(!less)]
	}
}

func (it *{{.IterT}}) value() *{{.NodeT}} {
	return it.start
}

func (it *{{.IterT}}) next() bool {
	if it.start != it.end {
		// incs can only be set for the first element of the iterator,
		// if it is, we just don't move to the next element.
		if it.incs {
			it.incs = false
			return true
		}
		/*
		 * right - towards the end of iteration (0 in forward iteration)
		 * left - towards beginning of the iteration (1 in forward iteration)
		 *
		 * Last returned element is it.start
		 * We got it through t := it.path[len(it.path)-1].
		 * if t has a tree to the right, the next element
		 * is the leftmost element of the right tree.
		 * If it doesn't, the next element is the one parent
		 * we have that's bigger than us.
		 *
		 * We don't check for underflow of path. If that
		 * happens something is already seriously wrong,
		 * crashing is the best option.
		 */
		if it.start.{{.LinkN}}.nodes[btoi(it.rev)].n != nil {
			it.diveDown(&it.start.{{.LinkN}}.nodes[btoi(it.rev)])
		} else {
			for {
				it.path = it.path[:len(it.path)-1]
				_, less := it.path[len(it.path)-1].n.{{.CmpF}}(it.start)
				if less == it.rev {
					break
				}
			}
			it.start = it.path[len(it.path)-1].n
		}
	}
	if it.start != it.end {
		return true
	} else if it.ince {
		it.ince = false
		return it.end != nil // can happen with empty iterator.
	} else {
		return false
	}
}
{{- end -}}
{{- if .F.foreach}}

func (tr *{{.TreeT}}) {{.F.foreach}}(b, m, a func(*{{.NodeT}})) {
	if tr.n == nil {
		return
	}
	if b != nil {
		b(tr.n)
	}
	tr.n.{{.LinkN}}.nodes[0].{{.F.foreach}}(b, m, a)
	if m != nil {
		m(tr.n)
	}
	tr.n.{{.LinkN}}.nodes[1].{{.F.foreach}}(b, m, a)
	if a != nil {
		a(tr.n)
	}
}
{{- end -}}
{{- if .F.check}}

// This function has a bit wonky prototype, but it's
// more natural to foreach on nodes and we want to
// be able to plug this into foreach.
func (tr *{{.TreeT}}) {{.F.check}}(n *{{.NodeT}}) error {
	lh := n.{{.LinkN}}.nodes[0].height()
	rh := n.{{.LinkN}}.nodes[1].height()
	nh := n.{{.LinkN}}.height
	// Verify height invariants
	eh := rh + 1
	if lh > rh {
		eh = lh + 1
	}
	balance := lh - rh
	if eh != nh || balance < -1 || balance > 1 {
		return fmt.Errorf("bad balance: %d %d %d, %V", nh, rh, lh, n)
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
{{- end}}
`))
