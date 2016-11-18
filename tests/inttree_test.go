package trees

import "testing"

//go:generate go run ../cmd/avlgen/main.go .

type iV struct {
	v     int
	tlink tLink `avlgen:"iT"`
}

func (a *iV) cmp(b *iV) (bool, bool) {
	return a.v == b.v, a.v < b.v
}

func TestIntsBasic(t *testing.T) {
	tr := iT{}

	for i := 0; i < 600; i++ {
		tr.insert(&iV{v: i})
	}
	for i := 0; i < 600; i++ {
		v := tr.lookup(&iV{v: i})
		if v.v != i {
			t.Errorf("%v != %v\n", i, v.v)
		}
	}
}

type iKV struct {
	k, v int
	tl   tl `avlgen:"ikvt,cmp:cmpiv,cmpval:cmpk(int),debug"`
}

func (a *iKV) cmpiv(b *iKV) (bool, bool) {
	return a.k == b.k, a.k < b.k
}

func (a *iKV) cmpk(b int) (bool, bool) {
	return a.k == b, a.k < b
}

func TestIntsLookupVal(t *testing.T) {
	tr := ikvt{}

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
