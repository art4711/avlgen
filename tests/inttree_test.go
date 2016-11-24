package trees

import (
	"math/rand"
	"testing"
)

//go:generate go run ../cmd/avlgen/main.go .

type iKV struct {
	k, v int
	tl   tl `avlgen:"ikvt,cmp:cmpiv,cmpval:cmpk(int),debug,no:last"`
}

func (a *iKV) cmpiv(b *iKV) (bool, bool) {
	return a.k == b.k, a.k < b.k
}

func (a *iKV) cmpk(b int) (bool, bool) {
	return a.k == b, a.k < b
}

func TestIntsLookupVal(t *testing.T) {
	tr := &ikvt{}

	for i := 0; i < 600; i++ {
		tr.insert(&iKV{k: i, v: i * 2})
	}
	for i := 0; i < 600; i++ {
		v := tr.lookup(&iKV{k: i})
		v2 := tr.lookupVal(i)
		if v.v != i*2 || v2 != v {
			t.Errorf("%v != %v/%v\n", i*2, v, v2)
		}
	}
	tr.foreach(nil, nil, func(n *iKV) {
		if err := tr.check(n); err != nil {
			t.Error(err)
		}
	})
}

func TestIntsDel(t *testing.T) {
	tr := ikvt{}

	for i := 0; i < 100000; i++ {
		tr.insert(&iKV{k: i, v: i})
	}
	for i := 99998; i >= 0; i -= 2 {
		n := tr.lookupVal(i)
		tr.delete(n)
	}
	for i := 99999; i >= 0; i -= 6 {
		n := tr.lookupVal(i)
		tr.delete(n)
	}
	c := 0
	tr.foreach(nil, nil, func(n *iKV) {
		if n.v%2 == 0 || n.v%3 == 0 {
			t.Errorf("%d not deleted", n.v)
		}
		if err := tr.check(n); err != nil {
			t.Error(err)
		}
		c++
	})
	if c != 33333 {
		t.Errorf("wrong number left: %d", c)
	}
}

func TestIntsDelVal(t *testing.T) {
	tr := ikvt{}

	for i := 0; i < 100000; i++ {
		tr.insert(&iKV{k: i, v: i})
	}
	for i := 99998; i >= 0; i -= 2 {
		tr.deleteVal(i)
	}
	for i := 99999; i >= 0; i -= 6 {
		tr.deleteVal(i)
	}
	c := 0
	tr.foreach(nil, nil, func(n *iKV) {
		if n.v%2 == 0 || n.v%3 == 0 {
			t.Errorf("%d not deleted", n.v)
		}
		if err := tr.check(n); err != nil {
			t.Error(err)
		}
		c++
	})
	if c != 33333 {
		t.Errorf("wrong number left: %d", c)
	}
}

func TestIntsSearchGEQ(t *testing.T) {
	tr := ikvt{}

	for i := 0; i < 100000; i += 5 {
		tr.insert(&iKV{k: i, v: i})
	}
	// There is no 100k
	for i := 0; i < 99996; i++ {
		n := tr.searchValGEQ(i)

		expect := ((i + 4) / 5) * 5
		if n == nil {
			t.Fatalf("nil n %d", i)
		}
		if n.v != expect {
			t.Errorf("%d %d != %d", i, expect, n.v)
		}
	}
}

func TestIntsSearchLEQ(t *testing.T) {
	tr := ikvt{}

	for i := 0; i < 100000; i += 5 {
		tr.insert(&iKV{k: i, v: i})
	}
	for i := 0; i < 100000; i++ {
		n := tr.searchValLEQ(i)

		expect := (i / 5) * 5
		if n == nil {
			t.Fatalf("nil n %d", i)
		}
		if n.v != expect {
			t.Errorf("%d %d != %d", i, expect, n.v)
		}
	}
}

func TestIntsBalance(t *testing.T) {
	tr := ikvt{}

	tr.insert(&iKV{k: 5})
	tr.insert(&iKV{k: 2})
	tr.insert(&iKV{k: 6})
	tr.insert(&iKV{k: 3})
	tr.insert(&iKV{k: 1})
	tr.insert(&iKV{k: 7})
	tr.insert(&iKV{k: 4})
	tr.deleteVal(5)
	tr.foreach(nil, nil, func(n *iKV) {
		if err := tr.check(n); err != nil {
			t.Errorf("%v", err)
		}
	})
}

func TestIntsRandom(t *testing.T) {
	const sz = 10000
	tr := &ikvt{}
	ins, del := 0, 0
	for i := 0; i < sz; i++ {
		r := rand.Intn(sz)
		if r > i {
			if tr.lookupVal(r) == nil {
				tr.insert(&iKV{k: r})
				ins++
			}
		} else {
			tr.deleteVal(r)
			del++
		}
		tr.foreach(nil, nil, func(n *iKV) {
			if err := tr.check(n); err != nil {
				t.Errorf("%v || %d %d", err, i, r)
			}
		})
	}
}

type iv struct {
	v   int
	ivl ivl `avlgen:"ivt,cmpval:cmpk(int),iter,export,no:lookup,no:delete,no:deleteVal,no:lookupVal"`
}

func (a *iv) cmp(b *iv) (bool, bool) {
	return a.v == b.v, a.v < b.v
}

func (a *iv) cmpk(b int) (bool, bool) {
	return a.v == b, a.v < b
}

