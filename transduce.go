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

// Basic incrementer function
func inc(v int) int {
	return v + 1
}

// basic direct map func
func DirectMap(f Mapper, collection []int) []int {
	for k, v := range collection {
		collection[k] = f(v)
	}

	return collection
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

func MapThruReduce(f Mapper, collection []int) []int {

}
