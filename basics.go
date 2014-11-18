package transduce

// Bedrock reducer.
func Reduce(coll []int, f Reducer, input interface{}) interface{} {
	for _, v := range coll {
		input = f(input, v)
	}

	return input
}

// basic direct map func
func DirectMap(f Mapper, collection []int) []int {
	newcoll := make([]int, len(collection))

	for k, v := range collection {
		newcoll[k] = f(v).(int)
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
	return Reduce(collection, func(accum interface{}, datum interface{}) interface{} {
		return append(accum.([]int), f(datum).(int))
	}, make([]int, 0)).([]int)
}

// filter, expressed directly through reduce
func FilterThruReduce(f Filterer, collection []int) []int {
	return Reduce(collection, func(accum interface{}, datum interface{}) interface{} {
		if f(datum) {
			return append(accum.([]int), datum.(int))
		} else {
			return accum
		}
	}, make([]int, 0)).([]int)
}

// map, expressed indirectly through a returned reducer func
func MapFunc(f Mapper) Reducer {
	return func(accum interface{}, datum interface{}) interface{} {
		return append(accum.([]int), f(datum).(int))
	}
}

// filter, expressed indirectly through a returned reducer func
func FilterFunc(f Filterer) Reducer {
	return func(accum interface{}, datum interface{}) interface{} {
		if f(datum) {
			return append(accum.([]int), datum.(int))
		} else {
			return accum
		}
	}
}
