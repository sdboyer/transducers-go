package transduce

// Transducer predicate function; used by Map, and others. Takes a value,
// transforms it, and returns the result.
type Mapper func(value interface{}) interface{}

// Trandsucer predicate function. Same as Map, but passes an index value
// indicating the number of times the predicate has been called.
type IndexedMapper func(index int, value interface{}) interface{}

// Transducer predicate function; used by most filtering-ish transducers. Takes
// a value and returns a bool, which the transducer uses to make a decision,
// typically (though not necessarily) about whether or not that value gets to
// proceed in the reduction chain.
type Filterer func(interface{}) bool

// Transducer predicate function. Exploders transform a value of some type into
// a stream of values. Used by Mapcat.
type Exploder func(interface{}) ValueStream

func sum(vs ValueStream) (total int) {
	vs.Each(func(value interface{}) {
		fml("SUM: total", total)
		total += value.(int)
	})
	fml("SUM: final total", total)

	return
}

func Sum(value interface{}) interface{} {
	return sum(value.(ValueStream))
}

func IsString(v interface{}) bool {
	_, ok := v.(string)
	return ok
}

// Basic Mapper function (increments by 1)
func Inc(value interface{}) interface{} {
	return value.(int) + 1
}

// Basic Filterer function (true if even)
func Even(value interface{}) bool {
	return value.(int)%2 == 0
}

// Dumb little thing to emulate clojure's range behavior
func t_range(l int) []int {
	slice := make([]int, l)

	for i := 0; i < l; i++ {
		slice[i] = i
	}

	return slice
}

// Flattens arbitrarily deep datastructures into a single ValueStream.
func Flatten(value interface{}) ValueStream {
	switch v := value.(type) {
	case ValueStream:
		return flattenValueStream(v)
	case []interface{}:
		// TODO maybe detect ValueStreams here, too, but probably better to just be consistent
		return ValueSlice(v).AsStream()
	case []int:
		return ToStream(v)
	case int, interface{}:
		var done bool
		// create single-eleement value stream
		return func() (interface{}, bool) {
			if done {
				return nil, true
			} else {
				done = true
				return v, false
			}
		}
	default:
		panic("not supported")
	}
}

// Wraps t_range into a ValueStream
func Range(limit interface{}) ValueStream {
	// lazy and inefficient to use MakeReduce here, do it directly
	return ToStream(t_range(limit.(int)))
}

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
