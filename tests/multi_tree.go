package trees

import "testing"

type mt struct {
	x, y int
	mtlx mtlx `avlgen:"mtx,cmp:cmpx,iter"`
	mtly mtly `avlgen:"mty,cmp:cmpy,iter"`
}

func (a *mt) cmpx(b *mt) (bool, bool) {
	return a.x == b.x, a.x < b.x
}

func (a *mt) cmpy(b *mt) (bool, bool) {
	return a.y == b.y, a.y < b.y
}

func TestMultiTree(t *testing.T) {
	const sz = 100
	tx := mtx{}
	ty := mty{}
	for i := 0; i < sz; i++ {
		m := &mt{x: i, y: sz - i}
		tx.insert(m)
		ty.insert(m)
	}
	ix := tx.iter(nil, nil, true, true)
	iy := ty.iter(ty.last(), ty.first(), true, true)
	for ix.next() && iy.next() {
		x := ix.value()
		y := iy.value()
		if x.x != y.y {
			t.Errorf("%v != %v", x, y)
		}
	}
}
