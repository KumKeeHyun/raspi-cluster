package lazylist

type LazyList interface {
	IsEmpty() bool
	Cons(int) LazyList
	Head() int
	Tail() LazyList
	ReduceNItem(int, int, func(int, int) int) int
	ReduceAllItem(int, func(int, int) int) int
	ReduceNList(int, []int, func([]int, int) []int) []int
	ReduceAllList([]int, func([]int, int) []int) []int
	TakeN(int) []int
	TakeAll() []int
	Filter(func(int) bool) LazyList
	Map(func(int) int) LazyList
}

var _ LazyList = &lazyList{}

func Empty() LazyList {
	return empty()
}

func RangeFromTo(f, t int) LazyList {
	return rangeFunc(t - f).mapFunc(func(a int) int {
		return a + f
	})
}

func Range(n int) LazyList {
	return rangeFunc(n)
}

func (l *lazyList) IsEmpty() bool {
	return l.isEmpty()
}

func (l *lazyList) Cons(item int) LazyList {
	return l.cons(item)
}

func (l *lazyList) Head() int {
	return l.head()
}

func (l *lazyList) Tail() LazyList {
	return l.tail()
}

func (l *lazyList) ReduceNItem(n, acc int, f func(a, b int) int) int {
	return l.reduceNItem(n, acc, f)
}

func (l *lazyList) ReduceAllItem(acc int, f func(a, b int) int) int {
	return l.reduceAllItem(acc, f)
}

func (l *lazyList) ReduceNList(n int, acc []int, f func(a []int, b int) []int) []int {
	return l.reduceNList(n, acc, f)
}

func (l *lazyList) ReduceAllList(acc []int, f func(a []int, b int) []int) []int {
	return l.reduceAllList(acc, f)
}

func (l *lazyList) TakeN(n int) []int {
	return l.takeN(n)
}

func (l *lazyList) TakeAll() []int {
	return l.takeAll()
}

func (l *lazyList) Filter(f func(a int) bool) LazyList {
	return l.filterFunc(f)
}

func (l *lazyList) Map(f func(a int) int) LazyList {
	return l.mapFunc(f)
}

type value struct {
	item     int
	nextEval evalFunc
}

type evalFunc func() *value

type lazyList struct {
	eval evalFunc
}

func newLazyList(list evalFunc) *lazyList {
	return &lazyList{
		eval: list,
	}
}

func empty() *lazyList {
	return newLazyList(func() *value {
		return nil
	})
}

func (l *lazyList) isEmpty() bool {
	return l.eval() == nil
}

func rangeFunc(n int) *lazyList {
	res := empty()
	if n < 1 {
		return res
	}
	return rangeFuncRec(res, n)
}

func rangeFuncRec(l *lazyList, n int) *lazyList {
	if n == 0 {
		return l
	}
	return rangeFuncRec(l.cons(n-1), n-1)
}

func (l *lazyList) cons(item int) *lazyList {
	return &lazyList{
		eval: func() *value {
			return &value{
				item:     item,
				nextEval: l.eval,
			}
		},
	}
}

func (l *lazyList) head() int {
	if l.IsEmpty() {
		return 0
	}
	return l.eval().item
}

func (l *lazyList) tail() *lazyList {
	if l.IsEmpty() {
		return l
	}
	return newLazyList(l.eval().nextEval)
}

func (l *lazyList) reduceNItem(n, acc int, f func(a, b int) int) int {
	if l.IsEmpty() || n == 0 {
		return acc
	}
	return l.tail().reduceNItem(n-1, f(acc, l.Head()), f)
}

func (l *lazyList) reduceAllItem(acc int, f func(a, b int) int) int {
	if l.IsEmpty() {
		return acc
	}
	return l.tail().reduceAllItem(f(acc, l.Head()), f)
}

func (l *lazyList) reduceNList(n int, acc []int, f func(a []int, b int) []int) []int {
	if l.IsEmpty() || n == 0 {
		return acc
	}
	return l.tail().reduceNList(n-1, f(acc, l.Head()), f)
}

func (l *lazyList) reduceAllList(acc []int, f func(a []int, b int) []int) []int {
	if l.IsEmpty() {
		return acc
	}
	return l.tail().reduceAllList(f(acc, l.Head()), f)
}

func (l *lazyList) takeN(n int) []int {
	return l.reduceNList(n, []int{}, func(a []int, b int) []int {
		a = append(a, b)
		return a
	})
}

func (l *lazyList) takeAll() []int {
	return l.reduceAllList([]int{}, func(a []int, b int) []int {
		a = append(a, b)
		return a
	})
}

func (l *lazyList) filterFunc(f func(a int) bool) *lazyList {
	if l.IsEmpty() {
		return l
	}

	if f(l.Head()) {
		return newLazyList(func() *value {
			return &value{
				item: l.Head(),
				nextEval: func() *value {
					return l.tail().filterFunc(f).eval()
				},
			}
		})
	} else {
		return l.tail().filterFunc(f)
	}
}

func (l *lazyList) mapFunc(f func(a int) int) *lazyList {
	if l.IsEmpty() {
		return l
	}
	return newLazyList(func() *value {
		return &value{
			item: f(l.Head()),
			nextEval: func() *value {
				return l.tail().mapFunc(f).eval()
			},
		}
	})
}
