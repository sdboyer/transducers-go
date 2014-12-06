package transducers

type reducerBase struct {
	next Reducer
}

func (r reducerBase) Complete(accum interface{}) interface{} {
	// Pure functions inherently can't have any completion work, so flow through
	return r.next.Complete(accum)
}

func (r reducerBase) Init() interface{} {
	return r.next.Init()
}

type reducerHelper struct {
	S func(accum interface{}, value interface{}) (interface{}, bool)
	C func(accum interface{}) interface{}
	I func() interface{}
}

func (r reducerHelper) Step(accum interface{}, value interface{}) (interface{}, bool) {
	return r.S(accum, value)
}

func (r reducerHelper) Complete(accum interface{}) interface{} {
	return r.C(accum)
}

func (r reducerHelper) Init() interface{} {
	return r.I()
}

// Creates a helper struct for defining a Reducer on the fly.
//
// This is mostly useful for creating a bottom reducer with minimal fanfare.
//
// This returns an instance of bareReducer, which is a struct containing three
// function pointers, one for each of the three methods of Reducer - S for
// Step, C for Complete, I for Init. The struct implements Reducer
// by simply passing method calls along to the contained function pointers.
//
// This makes it easier to create Reducers on the fly. The first argument is a
// reducer - if you pass nil, it'll create a no-op reducer for you. If you want to
// overwrite the other two, do it on the returned struct.
func CreateStep(s ReduceStep) reducerHelper {
	if s == nil {
		s = func(accum interface{}, value interface{}) (interface{}, bool) {
			return accum, false
		}
	}
	return reducerHelper{
		S: s,
		C: func(accum interface{}) interface{} {
			return accum
		},
		I: func() interface{} {
			return make([]interface{}, 0)
		},
	}
}
