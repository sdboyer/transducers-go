package transduce

import "fmt"

const dbg = false

func fml(v ...interface{}) {
	if dbg {
		fmt.Println(v)
	}
}

type reducerBase struct {
	next ReduceStep
}

func (r reducerBase) Complete(accum interface{}) interface{} {
	// Pure functions inherently can't have any completion work, so flow through
	return r.next.Complete(accum)
}

func (r reducerBase) Init() interface{} {
	return r.next.Init()
}

type reduceStepHelper struct {
	R func(accum interface{}, value interface{}) (interface{}, bool)
	C func(accum interface{}) interface{}
	I func() interface{}
}

func (r reduceStepHelper) Reduce(accum interface{}, value interface{}) (interface{}, bool) {
	return r.R(accum, value)
}

func (r reduceStepHelper) Complete(accum interface{}) interface{} {
	return r.C(accum)
}

func (r reduceStepHelper) Init() interface{} {
	return r.I()
}

// Creates a helper struct for defining a ReduceStep on the fly.
//
// This is mostly useful for creating a bottom reducer with minimal fanfare.
//
// This returns an instance of bareReducer, which is a struct containing three
// function pointers, one for each of the three methods of ReduceStep - R for
// Reduce, C for Complete, I for Init. The struct implements ReduceStep
// by simply passing method calls along to the contained function pointers.
//
// This makes it easier to create ReduceSteps on the fly. The first argument is a
// reducer - if you pass nil, it'll create a no-op reducer for you. If you want to
// overwrite the other two, do it on the returned struct.
func CreateStep(r Reducer) reduceStepHelper {
	if r == nil {
		r = func(accum interface{}, value interface{}) (interface{}, bool) {
			return accum, false
		}
	}
	return reduceStepHelper{
		R: r,
		C: func(accum interface{}) interface{} {
			return accum
		},
		I: func() interface{} {
			return make([]interface{}, 0)
		},
	}
}
