package transduce

// Transduce performs a non-lazy traversal/reduction over the provided value stream.
func Transduce(vs ValueStream, init []int, tlist ...Transducer) []int {
	// Final reducing func - append to slice
	// TODO really awkward patching this together like this - refactor it out smartly
	var t ReduceStep = pureReducer{inner: Append(Identity), complete: func(accum interface{}) interface{} {
		return accum
	}}

	// Walk backwards through transducer list to assemble in correct order.
	// Clojure folk refer to this as applying transducers to this job.
	for i := len(tlist) - 1; i >= 0; i-- {
		t = tlist[i].Transduce(t)
	}

	var ret interface{} = init
	var terminate bool

	for v, done := vs(); !done; v, done = vs() {
		fml("TRANSDUCE: Main loop:", v)
		ret, terminate = t.Reduce(ret, v)
		if terminate {
			break
		}
	}

	ret = t.Complete(ret)

	return ret.([]int)
}
