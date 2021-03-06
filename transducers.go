package transducers

import "math/rand"

// The master signature: a reducing step function.
type ReduceStep func(accum interface{}, value interface{}) (result interface{}, terminate bool)

// A transducer transforms a reducing function into a new reducing function.
type Transducer func(Reducer) Reducer

// This is separated out because, while Transducers must support Init() (in order to pass
// the call along), not all processors require Init, so they may take a partial step instead.
type Reducer interface {
	// The primary reducing step function, called during normal operation.
	Step(accum interface{}, value interface{}) (result interface{}, terminate bool) // Reducer

	// Complete is called when the input has been exhausted; stateful transducers
	// should flush any held state (e.g. values awaiting a full chunk) through here.
	Complete(accum interface{}) (result interface{})

	// Certain processors will call this to get an initial value for the accumulator.
	// TODO maybe should split this out to separate interface
	Init() interface{}
}

// Creates a transduction pipeline from a reducing function and a stack of transducers.
//
// Creating the pipeline means that state in the transducers has been initialized.
// In other words, while you can create as many pipelines as you want from a
// a stack of transducers, you have to create a new pipeline for each
// process/collection you run.
//
// This function is usually called by processors (Transduce, Eduction, etc.).
// If you're using one of those, they'll call it when the time is right.
func CreatePipeline(r Reducer, tds ...Transducer) (rs Reducer) {
	rs = Reducer(r)
	// Because a pipeline is a series of wrapped functions, we must walk the list
	// in reverse order and apply each transducer, starting from the bottom reducer.
	for i := len(tds) - 1; i >= 0; i-- {
		rs = tds[i](rs)
	}

	return
}

func (r ReduceStep) Step(accum interface{}, value interface{}) (result interface{}, terminate bool) {
	return r(accum, value)
}

func (r ReduceStep) Complete(accum interface{}) (result interface{}) {
	return accum
}

// TODO binding this to the general function type is maybe not the best idea, but it's simple
// and works for now
func (r ReduceStep) Init() interface{} {
	return make([]interface{}, 0)
}

/* Transducer implementations */

type map_r struct {
	reducerBase
	f Mapper
}

func (r map_r) Step(accum interface{}, value interface{}) (interface{}, bool) {
	return r.next.Step(accum, r.f(value))
}

// Map calls its predicate once for each value coming through, passing the
// result along to the next step.
func Map(f Mapper) Transducer {
	return func(r Reducer) Reducer {
		return map_r{reducerBase{r}, f}
	}
}

type filter struct {
	reducerBase
	f Filterer
}

func (r filter) Step(accum interface{}, value interface{}) (interface{}, bool) {
	var check bool
	if vs, ok := value.(ValueStream); ok {
		vs, value = vs.Split()
		check = r.f(vs)
	} else {
		check = r.f(value)
	}

	if check {
		return r.next.Step(accum, value)
	} else {
		return accum, false
	}
}

// Returns a Filter transducer. Filters call their predicate function for
// each incoming value, dropping the value if the predicate returns false
// and passing it along if it returns true.
func Filter(f Filterer) Transducer {
	return func(r Reducer) Reducer {
		return filter{reducerBase{r}, f}
	}
}

// for append operations at the bottom of a transducer stack
type append_bottom struct{}

func (r append_bottom) Step(accum interface{}, value interface{}) (interface{}, bool) {
	switch v := value.(type) {
	case []int:
		return append(accum.([]int), v...), false
	case int:
		return append(accum.([]int), v), false
	case ValueStream:
		v.Flatten().Each(func(value interface{}) {
			accum = append(accum.([]int), value.(int))
		})
		return accum, false
	default:
		panic("not supported")
	}
}

func (r append_bottom) Complete(accum interface{}) interface{} {
	return accum
}

func (r append_bottom) Init() interface{} {
	return make([]int, 0)
}

func Append() Reducer {
	return append_bottom{}
}

type mapcat struct {
	reducerBase
	f Exploder
}

func (r mapcat) Step(accum interface{}, value interface{}) (interface{}, bool) {
	stream := r.f(value)

	var v interface{}
	var done, terminate bool

	for { // <-- the *loop* is the 'cat'
		v, done = stream()
		if done {
			break
		}

		accum, terminate = r.next.Step(accum, v)
		if terminate {
			break
		}
	}

	return accum, terminate
}

// Mapcat first runs an exploder, then 'concats' results by passing each
// individual value along to the next transducer in the pipeline.
func Mapcat(f Exploder) Transducer {
	return func(r Reducer) Reducer {
		return mapcat{reducerBase{r}, f}
	}
}

type dedupe struct {
	reducerBase
	// TODO Slice is fine for prototype, but should replace with type-appropriate
	// search tree later
	seen valueSlice
}

func (r *dedupe) Step(accum interface{}, value interface{}) (interface{}, bool) {
	for _, v := range r.seen {
		if value == v {
			return accum, false
		}
	}

	r.seen = append(r.seen, value)
	return r.next.Step(accum, value)
}

