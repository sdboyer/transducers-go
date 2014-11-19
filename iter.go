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

// TODO maybe do this as a linked list instead?
type CompoundValueStream []ValueStream

func (cvs CompoundValueStream) AsStream() ValueStream {
	var pos int
	var f ValueStream

	f = func() (value interface{}, done bool) {
		value, done = cvs[pos]() // TODO superdee duper not threadsafe

		if !done {
			return value, done
		}

		// recurse through the streams till we get a value, or we hit the end
		pos++
		if pos >= len(cvs) { // no more streams
			return nil, true
		}

		return f()
	}

	return f
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

	f = func() (value interface{}, done bool) {
		length := len(ss)
		if length == 0 {
			// no streams left, we're definitely done
			return nil, true
		}

		// grab value from stream on top of stack
		value, done = ss[length-1]()

		if done {
			// this stream is done; pop the stack and recurse
			ss = ss[:length-1]
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

		// sad that defer has performance issues
		//
		// this approach signals termination to  the iterator when no longer valid
		//defer func() {
		//i.Next()
		//if !i.Valid() {
		//i.Done()
		//}
		//}()
		//
		// this approach just makes sure next gets called
		// defer i.Next()

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
