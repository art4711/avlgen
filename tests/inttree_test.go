package trees

import "testing"

func TestIntsBasic(t *testing.T) {
	tr := iT{}

	for i := 0; i < 600; i++ {
		tr.Insert(&iV{v: i})
	}
	for i := 0; i < 600; i++ {
		v := tr.Lookup(&iV{v: i})
		if v.v != i {
			t.Errorf("%v != %v\n", i, v.v)
		}
	}
}
