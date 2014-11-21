package transduce

import (
	"fmt"
	"math/rand"
)

// The master signature: a reducing step function.
type Reducer func(accum interface{}, value interface{}) (result interface{})

// Transducers are an interface, but...
type Transducer interface {
	Transduce(ReduceStep) ReduceStep
}

type TransducerFunc func(ReduceStep) ReduceStep

func (f TransducerFunc) Transduce(r ReduceStep) ReduceStep {
	return f(r)
}

type ReduceStep interface {
	// The primary reducing step function, called during normal operation.
	Reduce(accum interface{}, value interface{}) (result interface{})
	// Complete is called when the input has been exhausted; stateful transducers
	// should flush any held state (e.g. values awaiting a full chunk) through here.
	Complete(accum interface{}) (result interface{})
}

// We also provide an easy way to express transducers as pure functions. Sig is
// still just focused on the Reducer funcs, though, because the work of supporting
// supporting Complete() is pushed to the pureReducer, which
// PureFuncTransducer.Transduce() attaches.
type PureFuncTransducer func(Reducer) Reducer

func (f PureFuncTransducer) Transduce(r ReduceStep) ReduceStep {
	return pureReducer{reduce: f(r.Reduce), complete: r.Complete}
}

type pureReducer struct {
	reduce   Reducer
	complete func(accum interface{}) (result interface{})
}

func (r pureReducer) Reduce(accum interface{}, value interface{}) interface{} {
	return r.reduce(accum, value)
}

func (r pureReducer) Complete(accum interface{}) interface{} {
	// Pure functions can't have any completion work to do, so just pass along
	// TODO ...uh, right? either way, this is just a convenience
	return r.complete(accum)
}

type Mapper func(value interface{}) interface{}
type IndexedMapper func(index int, value interface{}) interface{}
type Filterer func(interface{}) bool

const dbg = true

func fml(v ...interface{}) {
	if dbg {
		fmt.Println(v)
	}
}

// Exploders transform a value of some type into a stream of values.
// No guarantees about the relationship between the type of input and output;
// output may be a collection of the input type, or may not.
type Exploder func(interface{}) ValueStream

func Sum(accum interface{}, val interface{}) (result interface{}) {
	return accum.(int) + val.(int)
}

func sum(vs ValueStream) (total int) {
	vs.Each(func(value interface{}) {
		fml("SUM: total", total)
		total += value.(int)
	})
	fml("SUM: final total", total)

	return
}

// Basic Mapper function (increments by 1)
func inc(value interface{}) interface{} {
	return value.(int) + 1
}

// Basic Filterer function (true if even)
func even(value interface{}) bool {
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
		return MakeReduce(v)
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
	return MakeReduce(t_range(limit.(int)))
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
	case []interface{}:
		return ValueSlice(c).AsStream()
	default:
		panic("not supported...yet")
	}
}

func Identity(accum interface{}, value interface{}) interface{} {
	return value
}

func Seq(vs ValueStream, init []int, tlist ...Transducer) []int {
	fml(tlist)
	// Final reducing func - append to slice
	// TODO really awkward patching this together like this - refactor it out smartly
	var t ReduceStep = pureReducer{reduce: Append(Identity), complete: func(accum interface{}) interface{} {
		return accum
	}}

	// Walk backwards through transducer list to assemble in correct order.
	// Clojure folk refer to this as applying transducers to this job.
	for i := len(tlist) - 1; i >= 0; i-- {
		fml(tlist[i])
		t = tlist[i].Transduce(t)
	}

	var ret interface{} = init

	for v, done := vs(); !done; v, done = vs() {
		fml("SEQ: Main loop:", v)
		// weird that we do nothing here
		ret = t.Reduce(ret, v)
	}

	ret = t.Complete(ret)

	return ret.([]int)
}

func Map(f Mapper) PureFuncTransducer {
	return func(r Reducer) Reducer {
		return func(accum interface{}, value interface{}) interface{} {
			fml("MAP: accum is", accum, "value is", value)
			return r(accum, f(value).(int))
		}
	}
}

func Filter(f Filterer) PureFuncTransducer {
	return func(r Reducer) Reducer {
		return func(accum interface{}, value interface{}) interface{} {
			fml("FILTER: accum is", accum, "value is", value)
			if f(value) {
				return r(accum, value)
			} else {
				return accum
			}
		}
	}
}

