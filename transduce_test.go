package transduce

import (
	"fmt"
	"testing"
)

var fml = fmt.Println

var ints = []int{1, 2, 3, 4, 5}
var evens = []int{2, 4}

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
