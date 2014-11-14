package transduce

import "fmt"

var fml = fmt.Println

//type Reducer func(accum interface{}, input interface{}) (result interface{})
//type Reducer func(accum int, input int) int
type Reducer func(interface{}, int) interface{}

//type Reducible func(collection []int, f Reducer, init interface{}) interface{}

// This is an outer piece, so doesn't need a type - use em how you want
// type Materializer func(Transducer, Iterator)

type TransduceReceiver func(...Transducer)

type ReduceFunctor interface {
	Reduced() bool
}

type Transducer func(Reducer) Reducer

type TransductionStack []Transducer

type Mapper func(int) int
type Filterer func(int) bool

type ReducingFunc func(accum interface{}, val interface{}) (result interface{})

func Append(accum interface{}, val int) (result interface{}) {
	return append(accum.([]int), val)
}

func Sum(accum interface{}, val int) (result interface{}) {
	return accum.(int) + val
}

// Bedrock reducer.
func Reduce(coll []int, f Reducer, input interface{}) interface{} {
	for _, v := range coll {
		input = f(input, v)
	}

	return input
}

// Basic Mapper function (increments by 1)
func inc(v int) int {
	return v + 1
}

// Basic Filterer function (true if even)
func even(v int) bool {
	return v%2 == 0
}

// basic direct map func
func DirectMap(f Mapper, collection []int) []int {
	newcoll := make([]int, len(collection))

	for k, v := range collection {
		newcoll[k] = f(v)
	}

	return newcoll
}

// basic direct filtering func
func DirectFilter(f Filterer, collection []int) []int {
	newcoll := make([]int, 0, len(collection))

	for _, v := range collection {
		if f(v) {
			newcoll = append(newcoll, v)
		}
	}

	return newcoll
}

// map, expressed directly through reduce
func MapThruReduce(f Mapper, collection []int) []int {
	return Reduce(collection, func(accum interface{}, datum int) interface{} {
		return append(accum.([]int), f(datum))
	}, make([]int, 0)).([]int)
}

// filter, expressed directly through reduce
func FilterThruReduce(f Filterer, collection []int) []int {
	return Reduce(collection, func(accum interface{}, datum int) interface{} {
		if f(datum) {
			return append(accum.([]int), datum)
		} else {
			return accum
		}
	}, make([]int, 0)).([]int)
}

// map, expressed indirectly through a returned reducer func
func MapFunc(f Mapper) Reducer {
	return func(accum interface{}, datum int) interface{} {
		return append(accum.([]int), f(datum))
	}
}

// filter, expressed indirectly through a returned reducer func
func FilterFunc(f Filterer) Reducer {
	return func(accum interface{}, datum int) interface{} {
		if f(datum) {
			return append(accum.([]int), datum)
		} else {
			return accum
		}
	}
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

// Shitty func to compose funcs
func Compose(funcs ...Transducer) TransductionStack {
	return funcs
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
