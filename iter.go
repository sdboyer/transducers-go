package transduce

// The lowest-possible-denominator iteration concept. Kinda terrible.
//
// TODO quite possible that this whole concept should be redone with channels
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

// Duplicates a ValueStream by moving the pointer to the original stream
// to an internal var, passing calls from either dup'd stream to the
// origin stream, and holding values provided from origin until both dup'd
// streams have consumed the value.
//
// Clean up after yourself with this one - it could be leaky. Also, very much
// not threadsafe.
//
// TODO figure out if there's a nifty way to make this threadsafe
func (vs *ValueStream) Dup() ValueStream {
	var src ValueStream = *vs
	var f1i, f2i int
	var held []interface{}

	*vs = func() (value interface{}, done bool) {
		if f1i >= f2i {
			// this stream is ahead, pull from the source
			value, done = src()
			if !done {
				held = append(held, value)
			}
		} else {
			value, held = held[0], held[1:]
		}

		if !done {
			f1i++
		}
		return
	}

	return func() (value interface{}, done bool) {
		if f2i >= f1i {
			// this stream is ahead, pull from the source
			value, done = src()
			if !done {
				held = append(held, value)
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

// Takes a ValueStream that (presumably) produces other ValueStreams, and,
// ostensibly for the caller, flattens them together into a single ValueStream
// by walking depth-first through an arbitrarily deep set of ValueStreams until
// actual values are found, then returning those directly out.
func flattenValueStream(vs ValueStream) ValueStream {
	// create stack of streams and push the first one on
	var ss []ValueStream
	ss = append(ss, vs)

	var f ValueStream
	fml("FVS: creating stream stack func")

	f = func() (value interface{}, done bool) {
		size := len(ss)
		fml("FVS: stream stack now size", size)
		if size == 0 {
			// no streams left, we're definitely done
			return nil, true
		}

		// grab value from stream on top of stack
		value, done = ss[size-1]()
		fml("FVS: value grabbed:", value, "done state", done)

		if done {
			// this stream is done; pop the stack and recurse
			fml("FVS: finished stream, popping and recursing")
			ss = ss[:size-1]
			return f()
		}

		if innerstream, ok := value.(ValueStream); ok {
			// we got another stream, push it on the stack and recurse
			fml("FVS: found new stream, pushing and recursing")
			ss = append(ss, innerstream)
			return f()
		}

		// most basic case - we found a leaf. return it.
		fml("FVS: found a value:", value)
		return
	}

	return f
}

type Streamable interface {
	AsStream() ValueStream
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

type Iterator interface {
	Current() (value interface{})
	Next()
	Valid() bool
	Done()
}

type IntSliceIterator struct {
	slice []int
	pos   int
}

func (i *IntSliceIterator) Current() interface{} {
	return i.slice[i.pos]
}

func (i *IntSliceIterator) Next() {
	// TODO atomicity
	i.pos++
}

func (i *IntSliceIterator) Valid() (valid bool) {
	return i.pos < len(i.slice)
}

func (i *IntSliceIterator) Done() {

}

type ValueSlice []interface{}

func (s ValueSlice) AsStream() ValueStream {
	return iteratorToValueStream(&InterfaceSliceIterator{slice: s})
}

type InterfaceSliceIterator struct {
	slice []interface{}
	pos   int
}

func (i *InterfaceSliceIterator) Current() interface{} {
	return i.slice[i.pos]
}

func (i *InterfaceSliceIterator) Next() {
	// TODO atomicity
	i.pos++
}

func (i *InterfaceSliceIterator) Valid() (valid bool) {
	return i.pos < len(i.slice)
}

func (i *InterfaceSliceIterator) Done() {

}
