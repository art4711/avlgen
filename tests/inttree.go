package trees

//go:generate go run ../cmd/avlgen/main.go -- inttree.go

type iV struct {
	v     int
	tlink tLink `avlgen:"iT"`
}

func (a *iV) cmp(b *iV) (bool, bool) {
	return a.v == b.v, a.v < b.v
}
