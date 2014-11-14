package transduce

import "fmt"

var fml = fmt.Println

type Reducer func(interface{}, int) interface{}

// This is an outer piece, so doesn't need a type - use em how you want
// type Materializer func(Transducer, Iterator)

type Transducer func(Reducer) Reducer

type Mapper func(int) int
type Filterer func(int) bool

type ReducingFunc func(accum interface{}, val interface{}) (result interface{})

func Sum(accum interface{}, val int) (result interface{}) {
	return accum.(int) + val
}

// Basic Mapper function (increments by 1)
func inc(v int) int {
	return v + 1
}

// Basic Filterer function (true if even)
func even(v int) bool {
	return v%2 == 0
}

// Bind a function to the given collection that will allow traversal for reducing
func MakeReduce(collection interface{}) ValueStream {
	// If the structure already provides a reducing method, just return that.
	if c, ok := collection.(Streamable); ok {
		return c.AsStream()
	}
	switch c := collection.(type) {
	case []int:
		return iteratorToValueStream(&IntSliceIterator{slice: c})
	default:
		panic("not supported...yet")
	}
}

func Seq(vs ValueStream, init []int, tlist ...Transducer) []int {
	//fml(tlist)
	// Final reducing func; appends to our list
	t := func(accum interface{}, value int) interface{} {
		//fml("Seq inner:", accum, value)
		return append(accum.([]int), value)
	}

	// Walk backwards through transducer list to assemble in
	// correct order
	for i := len(tlist) - 1; i >= 0; i-- {
		//fml(tlist[i])
		t = tlist[i](t)
	}

	var v interface{}
	var done bool
	var ret interface{} = init

	for {
		v, done = vs()
		if done {
			break
		}

		////fml("Main loop:", v)
		// weird that we do nothing here
		ret = t(ret, v.(int))
	}

	return ret.([]int)
}

func Map(f Mapper) Transducer {
	return func(r Reducer) Reducer {
		return func(accum interface{}, value int) interface{} {
			////fml("Map:", accum, value)
			return r(accum, f(value))
		}
	}
}

func Filter(f Filterer) Transducer {
	return func(r Reducer) Reducer {
		return func(accum interface{}, value int) interface{} {
			//fml("Filter:", accum, value)
			if f(value) {
				return r(accum, value)
			} else {
				return accum
			}
		}
	}
}
