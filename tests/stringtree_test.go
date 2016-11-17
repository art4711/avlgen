package trees

import (
	"strconv"
	"testing"
)

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

func BenchmarkSS1Mlinear(b *testing.B) {
	const sz = 1000000

	tr := sst{}
	b.Run("insert", func(b *testing.B) {
		for bn := 0; bn < b.N; bn++ {
			b.ReportAllocs()
			tr = sst{} // drop the whole tree each time
			for i := 0; i < sz; i++ {
				tr.insert(&ss{k: strconv.Itoa(i), v: strconv.Itoa(i * 7)})
			}
			b.SetBytes(sz)
		}
	})
	// we have one tree left
	b.Run("lookupVal", func(b *testing.B) {
		for bn := 0; bn < b.N; bn++ {
			b.ReportAllocs()
			for i := 0; i < sz; i++ {
				kv := tr.lookupVal(strconv.Itoa(i))
				//kv := tr.lookup(&ss{k: strconv.Itoa(i)})
				if kv == nil {
					b.Fatalf("no value at %d", i)
				}
				v, err := strconv.Atoi(kv.v)
				if err != nil || v != i*7 {
					b.Fatalf("bad value %v, %v\n", kv, err)
				}
			}
			b.SetBytes(sz)
		}
	})
}

func BenchmarkMapSS1Mlinear(b *testing.B) {
	const sz = 1000000
	tr := map[string]string{}
	b.Run("insert", func(b *testing.B) {
		b.ReportAllocs()
		for bn := 0; bn < b.N; bn++ {
			tr = map[string]string{} // drop the whole map each time
			for i := 0; i < sz; i++ {
				tr[strconv.Itoa(i)] = strconv.Itoa(i * 7)
			}
			b.SetBytes(sz)
		}
	})
	// we have one tree left
	b.Run("lookup", func(b *testing.B) {
		b.ReportAllocs()
		for bn := 0; bn < b.N; bn++ {
			for i := 0; i < sz; i++ {
				mv := tr[strconv.Itoa(i)]
				if mv == "" {
					b.Fatalf("no value at %d", i)
				}
				v, err := strconv.Atoi(mv)
				if err != nil || v != i*7 {
					b.Fatalf("bad value %v, %v\n", mv, err)
				}
			}
			b.SetBytes(sz)
		}
	})
}
