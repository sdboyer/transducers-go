package transduce

import "fmt"

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

	// NB: the original clojure gists also have (sequence ...) and (into []...)
	// processors. I didn't replicate sequence because, as best I can figure,
	// it's redundant with Eduction in a golang context.
	// I didn't replicate into because it's a use pattern that is awkward with
	// strict typing, and is readily accomplished via Transduce.

	// reduce immediately, appending the results of transduction into an int slice.
	fmt.Println(Transduce(data, Append(), xform...))
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
	fmt.Println(Transduce(Range(6), Append(), Map(Inc), Filter(Even)))
	// Output:
	// [2 4 6]
}

func Example_connectedProcesses() {
	// The Escape transducer allows values in a transduction process to "escape"
	// partway through processing into a channel. That channel can be used to
	// feed another transduction process, creating a well-defined cascade of
	// transduction processes.

	c := make(chan interface{}, 0)
	input := make(chan interface{}, 0)
	// send [0 1 2 3 4] into the input channel in separate goroutine
	go StreamIntoChan(Range(5), input)
	// connect c's input end to Escape transducer, start transduction in goroutine
	out1 := Go(input, 0, Escape(Even, c, true))
	// connect c's output end to a second transduction process in goroutine
	out2 := Go(c, 0, Map(Inc), Map(Inc), Map(Inc))

	slice1, slice2 := make([]int, 0), make([]int, 0)
	// Consume this in a separate goroutine because it's sorta in the middle
	go func() {
		for v := range out1 {
			slice1 = append(slice1, v.(int))
		}
	}()

	// Safe to consume this chan in the calling goroutine because it's at the bottom
	for v := range out2 {
		slice2 = append(slice2, v.(int))
	}

	fmt.Println(slice1)
	fmt.Println(slice2)
	// Output:
	// [1 3]
	// [3 5 7]
}
