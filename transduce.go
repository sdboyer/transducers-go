package transduce

import "fmt"

//var fml = fmt.Println

type Reducer func(interface{}, int) interface{}

// This is an outer piece, so doesn't need a type - use em how you want
// type Materializer func(Transducer, Iterator)

type Transducer func(Reducer) Reducer

type Mapper func(int) interface{}
type Filterer func(int) bool

const dbg = true

func fml(v ...interface{}) {
	if dbg {
		fmt.Println(v)
	}
}

// Exploders transform a value of some type into a stream of values.
// No guarantees about the relationship between the type of input and output;
// output may be a collection of the input type, or may not.
type Exploder func(interface{}) ValueStream

type ReducingFunc func(accum interface{}, val interface{}) (result interface{})

func Sum(accum interface{}, val int) (result interface{}) {
	return accum.(int) + val
}

// Basic Mapper function (increments by 1)
func inc(v int) interface{} {
	return v + 1
}

// Basic Filterer function (true if even)
func even(v int) bool {
	return v%2 == 0
}

// Dumb little thing to emulate clojure's range behavior
func Range(limit interface{}) ValueStream {
	l := limit.(int)
	slice := make([]int, l)

	for i := 0; i < l; i++ {
		slice[i] = i
	}

	return MakeReduce(slice) // lazy and inefficient, do it directly
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

func Identity(accum interface{}, val int) interface{} {
	return val
}

func Seq(vs ValueStream, init []int, tlist ...Transducer) []int {
	fml(tlist)
	// Final reducing func - append to the list
	t := Append(Identity)

	// Walk backwards through transducer list to assemble in
	// correct order
	for i := len(tlist) - 1; i >= 0; i-- {
		fml(tlist[i])
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

		fml("Main loop:", v)
		// weird that we do nothing here
		ret = t(ret, v.(int))
	}

	return ret.([]int)
}

func Map(f Mapper) Transducer {
	return func(r Reducer) Reducer {
		return func(accum interface{}, value int) interface{} {
			fml("Map:", accum, value)
			return r(accum, f(value).(int))
		}
	}
}

func Filter(f Filterer) Transducer {
	return func(r Reducer) Reducer {
		return func(accum interface{}, value int) interface{} {
			fml("Filter:", accum, value)
			if f(value) {
				return r(accum, value)
			} else {
				return accum
			}
		}
	}
}

func Append(r Reducer) Reducer {
	return func(accum interface{}, val int) interface{} {
		fml(accum)
		switch v := r(accum, val).(type) {
		case []int:
			return append(accum.([]int), v...)
		case int:
			return append(accum.([]int), v)
		default:
			panic("not supported")
		}
	}
}

// Mapcat first runs an exploder, then ''concats' results by
// passing each individual value along to the next transducer
// in the stack.
func Mapcat(f Exploder) Transducer {
	return func(r Reducer) Reducer {
		return func(accum interface{}, value int) interface{} {
			fml("Processing explode val:", value)
			stream := f(value)

			var v interface{}
			var done bool

			for { // <-- the *loop* is the 'cat'
				v, done = stream()
				if done {
					break
				}
				fml("Calling next t on val:", v, "accum is:", accum)

				accum = r(accum, v.(int))
			}

			return accum
		}
	}
}
