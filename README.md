# Transducers for Go

This is an implementation of transducers, a concept [from Clojure](http://clojure.org), for Go.

Transducers can be tricky to understand with just an abstract description, but here it is:

> Transducers are a means of building composable, reusable transformation algorithms.

Transducers were introduced in Clojure for a sorta-similar reason that `range` exists in Go: because they wanted one way of writing element-wise operations on channels *and* other collection structures (though that's just the tip of the iceberg).

I'm honestly not sure if I these are a good idea for Go or not. I've written this library as an exploratory experiment in their utility/applicability to Go, and would love feedback on transducers' utility for Go.

## What Transducers are

There's a lot out there already, no need to duplicate it here. Instead, know that this library attempts to faithfully reproduce their behaviors in Go, meaning that anything about how they *ought* to work as defined elsewhere, should hold true here.

Here's some resources - mostly in Clojure, of course:

* Rich Hickey's [StrangeLoop talk](https://www.youtube.com/watch?v=6mTbuzafcII) introducing transducers (and his recent [ClojureConj talk](https://www.youtube.com/watch?v=4KqUvG8HPYo))
* The [Clojure docs](http://clojure.org/transducers) page for transducers
* [Some](https://gist.github.com/ptaoussanis/e537bd8ffdc943bbbce7) [high-level](https://bendyworks.com/transducers-clojures-next-big-idea/) [summaries](http://thecomputersarewinning.com/post/Transducers-Are-Fundamental/) of transducers
* Some [examples](http://ianrumford.github.io/blog/2014/08/08/Some-trivial-examples-of-using-Clojure-Transducers/) of [uses](http://matthiasnehlsen.com/blog/2014/10/06/Building-Systems-in-Clojure-2/) for transducers...mostly just toy stuff
* A couple [blog](http://blog.podsnap.com/ducers2.html) [posts](http://conscientiousprogrammer.com/blog/2014/08/07/understanding-cloure-transducers-through-types/) examining type issues with transducers

## Pudding <= Proof

I'm calling this proof of concept "done" because [it's pretty much replicated](http://godoc.org/github.com/sdboyer/transducers-go#ex-package--ClojureParity) a very thorough example Rich Hickey put out there.

Here's some quick eye candy, though:

```go
// dot import for brevity, remember this is a nono
import . "github.com/sdboyer/transducers-go"

func main() {
	// To make things work, we need four things (definitions in glossary):
	// 1) an input stream
	input := Range(4) // ValueStream containing [0 1 2 3]
	// 2) a stack of Transducers
	transducers := []Transducer{Map(Inc), Filter(Even)} // increment then filter odds
	// 3) a reducer to put at the bottom of the transducer stack
	reducer := Append() // very simple reducer - just appends values into a []interface{}
	// 4) a processor that puts it all together
	result := Transduce(input, reducer, transducers...)

	fmt.Println(result) // [2 4]


	// Or, we can use the Go processor, which does the work in a separate goroutine
	// and returns results through a channel.

	// Make an input chan, and stream each value from Range(4) into it
	in_chan := make(chan interface{}, 0)
	go StreamIntoChan(Range(4), in_chan)

	// Go provides its own bottom reducer (that's where it sends values out through
	// the return channel). So we don't provide one - just the input channel. Note
	// that we can safely reuse the transducer stack we declared earlier.
	out_chan := Go(in_chan, 0, transducers...)

	result2 := make([]interface{}, 0) // zero out the slice
	for v := range out_chan {
		result2 = append(result2, v)
	}

	fmt.Println(result) // [2 4]
}
```

## The Arguments
I figure there's pros and cons to something like this. Makes sense to put em up front.

### Cons

* Dodges around the type system - there is little to no compile-time safety here.
* To that end: is Yet Another Generics Attempt™...though, see [#1](https://github.com/sdboyer/transducers-go/issues/1).
* Syntax is not as fluid as Clojure's (though creating such things is kind of a Lisp specialty).
* Pursuant to all of the above, it'd be hard to call this idiomatic Go.
* The `ValueStream` notion is a bedrock for this system, and has significant flaws.
* Since this is based on streams/sequences/iteration, there will be cases where it is unequivocally less efficient than batch processing (slices).
* Performance in general. While Reflect is not used at all (duh), I haven't done perf analysis yet, so I'm not sure how much overhead we're looking at. The stream operations in particular (splitting, slice->stream->slice) probably mean a lot of heap allocs and duplication of data.

### Pros

* Stream-based data processing - so, amenable to dealing with continuous/infinite/larger-than-memory datasets.
* Sure, channels let you be stream-based. But they're [low-level primitives](https://gist.github.com/kachayev/21e7fe149bc5ae0bd878). Plus they're largely orthogonal to this, which is about decomposing processing pipelines into their constituent parts.
* Transducers could be an interesting, powerful way of structuring applications into segments, or for encapsulating library logic in a way that is easy to reuse, and whose purpose is widely understood.
* While the loss of type assurances hurts - a lot - the spec for transducer behavior is clear enough that it's probably feasible to aim at "correctness" via exhaustive black-box tests. (hah)
* And about types - I found a little kernel of something useful when looking beyond parametric polymorphism - [more here](https://github.com/sdboyer/transducers-go/issues/1).

It's also been pointed out to me that these look a bit like [Google's Dataflow](http://googlecloudplatform.blogspot.com/2014/06/sneak-peek-google-cloud-dataflow-a-cloud-native-data-processing-service.html)

## Glossary

Transducers have some jargon. Here's an attempt to cut it down:

* **Reduce:** Many languages have some version of this, but the basic notion is, "traverse a collection; for each element, pass an accumulated value and the element to a func. Use the return of that func as the accum for the next element."
* **Reducing function:** I've tried to be consistent about this in the docs, but things may get a little muddled here, and especially in the clojure discussions, between two concepts: a single function with the signature `func (accum interface{}, value interface{}) (result interface{}, terminate bool)`, and an interface with a method named `Reduce` having that signature, a 1-arity function named `Complete`, and a 0-arity function named `Init`. In this lib, these are the func type `Reducer` and interface type `ReduceStep`, respectively. This split reflects the fact that while `Reduce()` is the main concept this whole framework is built around, the other two methods must come along with to make things work.
* **Transducer:** A function that *transforms* a *reducing* function. They take a reducing func and return another: `type Transducer func(ReduceStep) ReduceStep`, wrapping their transformation on top of whatever that reducing func does.
* **Predicate:** Some transducers - for example, `Map` and `Filter` - take a function to do their work. These injected functions are referred to as predicates.
* **Bottom reducer:** Since the Transducer signature dictates that they take and return a ReduceStep, they're cyclical. No matter how many Transducers you stack up, they must be rooted onto plain ReduceStep. That root is often referred to as the "bottom" reducer, as it sits at the bottom of the transduction stack.
* **Transducer stack:** In short: `[]Transducer`. The expectation is that a set of transducers can be assembled, and then applied over and over again via different transduction processors. This is because `[]Transducer` is a fundamentally stateless bunch of logic.
* **Processor:** On their own, transducers are inert logic. Processors take (at minimum) a collection and a transducer stack, compose a transducer pipeline from the stack, and apply it across the collection.

