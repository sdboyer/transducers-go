package transduce

type pureReducer struct {
	inner    Reducer
	next     ReduceStep
	complete func(accum interface{}) (result interface{})
}

func (r pureReducer) Reduce(accum interface{}, value interface{}) (interface{}, bool) {
	return r.inner(accum, value)
}

func (r pureReducer) Complete(accum interface{}) interface{} {
	// Pure functions can't have any completion work to do, so just pass along
	// TODO ...uh, right? either way, this is just a convenience
	return r.complete(accum)
}

type pureReducer2 struct {
	next ReduceStep
}

func (r pureReducer2) Complete(accum interface{}) interface{} {
	// Pure functions inherently can't have any completion work, so flow through
	return r.next.Complete(accum)
}
