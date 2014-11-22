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

func TestTransduceMF(t *testing.T) {
	mf := Transduce(MakeReduce(ints), make([]int, 0), Map(inc), Filter(even))
	fm := Transduce(MakeReduce(ints), make([]int, 0), Filter(even), Map(inc))

	intSliceEquals([]int{2, 4, 6}, mf, t)
	intSliceEquals([]int{3, 5}, fm, t)
}

func TestTransduceMapFilterMapcat(t *testing.T) {
	result := Transduce(MakeReduce(ints), make([]int, 0), Filter(even), Map(inc), Mapcat(Range))

	intSliceEquals([]int{0, 1, 2, 0, 1, 2, 3, 4}, result, t)
}

func TestTransduceMapFilterMapcatDedupe(t *testing.T) {
	xform := []Transducer{Filter(even), Map(inc), Mapcat(Range), Dedupe()}

	result := Transduce(MakeReduce(ints), make([]int, 0), xform...)
	intSliceEquals([]int{0, 1, 2, 3, 4}, result, t)

	// Dedupe is stateful. Do it twice to demonstrate that's handled
	result2 := Transduce(MakeReduce(ints), make([]int, 0), xform...)
	intSliceEquals([]int{0, 1, 2, 3, 4}, result2, t)
}

func TestTransduceChunkFlatten(t *testing.T) {
	xform := []Transducer{Chunk(3), Mapcat(Flatten)}
	result := Transduce(Range(6), make([]int, 0), xform...)
	// TODO crappy test b/c the steps are logical inversions - need to improv on Seq for better test

	intSliceEquals(([]int{0, 1, 2, 3, 4, 5}), result, t)
}

func TestTransduceChunkChunkByFlatten(t *testing.T) {
	chunker := func(value interface{}) interface{} {
		return sum(value.(ValueStream)) > 7
	}
	xform := []Transducer{Chunk(3), ChunkBy(chunker), Mapcat(Flatten)}
	result := Transduce(Range(19), make([]int, 0), xform...)

	intSliceEquals(t_range(19), result, t)
}

func TestMultiChunkBy(t *testing.T) {
	chunker := func(value interface{}) interface{} {
		switch v := value.(int); {
		case v < 4:
			return "boo"
		case v < 7:
			return false
		default:
			return "boo"
		}
	}

	result := Transduce(Range(10), make([]int, 0), ChunkBy(chunker), Map(Sum))
	intSliceEquals([]int{6, 15, 24}, result, t)
}

func TestTransduceSample(t *testing.T) {
	result := Transduce(Range(12), make([]int, 0), RandomSample(1))
	intSliceEquals(t_range(12), result, t)

	result2 := Transduce(Range(12), make([]int, 0), RandomSample(0))
	if len(result2) != 0 {
		t.Error("Random sampling with 0 ρ should filter out all results")
	}
}

func TestTakeNth(t *testing.T) {
	result := Transduce(Range(21), make([]int, 0), TakeNth(7))

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

	result := Transduce(MakeReduce(v), make([]int, 0), Keep(keepf), Map(mapf))

	intSliceEquals([]int{0, 1, 2, 15}, result, t)
}

func TestKeepIndexed(t *testing.T) {
	keepf := func(index int, value interface{}) interface{} {
		if !even(index) {
			return nil
		}
		return index * value.(int)
	}

	td := KeepIndexed(keepf)

	result := Transduce(Range(7), make([]int, 0), td)
	intSliceEquals([]int{0, 4, 16, 36}, result, t)

	result2 := Transduce(Range(7), make([]int, 0), td)
	intSliceEquals([]int{0, 4, 16, 36}, result2, t)
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

	result := Transduce(Range(19), make([]int, 0), Replace(tostrings), Replace(toints))

	intSliceEquals([]int{0, 1, 55, 3, 4, 5, 35, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 41}, result, t)
}

func TestMapChunkTakeFlatten(t *testing.T) {
	td := []Transducer{Map(inc), Chunk(2), Take(2), Mapcat(Flatten)}
	result := Transduce(Range(6), make([]int, 0), td...)
	intSliceEquals([]int{1, 2, 3, 4}, result, t)

	result2 := Transduce(Range(6), make([]int, 0), td...)
	intSliceEquals([]int{1, 2, 3, 4}, result2, t)
}

func TestTakeWhile(t *testing.T) {
	filter := func(value interface{}) bool {
		return value.(int) < 4
	}
	result := Transduce(Range(6), make([]int, 0), TakeWhile(filter))
	intSliceEquals([]int{0, 1, 2, 3}, result, t)
}

func TestDropDropDropWhileTake(t *testing.T) {
	dw := func(value interface{}) bool {
		return value.(int) < 5
	}
	td := []Transducer{Drop(1), Drop(1), DropWhile(dw), Take(5)}
	result := Transduce(Range(50), make([]int, 0), td...)

	intSliceEquals([]int{5, 6, 7, 8, 9}, result, t)
}

func TestRemove(t *testing.T) {
	result := Transduce(Range(8), make([]int, 0), Remove(even))
	intSliceEquals([]int{1, 3, 5, 7}, result, t)
}

func TestFlattenValueStream(t *testing.T) {
	stream := flattenValueStream(ValueSlice{
		MakeReduce([]int{0, 1}),
		ValueSlice{
			MakeReduce([]int{2, 3}),
			MakeReduce([]int{4, 5, 6}),
		}.AsStream(),
		MakeReduce([]int{7, 8}),
	}.AsStream())

	var flattened []int
	stream.Each(func(v interface{}) {
		flattened = append(flattened, v.(int))
	})

	intSliceEquals(t_range(9), flattened, t)
}
