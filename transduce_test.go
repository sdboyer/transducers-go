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

// tb == testbottom. simple appender
func tb() ReduceStep {
	b := BareReducer()
	b.R = func(accum interface{}, value interface{}) (interface{}, bool) {
		return append(accum.([]int), value.(int)), false
	}
	b.I = func() interface{} {
		return make([]int, 0)
	}

	return b
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
	mf := Transduce(ToStream(ints), tb(), Map(inc), Filter(even)).([]int)
	fm := Transduce(ToStream(ints), tb(), Filter(even), Map(inc)).([]int)

	intSliceEquals([]int{2, 4, 6}, mf, t)
	intSliceEquals([]int{3, 5}, fm, t)
}

func TestTransduceMapFilterMapcat(t *testing.T) {
	xform := []Transducer{Filter(even), Map(inc), Mapcat(Range)}
	result := Transduce(ToStream(ints), tb(), dt(xform)...).([]int)

	intSliceEquals([]int{0, 1, 2, 0, 1, 2, 3, 4}, result, t)
}

func TestTransduceMapFilterMapcatDedupe(t *testing.T) {
	xform := []Transducer{Filter(even), Map(inc), Mapcat(Range), Dedupe()}

	result := Transduce(ToStream(ints), tb(), dt(xform)...).([]int)
	intSliceEquals([]int{0, 1, 2, 3, 4}, result, t)

	// Dedupe is stateful. Do it twice to demonstrate that's handled
	result2 := Transduce(ToStream(ints), tb(), dt(xform)...).([]int)
	intSliceEquals([]int{0, 1, 2, 3, 4}, result2, t)
}

func TestTransduceChunkFlatten(t *testing.T) {
	xform := []Transducer{Chunk(3), Mapcat(Flatten)}
	result := Transduce(Range(6), tb(), dt(xform)...).([]int)
	// TODO crappy test b/c the steps are logical inversions - need to improv on Seq for better test

	intSliceEquals(([]int{0, 1, 2, 3, 4, 5}), result, t)
}

func TestTransduceChunkChunkByFlatten(t *testing.T) {
	chunker := func(value interface{}) interface{} {
		return sum(value.(ValueStream)) > 7
	}
	xform := []Transducer{Chunk(3), ChunkBy(chunker), Mapcat(Flatten)}
	result := Transduce(Range(19), tb(), dt(xform)...).([]int)

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
	result := Transduce(Range(10), tb(), dt(xform)...).([]int)
	intSliceEquals([]int{6, 15, 24}, result, t)
}

func TestTransduceSample(t *testing.T) {
	result := Transduce(Range(12), tb(), RandomSample(1)).([]int)
	intSliceEquals(t_range(12), result, t)

	result2 := Transduce(Range(12), tb(), RandomSample(0)).([]int)
	if len(result2) != 0 {
		t.Error("Random sampling with 0 ρ should filter out all results")
	}
}

func TestTakeNth(t *testing.T) {
	result := Transduce(Range(21), tb(), TakeNth(7)).([]int)

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

	result := Transduce(ToStream(v), tb(), dt(xform)...).([]int)

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

	result := Transduce(Range(7), tb(), td).([]int)
	intSliceEquals([]int{0, 4, 16, 36}, result, t)

	result2 := Transduce(Range(7), tb(), td).([]int)
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
	result := Transduce(Range(19), tb(), dt(xform)...).([]int)

	intSliceEquals([]int{0, 1, 55, 3, 4, 5, 35, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 41}, result, t)
}

func TestMapChunkTakeFlatten(t *testing.T) {
	xform := []Transducer{Map(inc), Chunk(2), Take(2), Mapcat(Flatten)}
	result := Transduce(Range(6), tb(), dt(xform)...).([]int)
	intSliceEquals([]int{1, 2, 3, 4}, result, t)

	result2 := Transduce(Range(6), tb(), dt(xform)...).([]int)
	intSliceEquals([]int{1, 2, 3, 4}, result2, t)
}

func TestTakeWhile(t *testing.T) {
	filter := func(value interface{}) bool {
		return value.(int) < 4
	}
	result := Transduce(Range(6), tb(), TakeWhile(filter)).([]int)
	intSliceEquals([]int{0, 1, 2, 3}, result, t)
}

func TestDropDropDropWhileTake(t *testing.T) {
	dw := func(value interface{}) bool {
		return value.(int) < 5
	}
	td := []Transducer{Drop(1), Drop(1), DropWhile(dw), Take(5)}
	result := Transduce(Range(50), tb(), dt(td)...).([]int)

	intSliceEquals([]int{5, 6, 7, 8, 9}, result, t)
}

func TestRemove(t *testing.T) {
	result := Transduce(Range(8), tb(), Remove(even)).([]int)
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

func TestEduction(t *testing.T) {
	var res ValueStream

	// simple 1:1
	xf1 := []Transducer{Map(inc)}
	res = Eduction(Range(5), dt(xf1)...)
	streamEquals(toi(1, 2, 3, 4, 5), res, t)

	// contractor
	xf2 := append(xf1, Filter(even))
	res = Eduction(Range(5), dt(xf2)...)
	streamEquals(toi(2, 4), res, t)

	// expander
	xf3 := append(xf2, Mapcat(Range))
	res = Eduction(Range(5), dt(xf3)...)
	streamEquals(toi(0, 1, 0, 1, 2, 3), res, t)

	// terminator
	xf4 := append(xf3, Take(5))
	res = Eduction(Range(5), dt(xf4)...)
	streamEquals(toi(0, 1, 0, 1, 2), res, t)

	// stateful/flusher
	xf5 := append(xf4, Chunk(2), Mapcat(Flatten)) // add flatten b/c no auto-recursive compare
	res = Eduction(Range(5), dt(xf5)...)
	streamEquals(toi(0, 1, 0, 1, 2), res, t)

	// feels like there are more permutations to check
}