func tIntIter(t *testing.T, first, last int, it *ivtIter) {
	inc := 1
	if first > last && last != -1 {
		inc = -1
	}
	i := first
	for it.next() {
		v := it.value().v
		if i != v {
			t.Errorf("%d != %d", v, i)
		}
		i += inc
	}
	if i-inc != last {
		t.Errorf("wrong last %d != %d (%d)", i-inc, last, inc)
	}
}

func TestIntsIter(t *testing.T) {
	tr := ivt{}

	for i := 0; i < 1000; i++ {
		tr.Insert(&iv{v: i})
	}
	t.Run("all", func(t *testing.T) {
		tIntIter(t, 0, 999, tr.IterVal(0, 0, true, true, true, true))
	})
	t.Run("all-explicit", func(t *testing.T) {
		tIntIter(t, 0, 999, tr.IterVal(0, 999, false, false, true, true))
	})
	t.Run("just-one-start", func(t *testing.T) {
		tIntIter(t, 0, 0, tr.IterVal(0, 0, false, false, true, true))
	})
	t.Run("just-one-start-no-ince", func(t *testing.T) {
		tIntIter(t, 0, -1, tr.IterVal(0, 0, false, false, true, false))
	})
	t.Run("just-one-start-no-incs", func(t *testing.T) {
		tIntIter(t, 0, -1, tr.IterVal(0, 0, false, false, false, true))
	})
	mid := tr.n.v
	t.Run("start-to-mid", func(t *testing.T) {
		tIntIter(t, 0, mid, tr.IterVal(0, mid, true, false, true, true))
	})
	t.Run("mid-to-end", func(t *testing.T) {
		tIntIter(t, mid, 999, tr.IterVal(mid, 0, false, true, true, true))
	})
	t.Run("just-mid", func(t *testing.T) {
		tIntIter(t, mid, mid, tr.IterVal(mid-1, mid+1, false, false, false, false))
	})
	t.Run("mid-and-neighbours", func(t *testing.T) {
		tIntIter(t, mid-1, mid+1, tr.IterVal(mid-1, mid+1, false, false, true, true))
	})
	t.Run("arbitrary-range", func(t *testing.T) {
		tIntIter(t, 17, 41, tr.IterVal(17, 42, false, false, true, false))
	})
	t.Run("other-constructor", func(t *testing.T) {
		tIntIter(t, 0, 999, tr.Iter(nil, nil, true, true))
	})
	t.Run("other-constructor", func(t *testing.T) {
		tIntIter(t, 0, 999, tr.Iter(tr.First(), tr.Last(), true, true))
	})
	t.Run("reverse", func(t *testing.T) {
		tIntIter(t, 999, 0, tr.Iter(tr.Last(), tr.First(), true, true))
	})
}

func BenchmarkII1Mlinear(b *testing.B) {
	const sz = 1000000

	tr := ikvt{}
	b.Run("insert", func(b *testing.B) {
		for bn := 0; bn < b.N; bn++ {
			b.ReportAllocs()
			tr = ikvt{} // drop the whole tree each time
			for i := 0; i < sz; i++ {
				tr.insert(&iKV{k: i, v: i * 3})
			}
			b.SetBytes(sz)
		}
	})
	// we have one tree left
	b.Run("lookup", func(b *testing.B) {
		for bn := 0; bn < b.N; bn++ {
			b.ReportAllocs()
			for i := 0; i < sz; i++ {
				kv := tr.lookup(&iKV{k: i})
				if kv.v != i*3 {
					b.Fatal("bad value %v\n", kv)
				}
			}
			b.SetBytes(sz)
		}
	})
	b.Run("lookupVal", func(b *testing.B) {
		for bn := 0; bn < b.N; bn++ {
			b.ReportAllocs()
			for i := 0; i < sz; i++ {
				kv := tr.lookupVal(i)
				if kv.v != i*3 {
					b.Fatal("bad value %v\n", kv)
				}
			}
			b.SetBytes(sz)
		}
	})
	b.Run("delete", func(b *testing.B) {
		for bn := 0; bn < b.N; bn++ {
			b.ReportAllocs()
			for i := 0; i < sz; i++ {
				tr.delete(tr.lookupVal(i))
			}
			b.SetBytes(sz)
		}
	})
	b.Run("deleteVal", func(b *testing.B) {
		for bn := 0; bn < b.N; bn++ {
			b.ReportAllocs()
			for i := 0; i < sz; i++ {
				tr.deleteVal(i)
			}
			b.SetBytes(sz)
		}
	})
}

func BenchmarkMapII1Mlinear(b *testing.B) {
	const sz = 1000000
	tr := map[int]int{}
	b.Run("insert", func(b *testing.B) {
		b.ReportAllocs()
		for bn := 0; bn < b.N; bn++ {
			tr = map[int]int{} // drop the whole map each time
			for i := 0; i < sz; i++ {
				tr[i] = i * 3
			}
			b.SetBytes(sz)
		}
	})
	// we have one map left
	b.Run("lookup", func(b *testing.B) {
		b.ReportAllocs()
		for bn := 0; bn < b.N; bn++ {
			for i := 0; i < sz; i++ {
				v := tr[i]
				if v != i*3 {
					b.Fatal("bad value %v\n", v)
				}
			}
			b.SetBytes(sz)
		}
	})
	b.Run("delete", func(b *testing.B) {
		b.ReportAllocs()
		for bn := 0; bn < b.N; bn++ {
			for i := 0; i < sz; i++ {
				delete(tr, i)
			}
			b.SetBytes(sz)
		}
	})
}
