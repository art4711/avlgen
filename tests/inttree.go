package trees

//go:generate go run ../cmd/avlgen.go iV tLink tLink iT trees inttree_gen.go

type iV struct {
	v     int
	tLink tLink
}

func (a *iV) cmp(b *iV) (bool, bool) {
	return a.v == b.v, a.v < b.v
}
