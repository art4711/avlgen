package avlgen

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/template"
)

type Conf struct {
	First bool
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
	CmpF string
	// Compare node to value.
	CmpVal     string
	CmpValType string
}

var tmpl *template.Template

func init() {
	t, err := template.New("code").Parse(tmplText)
	if err != nil {
		panic(err)
	}
	tmpl = t
}

func (c *Conf) parseTag(tag string) error {
	s := strings.Split(tag, ",")
	// The first element is always the name of the tree type.
	c.TreeT = s[0]
	s = s[1:]
	// The rest of the elements are split key:value pairs
	for i := range s {
		kv := strings.SplitN(s[i], ":", 2)
		if len(kv) != 2 {
			return fmt.Errorf("invalid tag format, expected '<key>:<value>', got '%s'", s[i])
		}
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
		}
	}
	return nil
}

func New(nodeT, linkT, linkN, treeT, packN string, first bool, tag string) (*Conf, error) {
	c := &Conf{
		First: first,
		PackN: packN,
		LinkT: linkT,
		TreeT: treeT,
		NodeT: nodeT,
		LinkN: linkN,
		CmpF:  "cmp",
	}
	if tag != "" {
		err := c.parseTag(tag)
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *Conf) Gen(out io.Writer) error {
	return tmpl.Execute(out, c)
}

const tmplText = `{{if .First}}package {{.PackN}}

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
	return x
}
{{end}}

type {{.LinkT}} struct {
	nodes  [2]{{.TreeT}}
	height int
}

type {{.TreeT}} struct {
	n *{{.NodeT}}
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

func (p *{{.TreeT}}) insert(x *{{.NodeT}}) {
	if p.n == nil {
		x.{{.LinkN}}.nodes[0].n = nil
		x.{{.LinkN}}.nodes[1].n = nil
		x.{{.LinkN}}.height = 1
		p.n = x
		return
	}
	_, less := x.{{.CmpF}}(p.n)

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
	p.n.{{.LinkN}}.nodes[btoi(less)].insert(x)
	p.rebalance()
}

func (a *{{.TreeT}}) lookup(x *{{.NodeT}}) *{{.NodeT}} {
	n := a.n

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

func (a *{{.TreeT}})lookupVal(x {{.CmpValType}}) *{{.NodeT}} {
	n := a.n
	for n != nil {
		eq, more := n.{{.CmpVal}}(x)
		if eq {
			break
		}
		// notice that the compare order is reversed to how lookup does, so less is more.
		n = n.{{.LinkN}}.nodes[btoi(!more)].n
	}
	return n
}
{{end}}

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
