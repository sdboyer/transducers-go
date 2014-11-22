package transduce

type reducerBase struct {
	next ReduceStep
}

func (r reducerBase) Complete(accum interface{}) interface{} {
	// Pure functions inherently can't have any completion work, so flow through
	return r.next.Complete(accum)
}
