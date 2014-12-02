# Transducers for Go

This is an implementation of transducers, a concept [from Clojure](http://clojure.org), for Go.

Transducers can be tricky to understand with just an abstract description, but here it is:

> Transducers are a means of building composable, reusable algorithmic transformations.

Transducers were introduced in Clojure for a sorta-similar reason that `range` exists in Go: they wanted one way of writing element-wise operations on channels *and* other collection structures (though that's just the tip of the iceberg).

I'm honestly not sure if I these are a good idea for Go or not. I've written this library as an exploratory experiment in their utility/applicability to Go, and would love feedback on transducers' utility for Go.

## What they are

There's a lot out there already, no need to duplicate it here. Instead, know that this library attempts to faithfully reproduce their behaviors in Go, meaning that anything about how they *ought* to work as defined elsewhere, should hold true here.

Here's some resources - mostly in Clojure, of course:

* Rich Hickey's [StrangeLoop talk](https://www.youtube.com/watch?v=6mTbuzafcII) introducing transducers (and his recent [ClojureConj talk](https://www.youtube.com/watch?v=4KqUvG8HPYo))
* The [Clojure docs](http://clojure.org/transducers) page for transducers
* [Some](https://gist.github.com/ptaoussanis/e537bd8ffdc943bbbce7) [high-level](https://bendyworks.com/transducers-clojures-next-big-idea/) [summaries](http://thecomputersarewinning.com/post/Transducers-Are-Fundamental/) of transducers
* Some [examples](http://ianrumford.github.io/blog/2014/08/08/Some-trivial-examples-of-using-Clojure-Transducers/) of [uses](http://matthiasnehlsen.com/blog/2014/10/06/Building-Systems-in-Clojure-2/) for transducers...mostly just toy stuff
* A couple [blog](http://blog.podsnap.com/ducers2.html) [posts](http://conscientiousprogrammer.com/blog/2014/08/07/understanding-cloure-transducers-through-types/) examining type issues with transducers

It's also been pointed out to me that these look a bit like [Google's Dataflow](http://googlecloudplatform.blogspot.com/2014/06/sneak-peek-google-cloud-dataflow-a-cloud-native-data-processing-service.html)

## Glossary

Transducers have some jargon. Here's an attempt to cut it down:

* **Reduce:** Many languages have some version of this, but the basic notion is, "traverse a collection; for each element, pass an accumulated value and the element to a func. Use the return of that func as the accum for the next element."
* **Reducing function:** I've tried to be consistent about this in the docs, but things may get a little muddled here, and especially in the clojure discussions, between two concepts: a single function with the signature `func (accum interface{}, value interface{}) (result interface{}, terminate bool)`, and an interface with a method named `Reduce` having that signature, a 1-arity function named `Complete`, and a 0-arity function named `Init`. In this lib, these are the func type `Reducer` and interface type `ReduceStep`, respectively. This split reflects the fact that while `Reduce()` is the main concept this whole framework is built around, the other two methods must come along with to make things work.
* **Transducer:** A function that *transforms* a *reducing* function. They take a reducing func and return another: `type Transducer func(ReduceStep) ReduceStep`, wrapping their transformation on top of whatever that reducing func does.
* **Bottom reducer:** Since the Transducer signature dictates that they take and return a ReduceStep, they're cyclical. No matter how many Transducers you stack up, they must be rooted onto plain ReduceStep. That root is often referred to as the "bottom" reducer, as it sits at the bottom of the transduction stack.
* **Transducer stack:** In short: `[]Transducer`. The expectation is that a set of transducers can be assembled, and then applied over and over again via different transduction processors. This is because `[]Transducer` is a fundamentally stateless bunch of logic.
* **Processor:** On their own, transducers are inert logic. Processors take (at minimum) a collection and a transducer stack, compose a transducer pipeline from the stack, and apply it across the collection.

## Transducers, generics, and the type system

See [#1](https://github.com/sdboyer/transducers-go/issues/1).
