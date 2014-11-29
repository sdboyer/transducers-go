package transduce

import (
	"fmt"
)

func AppendReducer() ReduceStep {
	b := BareReducer()
	b.R = func(accum interface{}, value interface{}) (interface{}, bool) {
		return append(accum.([]interface{}), value), false
	}
	b.I = func() interface{} {
		return make([]interface{}, 0)
	}

	return b
}

func Example_clojureParity() {
	// mirrors Rich Hickey's original "transducerfun.clj" gist:
	//  https://gist.github.com/richhickey/b5aefa622180681e1c81
	// note that that syntax is out of date and will not run, but this does:
	//  https://gist.github.com/sdboyer/9fca652f492257f35a41
	xform := []Transducer{
		Map(Inc),
		Filter(Even),
		Dedupe(),
		Mapcat(Range),
		Chunk(3),
		ChunkBy(func(value interface{}) interface{} {
			return sum(value.(ValueStream)) > 7
		}),
		Mapcat(Flatten),
		RandomSample(1.0),
		TakeNth(1),
		Keep(func(v interface{}) interface{} {
			if v.(int)%2 != 0 {
				return v.(int) * v.(int)
			} else {
				return nil
			}
		}),
		KeepIndexed(func(i int, v interface{}) interface{} {
			if i%2 == 0 {
				return i * v.(int)
			} else {
				return nil
			}
		}),
		Replace(map[interface{}]interface{}{2: "two", 6: "six", 18: "eighteen"}),
		Take(11),
		TakeWhile(func(v interface{}) bool {
			return v != 300
		}),
		Drop(1),
		DropWhile(IsString),
		Remove(IsString),
	}

	// An []interface{} slice (containing only ints) with vals [0 0 1 1 2 2 ... 17 17]
	data := ToSlice(Interleave(Range(18), Range(20)))

	// If this is hard to visualize, you can add logging transducers that will show the
	// set of values at each stage of transduction:
	// xform = AttachLoggers(fmt.Printf, xform...)

	// NB: the original clojure gists also have (sequence ...) and (into []...) processors.
	// I didn't replicate sequence because, as best I can figure, it's redundant with
	// Eduction in a golang context. I didn't replicate the latter because it's a use pattern
	// that is awkward with strict typing, and basically the same as Transduce.

	// reduce immediately, appending the results of transduction into an int slice.
	fmt.Println(Transduce(data, AppendReducer(), xform...))
	// produces first line of Output: [36 200 10]

	// Eduction takes the same transduction stack, but operates lazily - it returns
	// a ValueStream that triggers transduction only as a result of requesting
	// values from that returned stream.
	fmt.Println(ToSlice(Eduction(data, xform...)))
	// produces second line of Output: [36 200 10]

	// Same transduction stack again, but now with the Go processor, which takes an
	// input channel, runs transduction in a separate goroutine, and sends results
	// back out through an output channel (the one returned from the Go func).
	input := make(chan interface{}, 0)
	go StreamIntoChan(ToStream(data), input)
	output := Go(input, 0, xform...)
	for v := range output {
		fmt.Println(v)
	}
	// same output as other processors, just one value per line

	// Output:
	// [36 200 10]
	// [36 200 10]
	// 36
	// 200
	// 10
}

func Example_transduce() {
	// Transducing does an immediate (non-lazy) application of the transduction
	// stack to the incoming ValueStream.

	// Increments [0 1 2 3 4] by 1, then filters out odd values.
	fmt.Println(Transduce(Range(6), AppendReducer(), Map(Inc), Filter(Even)))
	// Output:
	// [2 4 6]
}
