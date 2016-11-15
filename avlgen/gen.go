package avlgen

import (
	"io"
	"text/template"
)

type Conf struct {
	// Name of the package.
	PackN string
	// Name of the link type.
	LinkT string
	// Name of the tree type.
	TreeT string
	// Name of the node type.
	NodeT string
	// Name of the node member that is our link.
	LinkN string
	// How to compare two nodes...
}

var tmpl *template.Template

func init() {
	t, err := template.New("code").Parse(tmplText)
	if err != nil {
		panic(err)
	}
	tmpl = t
}

func New(nodeT, linkT, linkN, treeT, packN string) *Conf {
	return &Conf{packN, linkT, treeT, nodeT, linkN}
}

func (c *Conf) Gen(out io.Writer) error {
	return tmpl.Execute(out, c)
}

const tmplText = `{{if .PackN}}package {{.PackN}}{{end}}

type {{.LinkT}} struct {
	nodes  [2]{{.TreeT}}
	height int
}

type {{.TreeT}} struct {
	n *{{.NodeT}}
}

func btoi(a bool) int {
	if a {
		return 1
	}
	return 0
}

func (p *{{.TreeT}}) height() int {
	if p.n == nil {
		return 0
	}
	return p.n.{{.LinkN}}.height
}

func (p *{{.TreeT}}) reheight() {
	l := p.n.{{.LinkN}}.nodes[0].height()
	r := p.n.{{.LinkN}}.nodes[1].height()
	if l > r {
		p.n.{{.LinkN}}.height = l + 1
	} else {
		p.n.{{.LinkN}}.height = r + 1
	}
}

func (p *{{.TreeT}}) balance() int {
	return p.n.{{.LinkN}}.nodes[0].height() - p.n.{{.LinkN}}.nodes[1].height()
}

func (p *{{.TreeT}}) rotate(d int) {
	n := p.n
	pivot := n.{{.LinkN}}.nodes[d^1].n
	n.{{.LinkN}}.nodes[d^1].n = pivot.{{.LinkN}}.nodes[d].n
	pivot.{{.LinkN}}.nodes[d].n = n
	pivot.{{.LinkN}}.nodes[d].reheight()
	p.n = pivot
	p.reheight()
}

func (p *{{.TreeT}}) rebalance() {
	n := p.n
	p.reheight()
	bal := p.balance()
	bl, br := bal < -1, bal > 1
	if !(bl || br) {
		return
	}
	d := btoi(bl)
	bd := n.{{.LinkN}}.nodes[d].balance()
	if (bl && bd >= 0) || (br && bd <= 0) {
		n.{{.LinkN}}.nodes[d].rotate(d)
	}
	p.rotate(d ^ 1)
}

func (p *{{.TreeT}}) Insert(x *{{.NodeT}}) {
	if p.n == nil {
		x.{{.LinkN}}.nodes[0].n = nil
		x.{{.LinkN}}.nodes[1].n = nil
		x.{{.LinkN}}.height = 1
		p.n = x
		return
	}
	_, less := x.cmp(p.n)

	/*
	 * We need to decide how to handle equality.
	 *
	 * Four options:
	 * 1. Silently assume it doesn't happen, just insert duplicate elements. (current option)
	 * 2. Silently ignore and don't insert.
	 * 3. Refuse to insert, return boolean for success.
	 * 4. Replace, return old element.
	 *
	 * The _ in the statement above is for equality.
	 */
	p.n.{{.LinkN}}.nodes[btoi(less)].Insert(x)
	p.rebalance()
}

func (a *{{.TreeT}}) Lookup(x *{{.NodeT}}) *{{.NodeT}} {
	n := a.n

	for n != nil {
		eq, less := x.cmp(n)
		if eq {
			break
		}
		n = n.{{.LinkN}}.nodes[btoi(less)].n
	}
	return n
}

func (p *{{.TreeT}}) foreach(b, m, a func(*{{.NodeT}})) {
	if p.n == nil {
		return
	}
	if b != nil {
		b(p.n)
	}
	p.n.{{.LinkN}}.nodes[0].foreach(b, m, a)
	if m != nil {
		m(p.n)
	}
	p.n.{{.LinkN}}.nodes[1].foreach(b, m, a)
	if a != nil {
		a(p.n)
	}
}
`