func Append(r Reducer) Reducer {
	return func(accum interface{}, value interface{}) interface{} {
		fml("APPEND: Appending", value, "onto", accum)
		switch v := r(accum, value).(type) {
		case []int:
			return append(accum.([]int), v...)
		case int:
			return append(accum.([]int), v)
		case ValueStream:
			flattenValueStream(v).Each(func(value interface{}) {
				fml("APPEND: *actually* appending ", value, "onto", accum)
				accum = append(accum.([]int), value.(int))
			})
			return accum
		default:
			panic("not supported")
		}
	}
}

// Mapcat first runs an exploder, then 'concats' results by
// passing each individual value along to the next transducer
// in the stack.
func Mapcat(f Exploder) PureFuncTransducer {
	return func(r Reducer) Reducer {
		return func(accum interface{}, value interface{}) interface{} {
			fml("MAPCAT: Processing explode val:", value)
			stream := f(value)

			var v interface{}
			var done bool

			for { // <-- the *loop* is the 'cat'
				v, done = stream()
				if done {
					break
				}
				fml("MAPCAT: Calling next t on val:", v, "accum is:", accum)

				accum = r(accum, v)
			}

			return accum
		}
	}
}

// Dedupe is a particular type of filter, but its statefulness
// means we need to treat it differently and can't reuse Filter
//
// TODO any reason to move this to its own struct?
func Dedupe() PureFuncTransducer {
	// Statefulness is encapsulated in the transducer function - when
	// a materializing function calls the transducer, it produces a
	// fresh state that lives only as long as that run.
	return func(r Reducer) Reducer {
		// TODO Slice is fine for prototype, but should replace with
		// type-appropriate search tree later
		seen := make([]interface{}, 0)
		return func(accum interface{}, value interface{}) interface{} {
			for _, v := range seen {
				if value == v {
					return accum
				}
			}

			seen = append(seen, value)
			return r(accum, value)
		}
	}
}

// Condense the traversed collection by partitioning it into
// chunks of []interface{} of the given length.
//
// Here's one place we sorely feel the lack of algebraic types.
func Chunk(length int) TransducerFunc {
	if length < 1 {
		panic("chunks must be at least one element in size")
	}

	return func(r ReduceStep) ReduceStep {
		// TODO look into most memory-savvy ways of doing this
		return &chunk{length: length, coll: make(ValueSlice, length, length), next: r}
	}
}

// Chunk is stateful, so it's handled with a struct instead of a pure function
type chunk struct {
	length int
	count  int
	coll   ValueSlice
	next   ReduceStep
}

func (t *chunk) Reduce(accum interface{}, value interface{}) interface{} {
	fml("CHUNK: Chunk count: ", t.count, "coll contents: ", t.coll)
	t.coll[t.count] = value
	t.count++ // TODO atomic

	if t.count == t.length {
		t.count = 0
		newcoll := make(ValueSlice, t.length, t.length)
		copy(newcoll, t.coll)
		fml("CHUNK: passing val to next td:", t.coll)
		return t.next.Reduce(accum, newcoll.AsStream())
	} else {
		return accum
	}
}

func (t *chunk) Complete(accum interface{}) interface{} {
	fml("CHUNK: Completing...")
	// if there's a partially-completed chunk, send it through reduction as-is
	if t.count != 0 {
		fml("CHUNK: Leftover values found, passing coll to next td:", t.coll[:t.count])
		// should be fine to send the original, we know we're done
		accum = t.next.Reduce(accum, t.coll[:t.count].AsStream())
	}

	return t.next.Complete(accum)
}

// Condense the traversed collection by partitioning it into chunks,
// represented by ValueStreams. A new contiguous stream is created every time
// the injected filter function returns true.
func ChunkBy(f Filterer) TransducerFunc {
	if f == nil {
		panic("cannot provide nil function pointer to ChunkBy")
	}

	return func(r ReduceStep) ReduceStep {
		// TODO look into most memory-savvy ways of doing this
		return &chunkBy{chunker: f, coll: make(ValueSlice, 0), next: r}
	}
}

