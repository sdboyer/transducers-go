package transduce

//type Reducer func(accum interface{}, input interface{}) (result interface{})
//type Reducer func(accum int, input int) int
type Reducer func(interface{}, int) interface{}

//type Transducer func(Reducer) Reducer

type Mapper func(int) int
type Filterer func(int) bool

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
	}, make([]int, 0, len(collection))).([]int)
}
