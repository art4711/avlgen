package trees

type ss struct {
	k, v string
	tsl  tsl `avlgen:"sst,cmpval:cmpk(string)"`
}

func (a *ss) cmp(b *ss) (bool, bool) {
	return a.k == b.k, a.k < b.k
}

func (a *ss) cmpk(b string) (bool, bool) {
	return a.k == b, a.k < b
}
