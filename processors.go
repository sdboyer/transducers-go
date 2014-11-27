package transduce

// Transduce performs a non-lazy traversal/reduction over the provided value stream.
func Transduce(coll interface{}, bottom ReduceStep, tlist ...Transducer) interface{} {
	// Final reducing func - append to slice
	t := CreatePipeline(bottom, tlist...)

	vs := ToStream(coll)
	var ret interface{} = t.Init()
	var terminate bool

	for v, done := vs(); !done; v, done = vs() {
		fml("TRANSDUCE: Main loop:", v)
		ret, terminate = t.Reduce(ret, v)
		if terminate {
			break
		}
	}

	ret = t.Complete(ret)

	return ret
}

// Applies the transducer stack to the provided collection, then encapsulates
// flow within a ValueStream, and returns the stream.
//
// Calling for a value on the returned ValueStream will send a value into
// the transduction pipe. Depending on the transducer components involved,
// a given pipeline could produce 0..N values for each input value.
// Nevertheless, Eduction ValueStreams still must return one value per call.
//
// In the case that sending an input results in 0 outputs reaching the
// bottom reducer, an Eduction will simply continue sending values to the input
// end until a value emerges, or the input stream is exhausted. In the latter
// case, the returned ValueStream will also indicate it is exhausted.
//
// If more than one value emerges for a single input, the eduction stream
// places the additional results into a FIFO queue. Subsequent calls to
// the stream will drain the queue before sending another value from the
// source stream into the pipeline.
//
// Note that using processor with transducers that have side effects is a
// particularly bad idea. It's also a bad idea to use it if your transduction
// stack has a high degree of fanout, as the queue can become quite large.
func Eduction(coll interface{}, tlist ...Transducer) ValueStream {
	var bottom Reducer = func(accum interface{}, value interface{}) (interface{}, bool) {
		return append(accum.([]interface{}), value), false
	}

	src := ToStream(coll)
	pipe := CreatePipeline(bottom, tlist...)
	var queue []interface{}
	var input interface{}
	var exhausted, terminate bool

	return func() (value interface{}, done bool) {
		// first, check the queue to see if it needs draining
		if len(queue) > 0 {
			// consume from queue - unshift slice
			// TODO is this leaky? not sure how to reason about that
			value, queue = queue[0], queue[1:]
			return value, false
		}

		if exhausted || terminate {
			// we'll get here if Complete enqueued more values after:
			//  a) we got an empty signal from the src stream, or
			//  b) if the pipeline returned the termination signal
			return nil, true
		}

		for {
			// queue is empty. feed the pipe till its not, or we exhaust src
			input, exhausted = src()
			if exhausted || terminate {
				// src is exhausted, send Complete signal
				queue = pipe.Complete(queue).([]interface{})
				// Complete may have flushed some stuff into the accum/queue
				if len(queue) > 0 {
					value, queue = queue[0], queue[1:]
					return value, false
				}
				// nope. so, we're done.
				return nil, true
			}

			// temporarily use the value var
			value, terminate = pipe.Reduce(queue, input)
			queue = value.([]interface{})
			if terminate {
				// this is here because it's less horrifying than the alternative
				queue = pipe.Complete(queue).([]interface{})
			}

			if len(queue) > 0 {
				value, queue = queue[0], queue[1:]
				return value, false
			}
		}
	}
}

// Given a channel, apply the transducer stack to values it produces,
// emitting values out the other end through the returned channel.
//
// This processor will spawn a separate goroutine that runs the transduction.
// This goroutine consumes on the provided chan, runs transduction, and sends
// resultant values into the returned chan. Thus, make sure that you're filling
// the input channel and consuming on the resultant channel from separate
// goroutines.
//
// The second parameter determines the buffering of the returned channel (it is
// passed directly to the make() call).
func Go(c <-chan interface{}, retbuf int, tlist ...Transducer) <-chan interface{} {
	out := make(chan interface{}, retbuf)
	pipe := CreatePipeline(chanReducer{c: out}, tlist...)

	var accum struct{} // accum is unused in this mode
	var terminate bool

	go func() {
		for v := range c {
			_, terminate = pipe.Reduce(accum, v)
			if terminate {
				break
			}
		}

		pipe.Complete(accum)
	}()

	return out
}

type chanReducer struct {
	c chan interface{}
}

func (c chanReducer) Reduce(accum interface{}, value interface{}) (interface{}, bool) {
	c.c <- value
	return accum, false
}

func (c chanReducer) Complete(accum interface{}) interface{} {
	close(c.c)
	return accum
}

func (c chanReducer) Init() interface{} {
	return nil
}