// Dedupe keeps track of values that have passed through it during this
// transduction process and drops any duplicates.
//
// Simple equality (==) is used for comparison. That will panic on non-hashable
// datastructures (maps, slices, channels)!
func Dedupe() Transducer {
	return func(r Reducer) Reducer {
		return &dedupe{reducerBase{r}, make([]interface{}, 0)}
	}
}

// Condense the traversed collection by partitioning it into
// chunks of []interface{} of the given length.
//
// Here's one place we sorely feel the lack of algebraic types.
func Chunk(length int) Transducer {
	if length < 1 {
		panic("chunks must be at least one element in size")
	}

	return func(r Reducer) Reducer {
		// TODO look into most memory-savvy ways of doing this
		return &chunk{length: length, coll: make(valueSlice, length, length), next: r}
	}
}

// Chunk is stateful, so it's handled with a struct instead of a pure function
type chunk struct {
	length    int
	count     int
	terminate bool
	coll      valueSlice
	next      Reducer
}

func (t *chunk) Step(accum interface{}, value interface{}) (interface{}, bool) {
	t.coll[t.count] = value
	t.count++ // TODO atomic

	if t.count == t.length {
		t.count = 0
		newcoll := make(valueSlice, t.length, t.length)
		copy(newcoll, t.coll)
		accum, t.terminate = t.next.Step(accum, newcoll.AsStream())
		return accum, t.terminate
	} else {
		return accum, false
	}
}

func (t *chunk) Complete(accum interface{}) interface{} {
	// if there's a partially-completed chunk, send it through reduction as-is
	if t.count != 0 && !t.terminate {
		// should be fine to send the original, we know we're done
		accum, t.terminate = t.next.Step(accum, t.coll[:t.count].AsStream())
	}

	return t.next.Complete(accum)
}

func (t *chunk) Init() interface{} {
	return t.next.Init()
}

// Condense the traversed collection by partitioning it into chunks,
// represented by ValueStreams. A new contiguous stream is created every time
// the injected filter function returns a different value from the previous.
func ChunkBy(f Mapper) Transducer {
	return func(r Reducer) Reducer {
		// TODO look into most memory-savvy ways of doing this
		return &chunkBy{chunker: f, coll: make(valueSlice, 0), next: r, first: true, last: nil}
	}
}

type chunkBy struct {
	chunker   Mapper
	first     bool
	last      interface{}
	coll      valueSlice
	next      Reducer
	terminate bool
}

func (t *chunkBy) Step(accum interface{}, value interface{}) (interface{}, bool) {
	var chunkval interface{}
	if vs, ok := value.(ValueStream); ok {
		vs, value = vs.Split()
		chunkval = t.chunker(vs)
	} else {
		chunkval = t.chunker(value)
	}

	if t.first { // nothing to compare against if first pass
		t.first = false
		t.last = chunkval
		t.coll = append(t.coll, value)
	} else if t.last == chunkval {
		t.coll = append(t.coll, value)
	} else {
		t.last = chunkval
		accum, t.terminate = t.next.Step(accum, t.coll.AsStream())
		t.coll = nil
		t.coll = append(t.coll, value)
	}
	return accum, t.terminate
}

func (t *chunkBy) Complete(accum interface{}) interface{} {
	// if there's a partially-completed chunk, send it through reduction as-is
	if len(t.coll) != 0 && !t.terminate {
		accum, t.terminate = t.next.Step(accum, t.coll.AsStream())
	}

	return t.next.Complete(accum)
}

func (t *chunkBy) Init() interface{} {
	return t.next.Init()
}

type randomSample struct {
	filter
}

// Passes the received value along to the next transducer, with the
// given probability.
func RandomSample(ρ float64) Transducer {
	if ρ < 0.0 || ρ > 1.0 {
		panic("ρ must be in the range [0.0,1.0].")
	}

	return func(r Reducer) Reducer {
		return randomSample{filter{reducerBase{r}, func(_ interface{}) bool {
			//panic("oh shit")
			return rand.Float64() < ρ
		}}}
	}
}

type takeNth struct {
	filter
}

// TakeNth takes every nth element to pass through it, discarding the remainder.
func TakeNth(n int) Transducer {
	var count int

	return func(r Reducer) Reducer {
		return takeNth{filter{reducerBase{r}, func(_ interface{}) bool {
			count++ // TODO atomic
			return count%n == 0
		}}}
	}
}

type keep struct {
	reducerBase
	f Mapper
}

func (r keep) Step(accum interface{}, value interface{}) (interface{}, bool) {
	nv := r.f(value)
	if nv != nil {
		return r.next.Step(accum, nv)
	}
	return accum, false
}

// Keep calls the provided mapper, then discards any nil value returned from the mapper.
func Keep(f Mapper) Transducer {
	return func(r Reducer) Reducer {
		return keep{reducerBase{r}, f}
	}
}

type keepIndexed struct {
	reducerBase
	count int
	f     IndexedMapper
}

