package transduce

import "testing"

var ints = []int{1, 2, 3, 4, 5}
var evens = []int{2, 4}

func intSliceEquals(a []int, b []int, t *testing.T) {
	if len(a) != len(b) {
		t.Error("Slices not even length")
	}

	for k, v := range a {
		if b[k] != v {
			t.Error("Error on index", k, ": expected", v, "got", b[k])
		}
	}

}

func TestDirectMap(t *testing.T) {
	res := DirectMap(inc, ints)

	for k, v := range ints {
		if res[k] != v+1 {
			t.Error("Error on index", k, ": expected", v+1, "got", res[k])
		}
	}
}

func TestDirectFilter(t *testing.T) {
	res := DirectFilter(even, ints)

	if len(res) != len(evens) {
		t.Error("Wrong length of result on direct filter")
	}

	for k, v := range evens {
		if res[k] != v {
			t.Error("Error on index", k, ": expected", v, "got", res[k])
		}
	}
}

func TestMapThruReduce(t *testing.T) {
	res := MapThruReduce(inc, ints)

	for k, v := range ints {
		if res[k] != v+1 {
			t.Error("Error on index", k, ": expected", v+1, "got", res[k])
		}
	}
}

func TestFilterThruReduce(t *testing.T) {
	res := FilterThruReduce(even, ints)

	if len(res) != len(evens) {
		t.Error("Wrong length of result on direct filter")
	}

	for k, v := range evens {
		if res[k] != v {
			t.Error("Error on index", k, ": expected", v, "got", res[k])
		}
	}
}

func TestMapReducer(t *testing.T) {
	res := Reduce(ints, MapFunc(inc), make([]int, 0)).([]int)

	for k, v := range ints {
		if res[k] != v+1 {
			t.Error("Error on index", k, ": expected", v+1, "got", res[k])
		}
	}
}

func TestFilterReducer(t *testing.T) {
	res := Reduce(ints, FilterFunc(even), make([]int, 0)).([]int)

	if len(res) != len(evens) {
		t.Error("Wrong length of result on direct filter")
	}

	for k, v := range evens {
		if res[k] != v {
			t.Error("Error on index", k, ": expected", v, "got", res[k])
		}
	}
}

func TestTransduceMF(t *testing.T) {
	mf := Seq(MakeReduce(ints), make([]int, 0), Map(inc), Filter(even))
	fm := Seq(MakeReduce(ints), make([]int, 0), Filter(even), Map(inc))

	intSliceEquals([]int{2, 4, 6}, mf, t)
	intSliceEquals([]int{3, 5}, fm, t)
}

func TestTransduceMapFilterMapcat(t *testing.T) {
	result := Seq(MakeReduce(ints), make([]int, 0), Filter(even), Map(inc), Mapcat(Range))

	intSliceEquals([]int{0, 1, 2, 0, 1, 2, 3, 4}, result, t)
}

func TestTransduceMapFilterMapcatDedupe(t *testing.T) {
	xform := []Transducer{Filter(even), Map(inc), Mapcat(Range), Dedupe()}

	result := Seq(MakeReduce(ints), make([]int, 0), xform...)
	intSliceEquals([]int{0, 1, 2, 3, 4}, result, t)

	// Dedupe is stateful. Do it twice to demonstrate that's handled
	result2 := Seq(MakeReduce(ints), make([]int, 0), xform...)
	intSliceEquals([]int{0, 1, 2, 3, 4}, result2, t)
}
