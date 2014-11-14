package transduce

// The lowest-possible-denominator iteration concept
type ValueStream func() (value interface{}, done bool)

type Streamable interface {
	AsStream() ValueStream
}

// Wrap an iterator up into a ValueStream func.
func iteratorToValueStream(i Iterator) func() (value interface{}, done bool) {
	return func() (interface{}, bool) {
		if !i.Valid() {
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

		return v, i.Valid() // TODO this pops and seeks...kinda weird. fix later when it hurts
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

func (i IntSliceIterator) Current() interface{} {
	return i.slice[i.pos]
}

func (i IntSliceIterator) Next() {
	// atomicity
	i.pos++
}

func (i IntSliceIterator) Valid() (valid bool) {
	return i.pos < len(i.slice)
}

func (i IntSliceIterator) Done() {

}

//func (i IntSliceIterator) Rewind() {
//i.pos = 0
//}
