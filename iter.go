package transducers

// ValueStreams are the core abstraction that facilitate value-oriented
// communication in a transduction pipeline. Unfortunately, various typing
// issues preclude the use of slices directly.
//
// They are a lowest-possible-denominator iterator concept: forward-only,
// non-seeking, non-rewinding. They're kinda terrible. But they're also
// very simple: call the function. If the second val is true, the iterator
// is exhausted. If not, the first return value is the next value.
//
// The most important thing to understand is that ValueStreams are the
// *only* thing that transducers treat as being a set of values: if a slice
// or map is passed through a transduction process, transducers that deal
// with multiple values will treat a slice as a single value, but a ValueStream
// as multiple. This is why Exploders return a ValueStream.
//
// TODO At minimum, a proper implementation would probably need to include a
// adding a parameter that allows the caller to indicate they no longer
// need the stream. (Not quite a 'close', but possibly interpreted that way)
type ValueStream func() (value interface{}, done bool)

// Convenience function that receives from the stream and passes the
// emitted value to an injected function. Will not return until the
// stream reports being exhausted - watch for deadlocks!
func (vs ValueStream) Each(f func(interface{})) {
	for {
		v, done := vs()
		if done {
			return
		}
		f(v)
	}
}

// Recursively reads this stream out into an []interface{}.
//
// Will consume until the stream says its done - unsafe for infinite streams,
// and will block if the stream is based on a blocking datasource (e.g., chan).
func ToSlice(vs ValueStream) (into []interface{}) {
	for value, done := vs(); !done; value, done = vs() {
		if ivs, ok := value.(ValueStream); ok {
			ivs, value = ivs.Split()
			into = append(into, ToSlice(ivs))
		} else {
			into = append(into, value)
		}
	}

	return into
}

// Recursively create a slice from a duplicate (Split()) of the given stream.
//
// Will consume until the stream says its done - unsafe for infinite streams,
// and will block if the stream is based on a blocking datasource (e.g., chan).
//
// The slice is read out of the duplicate, and the passed value is repointed to
// an unconsumed stream, so it is safe for convenient use patterns. See example.
func IntoSlice(vs *ValueStream) (into []interface{}) {
	var dup ValueStream
	// write through pointer to replace original val with first split
	*vs, dup = vs.Split() // write through pointer to replace

	for value, done := dup(); !done; value, done = dup() {
		if ivs, ok := value.(ValueStream); ok {
			into = append(into, IntoSlice(&ivs))
		} else {
			into = append(into, value)
		}
	}

	return into
}

// Splits a value stream into two identical streams that can both be
// consumed independently.
//
// Note that calls to the original stream will still work - and any values
// consumed that way will be missed by the split streams. Be very careful!
//
// TODO I think this might leak?
// TODO figure out if there's a nifty way to make this threadsafe
func (vs ValueStream) Split() (ValueStream, ValueStream) {
	var src ValueStream = vs
	var f1i, f2i int
	var held []interface{}

	return func() (value interface{}, done bool) {
			if f1i >= f2i {
				// this stream is ahead, pull from the source
				value, done = src()
				if !done {
					// recursively dup streams
					if vs, ok := value.(ValueStream); ok {
						vs, value = vs.Split()
						held = append(held, vs)
					} else {
						held = append(held, value)
					}
				}
			} else {
				value, held = held[0], held[1:]
			}

			if !done {
				f1i++
			}
			return
		}, func() (value interface{}, done bool) {
			if f2i >= f1i {
				// this stream is ahead, pull from the source
				value, done = src()
				if !done {
					// recursively dup streams
					if vs, ok := value.(ValueStream); ok {
						vs, value = vs.Split()
						held = append(held, vs)
					} else {
						held = append(held, value)
					}
				}
			} else {
				value, held = held[0], held[1:]
			}

			if !done {
				f2i++
			}
			return
		}
}

func StreamIntoChan(vs ValueStream, c chan<- interface{}) {
	vs.Each(func(v interface{}) {
		c <- v
	})
	close(c)
}

