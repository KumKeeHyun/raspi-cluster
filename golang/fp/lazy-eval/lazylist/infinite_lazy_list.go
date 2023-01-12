package lazylist

type InfiniteLazyList interface {
	Head() int
	Tail() InfiniteLazyList
	ReduceNItem(int, int, func(int, int) int) int
	ReduceNList(int, []int, func([]int, int) []int) []int
	TakeN(int) []int
	Filter(func(int) bool) InfiniteLazyList
	Map(func(int) int) InfiniteLazyList
}

type infiniteLazyList struct {
	l *lazyList
}

var _ InfiniteLazyList = &infiniteLazyList{}

func (il *infiniteLazyList) Head() int {
	return il.l.head()
}

func (il *infiniteLazyList) Tail() InfiniteLazyList {
	il.l = il.l.tail()
	return il
}

func (il *infiniteLazyList) ReduceNItem(n int, acc int, f func(int, int) int) int {
	return il.l.reduceNItem(n, acc, f)
}

func (il *infiniteLazyList) ReduceNList(n int, acc []int, f func([]int, int) []int) []int {
	return il.l.reduceNList(n, acc, f)
}

func (il *infiniteLazyList) TakeN(n int) []int {
	return il.l.takeN(n)
}

func (il *infiniteLazyList) Filter(f func(int) bool) InfiniteLazyList {
	il.l = il.l.filterFunc(f)
	return il
}

func (il *infiniteLazyList) Map(f func(int) int) InfiniteLazyList {
	il.l = il.l.mapFunc(f)
	return il
}