func (r *keepIndexed) Step(accum interface{}, value interface{}) (interface{}, bool) {
	nv := r.f(r.count, value)
	r.count++ // TODO atomic

	if nv != nil {
		return r.next.Step(accum, nv)
	}

	return accum, false

}

// KeepIndexed calls the provided indexed mapper, then discards any nil value
// return from the mapper.
func KeepIndexed(f IndexedMapper) Transducer {
	return func(r Reducer) Reducer {
		return &keepIndexed{reducerBase{r}, 0, f}
	}
}

type replace struct {
	reducerBase
	pairs map[interface{}]interface{}
}

func (r replace) Step(accum interface{}, value interface{}) (interface{}, bool) {
	if v, exists := r.pairs[value]; exists {
		return r.next.Step(accum, v)
	}
	return r.next.Step(accum, value)
}

// Given a map of replacement value pairs, will replace any value moving through
// that has a key in the map with the corresponding value.
func Replace(pairs map[interface{}]interface{}) Transducer {
	return func(r Reducer) Reducer {
		return replace{reducerBase{r}, pairs}
	}
}

type take struct {
	reducerBase
	max   uint
	count uint
}

func (r *take) Step(accum interface{}, value interface{}) (interface{}, bool) {
	r.count++ // TODO atomic
	if r.count < r.max {
		return r.next.Step(accum, value)
	}
	// should NEVER be called again after this. add a panic branch?
	accum, _ = r.next.Step(accum, value)
	return accum, true
}

// Take specifies a maximum number of values to receive, after which it will
// terminate the transducing process.
func Take(max uint) Transducer {
	return func(r Reducer) Reducer {
		return &take{reducerBase{r}, max, 0}
	}
}

type takeWhile struct {
	reducerBase
	f Filterer
}

func (r takeWhile) Step(accum interface{}, value interface{}) (interface{}, bool) {
	if !r.f(value) {
		return accum, true
	}
	return r.next.Step(accum, value)
}

// TakeWhile accepts values until the injected filterer function returns false.
func TakeWhile(f Filterer) Transducer {
	return func(r Reducer) Reducer {
		return takeWhile{reducerBase{r}, f}
	}
}

type drop struct {
	reducerBase
	min   uint
	count uint
}

func (r *drop) Step(accum interface{}, value interface{}) (interface{}, bool) {
	if r.count < r.min {
		// Increment inside so no mutation after threshold is met
		r.count++ // TODO atomic
		return accum, false
	}
	// should NEVER be called again after this. add a panic branch?
	return r.next.Step(accum, value)
}

// Drop specifies a number of values to initially ignore, after which it will
// let everything through unchanged.
func Drop(min uint) Transducer {
	return func(r Reducer) Reducer {
		return &drop{reducerBase{r}, min, 0}
	}
}

type dropWhile struct {
	reducerBase
	f        Filterer
	accepted bool
}

func (r *dropWhile) Step(accum interface{}, value interface{}) (interface{}, bool) {
	if !r.accepted {
		if !r.f(value) {
			r.accepted = true
		} else {
			return accum, false
		}
	}
	return r.next.Step(accum, value)
}

// DropWhile drops values until the injected filterer function returns false.
func DropWhile(f Filterer) Transducer {
	return func(r Reducer) Reducer {
		return &dropWhile{reducerBase{r}, f, false}
	}
}

type remove struct {
	filter
}

// Remove drops items when the injected filterer function returns true.
//
// It is the inverse of Filter.
func Remove(f Filterer) Transducer {
	return func(r Reducer) Reducer {
		return remove{filter{reducerBase{r}, func(value interface{}) bool {
			return !f(value)
		}}}
	}
}

type escape struct {
	reducerBase
	f   Filterer
	c   chan<- interface{}
	coc bool
}

func (r escape) Step(accum interface{}, value interface{}) (interface{}, bool) {
	var check bool
	if vs, ok := value.(ValueStream); ok {
		vs, value = vs.Split()
		check = r.f(vs)
	} else {
		check = r.f(value)
	}

	if check {
		r.c <- value
		return accum, false
	} else {
		return r.next.Step(accum, value)
	}
}

func (r escape) Complete(accum interface{}) interface{} {
	if r.coc {
		close(r.c)
	}
	return r.next.Complete(accum)
}

// A Escape transducer takes a filter func and a send-only value channel. If
// the filtering func returns true, it allows the value to 'escape' from the
// current transduction process and into the channel - which itself may be,
// though is not necessarily, the entry point to a transducing process itself.
// If the filtering func returns false, then the value is passed along to the
// next reducing step unchanged.
//
// Obviously, this transducer has side effects.
//
// The third parameter governs whether the passed channel is closed when the
// Escape reduce step's Complete() method is called (which occurs when the
// transducing process this is involved is complete). This is very useful for
// auto-cleanup, but could cause panics (send on closed channel) if the channel
// is being sent to from elsewhere. Be cognizant.
func Escape(f Filterer, c chan<- interface{}, closeOnComplete bool) Transducer {
	return func(r Reducer) Reducer {
		return escape{reducerBase{r}, f, c, closeOnComplete}
	}
}