// Assuming this stream contains other streams, Flatten returns a new stream
// that removes all nesting on the fly, flattening all the streams down into
// the same top-level construct. It's the linearized results of a
// depth-first tree traversal on an arbitrarily deep tree of nested streams.
//
// Meaning that, it a stream containing other streams looking like:
//
// [[1 [2 3] 4] 5 6 [7 [8 [9] [10 11]]]]
//
// And presents it through the returned stream as:
//
// [1 2 3 4 5 6 7 8 9 10 11]
//
// Remember - streams are forward-only, and the flattening stream wraps the
// original source stream. Once you call this, if you consume from the original
// source stream again, that value will be lost to the flattener.
func (vs ValueStream) Flatten() ValueStream {
	// create stack of streams and push the first one on
	var ss []ValueStream
	ss = append(ss, vs)

	var f ValueStream

	f = func() (value interface{}, done bool) {
		size := len(ss)
		if size == 0 {
			// no streams left, we're definitely done
			return nil, true
		}

		// grab value from stream on top of stack
		value, done = ss[size-1]()

		if done {
			// this stream is done; pop the stack and recurse
			ss = ss[:size-1]
			return f()
		}

		if innerstream, ok := value.(ValueStream); ok {
			// we got another stream, push it on the stack and recurse
			ss = append(ss, innerstream)
			return f()
		}

		// most basic case - we found a leaf. return it.
		return
	}

	return f
}

// If something has a special way of representing itself a stream, it should
// implement this method.
type Streamable interface {
	AsStream() ValueStream
}

// Bind a function to the given collection that will allow traversal for reducing
func ToStream(collection interface{}) ValueStream {
	// If the structure already provides a reducing method, just return that.
	if c, ok := collection.(Streamable); ok {
		return c.AsStream()
	}

	switch c := collection.(type) {
	case []int:
		return iteratorToValueStream(&intSliceIterator{slice: c})
	case []interface{}:
		return valueSlice(c).AsStream()
	case ValueStream:
		return c
	case <-chan interface{}:
		return func() (value interface{}, done bool) {
			value, done = <-c
			return
		}
	case chan interface{}:
		return func() (value interface{}, done bool) {
			value, done = <-c
			return
		}
	default:
		panic("not supported...yet")
	}
}

// Wrap an iterator up into a ValueStream func.
func iteratorToValueStream(i Iterator) func() (value interface{}, done bool) {
	return func() (interface{}, bool) {
		if !i.Valid() {
			i.Done()
			return nil, true
		}

		v := i.Current()
		i.Next()

		return v, false
	}
}

// Interleaves two streams together, getting a value from the first stream,
// then a value from the second stream.
//
// Values from both streams are collected at the same time, and if either
// input is exhausted, the interleaved stream terminates immediately, even if
// the other stream does have a value available.
func Interleave(s1 ValueStream, s2 ValueStream) ValueStream {
	var done bool
	var v1, v2 interface{}
	var index int

	return func() (interface{}, bool) {
		if done {
			return nil, done
		}

		if index%2 == 0 {
			// check both streams at once - if either is exhausted, stop
			v1, done = s1()
			if !done {
				v2, done = s2()
			}
			if done {
				return nil, done
			}

			index++
			return v1, false
		} else {
			index++
			return v2, false
		}

	}
}

// Simple iterator interface. Mostly used internally to handle slices.
//
// TODO Done() isn't used at all. also kinda terrible in general.
type Iterator interface {
	Current() (value interface{})
	Next()
	Valid() bool
	Done()
}

type intSliceIterator struct {
	slice []int
	pos   int
}

func (i *intSliceIterator) Current() interface{} {
	return i.slice[i.pos]
}

func (i *intSliceIterator) Next() {
	// TODO atomicity
	i.pos++
}

func (i *intSliceIterator) Valid() (valid bool) {
	return i.pos < len(i.slice)
}

func (i *intSliceIterator) Done() {

}

type valueSlice []interface{}

func (s valueSlice) AsStream() ValueStream {
	return iteratorToValueStream(&interfaceSliceIterator{slice: s})
}

type interfaceSliceIterator struct {
	slice []interface{}
	pos   int
}

func (i *interfaceSliceIterator) Current() interface{} {
	return i.slice[i.pos]
}

func (i *interfaceSliceIterator) Next() {
	// TODO atomicity
	i.pos++
}

func (i *interfaceSliceIterator) Valid() (valid bool) {
	return i.pos < len(i.slice)
}

func (i *interfaceSliceIterator) Done() {

}
