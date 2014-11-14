package transduce

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
