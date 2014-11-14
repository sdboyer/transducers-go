package transduce

// The lowest-possible-denominator iteration concept. It's kinda
// terrible because it must inherently inspect both the current and
// next value on each call.
type ValueStream func() (value interface{}, done bool)

type Streamable interface {
	AsStream() ValueStream
}

// Wrap an iterator up into a ValueStream func.
func iteratorToValueStream(i Iterator) func() (value interface{}, done bool) {
	return func() (interface{}, bool) {
		if !i.Valid() {
			i.Done()
			return nil, true
		}

		// sad that defer has performance issues
		//
		// this approach signals termination to  the iterator when no longer valid
		//defer func() {
		//i.Next()
		//if !i.Valid() {
		//i.Done()
		//}
		//}()
		//
		// this approach just makes sure next gets called
		// defer i.Next()

		v := i.Current()
		i.Next()

		return v, false // TODO this pops and seeks...kinda weird. fix later when it hurts
	}
}

type Iterator interface {
	//Rewind()
	Current() (value interface{})
	Next()
	Valid() bool
	Done()
}

type IntSliceIterator struct {
	slice []int
	pos   int
}

func (i *IntSliceIterator) Current() interface{} {
	//fml("Current, current value:", i.slice[i.pos])
	return i.slice[i.pos]
}

func (i *IntSliceIterator) Next() {
	// atomicity
	//fml("Next, old position:", i.pos)
	i.pos++
}

func (i *IntSliceIterator) Valid() (valid bool) {
	return i.pos < len(i.slice)
}

func (i *IntSliceIterator) Done() {

}

//func (i IntSliceIterator) Rewind() {
//i.pos = 0
//}
