package lazylist

type evaluatedList struct {
	item     int
	nextEval evalFunc
}

type evalFunc func() *evaluatedList

type lazyList struct {
	eval evalFunc
}

func newLazyList(list evalFunc) *lazyList {
	return &lazyList{
		eval: list,
	}
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

func rangeFromTo(f, t int) *lazyList {
	return rangeFunc(t - f).mapFunc(func(a int) int {
		return a + f
	})
}

func sliceToLazyList(s []int) *lazyList {
	res := empty()
	if len(s) == 0 {
		return res
	}
	for i := len(s) - 1; i >= 0; i-- {
		res = res.cons(s[i])
	}
	return res
}

func empty() *lazyList {
	return newLazyList(func() *evaluatedList {
		return nil
	})
}

func (l *lazyList) isEmpty() bool {
	return l.eval() == nil
}

func (l *lazyList) cons(item int) *lazyList {
	return &lazyList{
		eval: func() *evaluatedList {
			return &evaluatedList{
				item:     item,
				nextEval: l.eval,
			}
		},
	}
}

func (l *lazyList) head() int {
	if l.isEmpty() {
		return 0
	}
	return l.eval().item
}

func (l *lazyList) tail() *lazyList {
	if l.isEmpty() {
		return l
	}
	return newLazyList(l.eval().nextEval)
}

func (l *lazyList) reduceNItem(n, acc int, f func(a, b int) int) int {
	if l.isEmpty() || n == 0 {
		return acc
	}
	return l.tail().reduceNItem(n-1, f(acc, l.head()), f)
}

func (l *lazyList) reduceAllItem(acc int, f func(a, b int) int) int {
	if l.isEmpty() {
		return acc
	}
	return l.tail().reduceAllItem(f(acc, l.head()), f)
}

func (l *lazyList) reduceNList(n int, acc []int, f func(a []int, b int) []int) []int {
	if l.isEmpty() || n == 0 {
		return acc
	}
	return l.tail().reduceNList(n-1, f(acc, l.head()), f)
}

func (l *lazyList) reduceAllList(acc []int, f func(a []int, b int) []int) []int {
	if l.isEmpty() {
		return acc
	}
	return l.tail().reduceAllList(f(acc, l.head()), f)
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
	if l.isEmpty() {
		return l
	}

	if f(l.head()) {
		return newLazyList(func() *evaluatedList {
			return &evaluatedList{
				item: l.head(),
				nextEval: func() *evaluatedList {
					return l.tail().filterFunc(f).eval()
				},
			}
		})
	} else {
		return l.tail().filterFunc(f)
	}
}

func (l *lazyList) mapFunc(f func(a int) int) *lazyList {
	if l.isEmpty() {
		return l
	}
	return newLazyList(func() *evaluatedList {
		return &evaluatedList{
			item: f(l.head()),
			nextEval: func() *evaluatedList {
				return l.tail().mapFunc(f).eval()
			},
		}
	})
}
