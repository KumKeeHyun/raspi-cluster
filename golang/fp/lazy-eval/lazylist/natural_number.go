package lazylist

type InfinityLazyList interface {
	Head() int
	Tail() InfinityLazyList
	ReduceNItem(int, int, func(int, int) int) int
	ReduceNList(int, []int, func([]int, int) []int) []int
	TakeN(int) []int
	Filter(func(int) bool) InfinityLazyList
	Map(func(int) int) InfinityLazyList
}

type infinityLazyList struct {
	l *lazyList
}

var _ InfinityLazyList = &infinityLazyList{}

func (il *infinityLazyList) Head() int {
	return il.l.head()
}

func (il *infinityLazyList) Tail() InfinityLazyList {
	il.l = il.l.tail()
	return il
}

func (il *infinityLazyList) ReduceNItem(n int, acc int, f func(int, int) int) int {
	return il.l.reduceNItem(n, acc, f)
}

func (il *infinityLazyList) ReduceNList(n int, acc []int, f func([]int, int) []int) []int {
	return il.l.reduceNList(n, acc, f)
}

func (il *infinityLazyList) TakeN(n int) []int {
	return il.l.takeN(n)
}

func (il *infinityLazyList) Filter(f func(int) bool) InfinityLazyList {
	il.l = il.l.filterFunc(f)
	return il
}

func (il *infinityLazyList) Map(f func(int) int) InfinityLazyList {
	il.l = il.l.mapFunc(f)
	return il
}

func NaturalNumList() InfinityLazyList {
	return &infinityLazyList{
		l: naturalNumList(1),
	}
}

func naturalNumList(n int) *lazyList {
	return newLazyList(func() *value {
		return &value{
			item: n,
			nextEval: func() *value {
				return naturalNumList(n + 1).eval()
			},
		}
	})
}
