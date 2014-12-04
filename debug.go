package transduce

// Interleaves logging transducers into the provided transducer stack.
//
// The first parameter is a logging function - e.g., fmt.Printf - into which
// the logging transducers will send their output through this function.
//
// NOTE: this will block, and/or exhaust memory, on infinite streams.
func AttachLoggers(logger func(string, ...interface{}) (int, error), tds ...Transducer) []Transducer {
	tlfunc := func(r ReduceStep) ReduceStep {
		tl := &topLogger{reduceLogger{logger: logger, next: r}}
		return tl
	}

	newstack := make([]Transducer, 0)
	newstack = append(newstack, tlfunc)
	for i := 0; i < len(tds); i++ {
		newstack = append(newstack, tds[i])
		newstack = append(newstack, logtd(logger))
	}

	return newstack
}

type topLogger struct {
	reduceLogger
}

func (r *topLogger) Complete(accum interface{}) interface{} {
	if r.term {
		r.logger("SRC -> %v |TERM|\n\t%T", r.values, r.next)
	} else {
		r.logger("SRC -> %v\n\t%T", r.values, r.next)
	}
	//r.logger("SRC -> %v\n\t%T", r.values, r.next)
	accum = r.next.Complete(accum)
	r.logger("\nEND\n")

	return accum
}

func logtd(logger func(string, ...interface{}) (int, error)) Transducer {
	return func(r ReduceStep) ReduceStep {
		lt := &reduceLogger{logger: logger, next: r}
		return lt
	}
}

type reduceLogger struct {
	values []interface{}
	logger func(string, ...interface{}) (int, error)
	next   ReduceStep
	term   bool
}

func (r *reduceLogger) Init() interface{} {
	return r.next.Init()
}

func (r *reduceLogger) Complete(accum interface{}) interface{} {
	if r.term {
		r.logger(" -> %v |TERM|\n\t%T", r.values, r.next)
	} else {
		r.logger(" -> %v\n\t%T", r.values, r.next)
	}
	return r.next.Complete(accum)
}

func (r *reduceLogger) Reduce(accum interface{}, value interface{}) (interface{}, bool) {
	// if the transducer produces a ValueStream, dup and dump it. (so, already not infinite-safe)
	if vs, ok := value.(ValueStream); ok {
		r.values = append(r.values, IntoSlice(&vs))
		value = vs
	} else {
		r.values = append(r.values, value)
	}

	accum, r.term = r.next.Reduce(accum, value)
	return accum, r.term
}
