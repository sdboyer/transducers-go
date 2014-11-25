package transduce

import (
	"fmt"
	"testing"
)

var ints = []int{1, 2, 3, 4, 5}
var evens = []int{2, 4}

func printf(s string, ph ...interface{}) {
	fmt.Printf(s, ph...)
}

func dt(t []Transducer) []Transducer {
	return AttachLoggers(printf, t...)
}

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

func toi(i ...interface{}) []interface{} {
	return i
}

func streamEquals(expected []interface{}, s ValueStream, t *testing.T) {
	for k, v := range expected {
		val, done := s()
		if done {
			t.Errorf("Stream terminated before end of slice reached")
		}
		if v != val {
			t.Errorf("Error on index %v: expected %v got %v", k, v, val)
		}
	}

	_, done := s()
	if !done {
		t.Errorf("Exhausted slice, but stream had more values")
	}
}

func TestTransduceMF(t *testing.T) {
	mf := Transduce(ToStream(ints), make([]int, 0), Map(inc), Filter(even))
	fm := Transduce(ToStream(ints), make([]int, 0), Filter(even), Map(inc))

	intSliceEquals([]int{2, 4, 6}, mf, t)
	intSliceEquals([]int{3, 5}, fm, t)
}

func TestTransduceMapFilterMapcat(t *testing.T) {
	xform := []Transducer{Filter(even), Map(inc), Mapcat(Range)}
	result := Transduce(ToStream(ints), make([]int, 0), dt(xform)...)

	intSliceEquals([]int{0, 1, 2, 0, 1, 2, 3, 4}, result, t)
}

func TestTransduceMapFilterMapcatDedupe(t *testing.T) {
	xform := []Transducer{Filter(even), Map(inc), Mapcat(Range), Dedupe()}

	result := Transduce(ToStream(ints), make([]int, 0), dt(xform)...)
	intSliceEquals([]int{0, 1, 2, 3, 4}, result, t)

	// Dedupe is stateful. Do it twice to demonstrate that's handled
	result2 := Transduce(ToStream(ints), make([]int, 0), dt(xform)...)
	intSliceEquals([]int{0, 1, 2, 3, 4}, result2, t)
}

func TestTransduceChunkFlatten(t *testing.T) {
	xform := []Transducer{Chunk(3), Mapcat(Flatten)}
	result := Transduce(Range(6), make([]int, 0), dt(xform)...)
	// TODO crappy test b/c the steps are logical inversions - need to improv on Seq for better test

	intSliceEquals(([]int{0, 1, 2, 3, 4, 5}), result, t)
}

func TestTransduceChunkChunkByFlatten(t *testing.T) {
	chunker := func(value interface{}) interface{} {
		return sum(value.(ValueStream)) > 7
	}
	xform := []Transducer{Chunk(3), ChunkBy(chunker), Mapcat(Flatten)}
	result := Transduce(Range(19), make([]int, 0), dt(xform)...)

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

	xform := []Transducer{ChunkBy(chunker), Map(Sum)}
	result := Transduce(Range(10), make([]int, 0), dt(xform)...)
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

	xform := []Transducer{Keep(keepf), Map(mapf)}

	result := Transduce(ToStream(v), make([]int, 0), dt(xform)...)

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

	xform := []Transducer{Replace(tostrings), Replace(toints)}
	result := Transduce(Range(19), make([]int, 0), dt(xform)...)

	intSliceEquals([]int{0, 1, 55, 3, 4, 5, 35, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 41}, result, t)
}

func TestMapChunkTakeFlatten(t *testing.T) {
	xform := []Transducer{Map(inc), Chunk(2), Take(2), Mapcat(Flatten)}
	result := Transduce(Range(6), make([]int, 0), dt(xform)...)
	intSliceEquals([]int{1, 2, 3, 4}, result, t)

	result2 := Transduce(Range(6), make([]int, 0), dt(xform)...)
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
	result := Transduce(Range(50), make([]int, 0), dt(td)...)

	intSliceEquals([]int{5, 6, 7, 8, 9}, result, t)
}

func TestRemove(t *testing.T) {
	result := Transduce(Range(8), make([]int, 0), Remove(even))
	intSliceEquals([]int{1, 3, 5, 7}, result, t)
}

func TestFlattenValueStream(t *testing.T) {
	stream := flattenValueStream(ValueSlice{
		ToStream([]int{0, 1}),
		ValueSlice{
			ToStream([]int{2, 3}),
			ToStream([]int{4, 5, 6}),
		}.AsStream(),
		ToStream([]int{7, 8}),
	}.AsStream())

	var flattened []int
	stream.Each(func(v interface{}) {
		flattened = append(flattened, v.(int))
	})

	intSliceEquals(t_range(9), flattened, t)
}

func TestStreamDup(t *testing.T) {
	stream := Range(3)
	dupd := (&stream).Dup()

	var res1, res2 []int
	stream.Each(func(value interface{}) {
		res1 = append(res1, value.(int))
	})

	dupd.Each(func(value interface{}) {
		res2 = append(res2, value.(int))
	})

	intSliceEquals([]int{0, 1, 2}, res1, t)
	intSliceEquals([]int{0, 1, 2}, res2, t)
}

// Eduction tests cover 19 variations - 5 base (1 input: {1 out, 0 out, 0..1 out, 1..n out, 0..n out}),
// permuted with a transducer that flushes on complete, and an early terminator. only 19 instead of
// 20 because the 1:0 case cannot possibly have anything that
func TestEduction(t *testing.T) {
	var xf []Transducer
	var res ValueStream

	// simple 1:1
	xf = []Transducer{Map(inc)}
	res = Eduction(Range(5), dt(xf)...)
	streamEquals(toi(1, 2, 3, 4, 5), res, t)

	// contractor
	xf = append(xf, Filter(even))
	res = Eduction(Range(5), dt(xf)...)
	streamEquals(toi(2, 4), res, t)

	// expander
	xf = append(xf, Mapcat(Range))
	res = Eduction(Range(5), dt(xf)...)
	streamEquals(toi(0, 1, 0, 1, 2, 3), res, t)

	// terminator
	xf = append(xf, Take(5))
	res = Eduction(Range(5), dt(xf)...)
	streamEquals(toi(0, 1, 0, 1, 2), res, t)

	// stateful/flusher
	xf = append(xf, Chunk(2))
	res = Eduction(Range(5), dt(xf)...)
	// streamEquals(toi(toi(0, 1), toi(0, 1), 2), res, t) need recursive stream flattening...
}
