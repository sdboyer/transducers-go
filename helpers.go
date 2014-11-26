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

type bareReducer struct {
	R func(accum interface{}, value interface{}) (interface{}, bool)
	C func(accum interface{}) interface{}
	I func() interface{}
}

func (r bareReducer) Reduce(accum interface{}, value interface{}) (interface{}, bool) {
	return r.R(accum, value)
}

func (r bareReducer) Complete(accum interface{}) interface{} {
	return r.C(accum)
}

func (r bareReducer) Init() interface{} {
	return r.I()
}

func BareReducer() bareReducer {
	return bareReducer{
		R: func(accum interface{}, value interface{}) (interface{}, bool) {
			return accum, false
		},
		C: func(accum interface{}) interface{} {
			return accum
		},
		I: func() interface{} {
			return make([]interface{}, 0)
		},
	}
}
