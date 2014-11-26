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
