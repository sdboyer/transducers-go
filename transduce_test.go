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

func TestTransduceChunkFlatten(t *testing.T) {
	xform := []Transducer{Chunk(3), Mapcat(Flatten)}
	result := Seq(Range(6), make([]int, 0), xform...)
	// TODO crappy test b/c the steps are logical inversions - need to improv on Seq for better test

	intSliceEquals(([]int{0, 1, 2, 3, 4, 5}), result, t)
}

func TestTransduceChunkChunkByFlatten(t *testing.T) {
	chunker := func(value interface{}) bool {
		return sum(value.(ValueStream)) > 7
	}
	xform := []Transducer{Chunk(3), ChunkBy(chunker), Mapcat(Flatten)}
	result := Seq(Range(19), make([]int, 0), xform...)

	intSliceEquals(t_range(19), result, t)
}

func TestTransduceSample(t *testing.T) {
	result := Seq(Range(12), make([]int, 0), RandomSample(1))
	intSliceEquals(t_range(12), result, t)

	result2 := Seq(Range(12), make([]int, 0), RandomSample(0))
	if len(result2) != 0 {
		t.Error("Random sampling with 0 ρ should filter out all results")
	}
}

func TestTakeNth(t *testing.T) {
	result := Seq(Range(21), make([]int, 0), TakeNth(7))

	intSliceEquals([]int{6, 13, 20}, result, t)
}

func TestKeep(t *testing.T) {
	v := []interface{}{0, nil, 1, 2, nil, false}

	// include this type converter to make the bool into an int, or seq will have
	// a type panic at the end. Just to prove that Keep retains false vals.
	mapf := func(val interface{}) interface{} {
		if _, ok := val.(bool); ok {
			return 15
		}
		return val
	}

	keepf := func(val interface{}) interface{} {
		return val
	}

	result := Seq(MakeReduce(v), make([]int, 0), Keep(keepf), Map(mapf))

	intSliceEquals([]int{0, 1, 2, 15}, result, t)
}

func TestReplace(t *testing.T) {
	tostrings := map[interface{}]interface{}{
		2:  "two",
		6:  "six",
		18: "eighteen",
	}

	toints := map[interface{}]interface{}{
		"two":      55,
		"six":      35,
		"eighteen": 41,
	}

	result := Seq(Range(19), make([]int, 0), Replace(tostrings), Replace(toints))

	intSliceEquals([]int{0, 1, 55, 3, 4, 5, 35, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 41}, result, t)
}
