package trees

import "testing"

type mt struct {
	x, y int
	mtlx mtlx `avlgen:"mtx,cmp:cmpx,no:last"`
	mtly mtly `avlgen:"mty,cmp:cmpy,no:delete,no:first,no:last,export"`
}

func (a *mt) cmpx(b *mt) (bool, bool) {
	return a.x == b.x, a.x < b.x
}

func (a *mt) cmpy(b *mt) (bool, bool) {
	return a.y == b.y, a.y < b.y
}

func TestMultiTree(t *testing.T) {
	tx := mtx{}
	ty := mty{}

	vals := []int{5, 2, 6, 3, 1, 7, 4}
	for i := range vals {
		m := &mt{x: vals[i], y: vals[len(vals)-1-i]}
		tx.insert(m)
		ty.Insert(m)
	}
	for i := 1; i <= 7; i++ {
		tx.delete(tx.lookup(&mt{x: 1 + ((i * 2) % 7)}))
	}
	for i := 1; i <= 7; i++ {
		x := &mt{y: i}
		if ty.Lookup(x) == nil {
			t.Error("!y")
		}
	}
}

func TestMultiCoverage(t *testing.T) {
	cover := mtx{}
	cover.insert(&mt{x: 1})
	cover.insert(&mt{x: 3})
	cover.insert(&mt{x: 2})
	cover.delete(&mt{x: 17})
}
