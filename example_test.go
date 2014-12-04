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

	// NB: the original clojure gists also have (sequence ...) and (into []...)
	// processors. I didn't replicate `sequence` because, as best I can figure,
	// it's redundant with Eduction in the context I've created (no seqs).
	// I didn't replicate `into` because it's a use pattern that is awkward with
	// static typing, and is readily accomplished via Transduce.

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

func Example_clojureParityLogged() {
	// Each moving part can be hard to follow, so this is the same example but with
	// where we wrap the transducer stack with loggers.
	xform := AttachLoggers(fmt.Printf,
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
	)

	data := ToSlice(Interleave(Range(18), Range(20)))
	Transduce(data, Append(), xform...)

	// Output:
	//SRC -> [0 0 1 1 2 2 3 3 4 4 5 5 6 6 7 7 8 8 9 9 10 10 11] |TERM|
	//	transduce.map_r -> [1 1 2 2 3 3 4 4 5 5 6 6 7 7 8 8 9 9 10 10 11 11 12] |TERM|
	//	transduce.filter -> [2 2 4 4 6 6 8 8 10 10 12] |TERM|
	//	*transduce.dedupe -> [2 4 6 8 10 12] |TERM|
	//	transduce.mapcat -> [0 1 0 1 2 3 0 1 2 3 4 5 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 8 9 0 1 2] |TERM|
	//	*transduce.chunk -> [[0 1 0] [1 2 3] [0 1 2] [3 4 5] [0 1 2] [3 4 5] [6 7 0] [1 2 3] [4 5 6] [7 8 9] [0 1 2]] |TERM|
	//	*transduce.chunkBy -> [[[0 1 0] [1 2 3] [0 1 2]] [[3 4 5]] [[0 1 2]] [[3 4 5] [6 7 0]] [[1 2 3]] [[4 5 6] [7 8 9]]] |TERM|
	//	transduce.mapcat -> [0 1 0 1 2 3 0 1 2 3 4 5 0 1 2 3 4 5 6 7 0 1 2 3 4 5] |TERM|
	//	transduce.randomSample -> [0 1 0 1 2 3 0 1 2 3 4 5 0 1 2 3 4 5 6 7 0 1 2 3 4 5] |TERM|
	//	transduce.takeNth -> [0 1 0 1 2 3 0 1 2 3 4 5 0 1 2 3 4 5 6 7 0 1 2 3 4 5] |TERM|
	//	transduce.keep -> [1 1 9 1 9 25 1 9 25 49 1 9 25] |TERM|
	//	*transduce.keepIndexed -> [0 18 36 6 200 10 300] |TERM|
	//	transduce.replace -> [0 eighteen 36 six 200 10 300] |TERM|
	//	*transduce.take -> [0 eighteen 36 six 200 10 300] |TERM|
	//	transduce.takeWhile -> [0 eighteen 36 six 200 10]
	//	*transduce.drop -> [eighteen 36 six 200 10]
	//	*transduce.dropWhile -> [36 six 200 10]
	//	transduce.remove -> [36 200 10]
	//	transduce.append_bottom
	//END
}

func ExampleTransduce() {
	// Transducing does an immediate (non-lazy) application of the transduction
	// stack to the incoming ValueStream.

	// Increments [0 1 2 3 4] by 1, then filters out odd values.
	fmt.Println(Transduce(Range(6), Append(), Map(Inc), Filter(Even)))
	// Output:
	// [2 4 6]
}

func ExampleIntoSlice() {
	// ValueStreams are forward-only iterators, so reading them into a slice will
	// exhaust the iterator. Not great if you have to pass the stream along.
	stream := Range(3)
	fmt.Println(ToSlice(stream)) // [0 1 2]
	fmt.Println(ToSlice(stream)) // [], because the first call exhausted the stream

	// IntoSlice() will read a dup/split of the stream, then use some pointer
	// trickery to update local variable in your calling scope:
	stream = Range(3)
	fmt.Println(IntoSlice(&stream)) // [0 1 2]
	fmt.Println(IntoSlice(&stream)) // [0 1 2]

	// Output:
	// [0 1 2]
	// []
	// [0 1 2]
	// [0 1 2]
}

func ExampleEscape() {
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
