package transduce

import (
	"fmt"
	"testing"
)

var fml = fmt.Println

var ints = []int{1, 2, 3, 4, 5}
var odds = []int{1, 3, 5}

func TestDirectMap(t *testing.T) {
	res := DirectMap(inc, ints)

	for k, v := range ints {
		if res[k] != v+1 {
			t.Error("Error on index", k, ": expected", v+1, "got", res[k])
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
