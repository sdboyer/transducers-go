package transduce

import "reflect"

// Interleaves logging transducers into the provided transducer stack.
//
// The first parameter is a logging function - e.g., fmt.Printf - into which
// the logging transducers will send their output through this function..
//
// IMPORTANT NOTE: a transducer stack ([]Transducer) is ordinarily expected to be
// stateless - state is initialized when the bottom reducer is provided and the
// stack is resolved into a pipeline - see CreatePipeline(). This logger breaks
// that in the interest of providing the most informative possible output. As
// such, should be used solely for demonstrative or debugging purposes.
//
// FURTHER NOTE: this will block, and/or exhaust memory, on infinite sequences.
func AttachLoggers(logger func(string, ...interface{}), tds ...Transducer) []Transducer {
	tl := &topLogger{logtds: make([]*reduceLogger, 0), logger: logger}
	tlfunc := func(r ReduceStep) ReduceStep {
		tl.next = r
		return tl
	}

	newstack := make([]Transducer, 0)
	newstack = append(newstack, tlfunc)
	for i := 0; i < len(tds); i++ {
		newstack = append(newstack, tds[i])
		newstack = append(newstack, logtd(tl, logger))
	}

	return newstack
}

type topLogger struct {
	logtds   []*reduceLogger
	values   []interface{}
	logger   func(string, ...interface{})
	next     ReduceStep
	nextType reflect.Type // just store this so we're not constantly recalculating
}

func (r *topLogger) Complete(accum interface{}) interface{} {
	accum = r.next.Complete(accum)
	r.logger("Transducer stack: %T", r.next)
	for _, t := range r.logtds {
		r.logger(" | %T ", t.next)
	}
	r.logger("\n")

	r.logger("SRC -> %v\n\t%T", r.values, r.next)
	for _, t := range r.logtds {
		t.dump()
	}
	r.logger("\nEND\n")

	return accum
}

func (r *topLogger) Reduce(accum interface{}, value interface{}) (interface{}, bool) {
	// if the transducer produces a ValueStream, dup and dump it. (so, already not infinite-safe)
	if vs, ok := value.(ValueStream); ok {
		inner := make([]interface{}, 0)
		value = (&vs).Dup()
		vs.Each(func(v interface{}) {
			inner = append(inner, v)
		})
		r.values = append(r.values, inner)
	} else {
		r.values = append(r.values, value)
	}

	var term bool
	accum, term = r.next.Reduce(accum, value)
	if term {
		r.values = append(r.values, "!")
	}
	return accum, term
}

func logtd(tl *topLogger, logger func(string, ...interface{})) Transducer {
	return func(r ReduceStep) ReduceStep {
		lt := &reduceLogger{logger: logger, next: r}
		// these are called in reverse order, so must prepend
		tl.logtds = append([]*reduceLogger{lt}, tl.logtds...)
		return lt
	}
}

type reduceLogger struct {
	tl     *topLogger
	values []interface{}
	logger func(string, ...interface{})
	next   ReduceStep
}

func (r *reduceLogger) Complete(accum interface{}) interface{} {
	return r.next.Complete(accum)
}

func (r *reduceLogger) Reduce(accum interface{}, value interface{}) (interface{}, bool) {
	// if the transducer produces a ValueStream, dup and dump it. (so, already not infinite-safe)
	if vs, ok := value.(ValueStream); ok {
		inner := make([]interface{}, 0)
		value = (&vs).Dup()
		vs.Each(func(v interface{}) {
			inner = append(inner, v)
		})
		r.values = append(r.values, inner)
	} else {
		r.values = append(r.values, value)
	}

	var term bool
	accum, term = r.next.Reduce(accum, value)
	if term {
		r.values = append(r.values, "!")
	}
	return accum, term
}

func (r *reduceLogger) dump() {
	r.logger(" -> %v\n\t%T", r.values, r.next)
}
