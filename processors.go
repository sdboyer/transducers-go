package transduce

// Transduce performs a non-lazy traversal/reduction over the provided value stream.
func Transduce(vs ValueStream, init []int, tlist ...Transducer) []int {
	// Final reducing func - append to slice
	t := CreatePipeline(Append(), tlist...)

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
