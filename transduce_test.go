package transduce

import (
	"fmt"
	"testing"
)

var fml = fmt.Println

var ints = []int{1, 2, 3, 4, 5}

func TestDirectMap(t *testing.T) {
	res := DirectMap(inc, ints)

	for k, v := range ints {
		if res[k] != v {
			t.Error("Error on index", k, ": expected", v+1, "got", res[k])
		}
	}
}
