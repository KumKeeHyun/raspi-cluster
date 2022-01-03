package lazylist

type FiniteLazyList interface {
	IsEmpty() bool
	Cons(int) FiniteLazyList
	Head() int
	Tail() FiniteLazyList
	ReduceNItem(int, int, func(int, int) int) int
	ReduceAllItem(int, func(int, int) int) int
	ReduceNList(int, []int, func([]int, int) []int) []int
	ReduceAllList([]int, func([]int, int) []int) []int
	TakeN(int) []int
	TakeAll() []int
	Filter(func(int) bool) FiniteLazyList
	Map(func(int) int) FiniteLazyList
}

type finiteLazyList struct {
	l *lazyList
}

var _ FiniteLazyList = &finiteLazyList{}

func Empty() FiniteLazyList {
	l := empty()
	return &finiteLazyList{
		l: l,
	}
}

func RangeFromTo(f, t int) FiniteLazyList {
	l := rangeFromTo(f, t)
	return &finiteLazyList{
		l: l,
	}
}

func Range(n int) FiniteLazyList {
	l := rangeFunc(n)
	return &finiteLazyList{
		l: l,
	}
}

func SliceToLazyList(s []int) FiniteLazyList {
	l := sliceToLazyList(s)
	return &finiteLazyList{
		l: l,
	}
}

func (fl *finiteLazyList) IsEmpty() bool {
	return fl.l.isEmpty()
}

func (fl *finiteLazyList) Cons(item int) FiniteLazyList {
	fl.l = fl.l.cons(item)
	return fl
}

func (fl *finiteLazyList) Head() int {
	return fl.l.head()
}

func (fl *finiteLazyList) Tail() FiniteLazyList {
	fl.l = fl.l.tail()
	return fl
}

func (fl *finiteLazyList) ReduceNItem(n, acc int, f func(a, b int) int) int {
	return fl.l.reduceNItem(n, acc, f)
}

func (fl *finiteLazyList) ReduceAllItem(acc int, f func(a, b int) int) int {
	return fl.l.reduceAllItem(acc, f)
}

func (fl *finiteLazyList) ReduceNList(n int, acc []int, f func(a []int, b int) []int) []int {
	return fl.l.reduceNList(n, acc, f)
}

func (fl *finiteLazyList) ReduceAllList(acc []int, f func(a []int, b int) []int) []int {
	return fl.l.reduceAllList(acc, f)
}

func (fl *finiteLazyList) TakeN(n int) []int {
	return fl.l.takeN(n)
}

func (fl *finiteLazyList) TakeAll() []int {
	return fl.l.takeAll()
}

func (fl *finiteLazyList) Filter(f func(a int) bool) FiniteLazyList {
	fl.l = fl.l.filterFunc(f)
	return fl
}

func (fl *finiteLazyList) Map(f func(a int) int) FiniteLazyList {
	fl.l = fl.l.mapFunc(f)
	return fl
}