type chunkBy struct {
	chunker Filterer
	coll    ValueSlice
	next    ReduceStep
}

func (t *chunkBy) Reduce(accum interface{}, value interface{}) interface{} {
	fml("CHUNKBY: Chunk size: ", len(t.coll), "coll contents: ", t.coll)
	if vs, ok := value.(ValueStream); ok {
		var vals ValueSlice
		fml("CHUNKBY: operating on ValueStream")
		// TODO this SUUUUUCKS, we have to duplicate the stream
		// TODO the fact that the logic splits like this is indicative of a deeper problem
		vs.Each(func(v interface{}) {
			vals = append(vals, v)
		})

		fml("CHUNKBY: collected vals:", vals)

		// TODO this is not chunkby...it should make a new group every time a new val comes back
		if !t.chunker(vals.AsStream()) {
			fml("CHUNKBY: chunk unfinished; appending these vals to coll:", vals)
			t.coll = append(t.coll, vals.AsStream())
		} else {
			fml("CHUNKBY: passing value streams to next td:", t.coll)
			accum = t.next.Reduce(accum, t.coll.AsStream())
			t.coll = nil
			t.coll = append(t.coll, vals.AsStream())
		}
	} else {
		fml("CHUNKBY: operating on non-ValueStream")
		if !t.chunker(value) {
			fml("CHUNKBY: chunk unfinished; appending this val to coll:", value)
			t.coll = append(t.coll, value)
		} else {
			fml("CHUNKBY: passing coll to next td:", t.coll)
			accum = t.next.Reduce(accum, t.coll.AsStream())
			t.coll = nil // TODO erm...correct way to zero out a slice?
			t.coll = append(t.coll, value)
		}
	}

	return accum
}

func (t *chunkBy) Complete(accum interface{}) interface{} {
	fml("CHUNKBY: Completing...")
	// if there's a partially-completed chunk, send it through reduction as-is
	if len(t.coll) != 0 {
		fml("CHUNKBY: Leftover values found, passing coll to next td:", t.coll)
		accum = t.next.Reduce(accum, t.coll.AsStream())
	}

	return t.next.Complete(accum)
}

// Passes the received value along to the next transducer, with the
// given probability.
func RandomSample(ρ float64) PureFuncTransducer {
	if ρ < 0.0 || ρ > 1.0 {
		panic("ρ must be in the range [0.0,1.0].")
	}

	return Filter(func(_ interface{}) bool {
		return rand.Float64() < ρ
	})
}

// TakeNth takes every nth element to pass through it, discarding the remainder.
func TakeNth(n int) PureFuncTransducer {
	var count int

	return Filter(func(_ interface{}) bool {
		count++ // TODO atomic
		if count%n == 0 {
			return true
		}
		return false
	})
}

// Keep calls the provided mapper, then discards any nil value returned from the mapper.
func Keep(f Mapper) PureFuncTransducer {
	return func(r Reducer) Reducer {
		return func(accum interface{}, value interface{}) interface{} {
			fml("KEEP: accum is", accum, "value is", value)
			nv := f(value)
			if nv != nil {
				return r(accum, nv)
			}
			fml("KEEP: discarding nil")
			return accum
		}
	}
}

// KeepIndexed calls the provided indexed mapper, then discards any nil value
// return from the mapper.
func KeepIndexed(f IndexedMapper) PureFuncTransducer {
	return func(r Reducer) Reducer {
		var count int
		return func(accum interface{}, value interface{}) interface{} {
			fml("KEEPINDEXED: accum is", accum, "value is", value, "count is", count)
			nv := f(count, value)
			count++ // TODO atomic

			if nv != nil {
				return r(accum, nv)
			}

			fml("KEEPINDEXED: discarding nil")
			return accum
		}
	}
}

// Given a map of replacement value pairs, will replace any value moving through
// that has a key in the map with the corresponding value.
func Replace(pairs map[interface{}]interface{}) PureFuncTransducer {
	return func(r Reducer) Reducer {
		return func(accum interface{}, value interface{}) interface{} {
			if v, exists := pairs[value]; exists {
				fml("REPLACE: match found, replacing", value, "with", v)
				return r(accum, v)
			}
			fml("REPLACE: no match, passing along", value)
			return r(accum, value)
		}
	}
}
