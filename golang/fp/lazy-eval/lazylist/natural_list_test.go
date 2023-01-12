package lazylist

import (
	"testing"
)

func TestNaturalNumber(t *testing.T) {
	list := NaturalList()

	got := list.TakeN(5)
	want := []int{1, 2, 3, 4, 5}

	equalsThat(got, want, t)
}

func TestNaturalNumberFilter(t *testing.T) {
	list := NaturalList()

	got := list.Filter(func(a int) bool {
		return a%2 == 0
	}).TakeN(5)
	want := []int{2, 4, 6, 8, 10}

	equalsThat(got, want, t)
}

func TestNaturalNumberMap(t *testing.T) {
	list := NaturalList()

	got := list.Map(func(a int) int {
		return a + 10
	}).TakeN(5)
	want := []int{11, 12, 13, 14, 15}

	equalsThat(got, want, t)
}

func TestNaturalNumberFilterMapReduce(t *testing.T) {
	list := NaturalList()

	got := list.Filter(func(a int) bool {
		return a%2 == 0
	}).Map(func(a int) int {
		return a + 1
	}).ReduceNItem(5, 0, func(a, b int) int {
		return a + b
	})
	want := 35

	equalsThat(got, want, t)
}

func BenchmarkLazy(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	_ = NaturalList().Map(func(i int) int {
		if i%100 != 0 {
			return i
		} else {
			return i * 100
		}
	}).Map(func(i int) int {
		if i%3 == 0 {
			return i * 2
		} else {
			return i * 3
		}
	}).TakeN(100000)
}
func BenchmarkNonLazy(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	list := make([]int, 0, 100000)
	for i := 1; i <= 100000; i++ {
		list = append(list, i)
	}

	m1 := mapFunc(list, func(i int) int {
		if i%100 != 0 {
			return i
		} else {
			return i * 100
		}
	})
	_ = mapFunc(m1, func(i int) int {
		if i%3 == 0 {
			return i * 2
		} else {
			return i * 3
		}
	})

}

func mapFunc(l []int, f func(int) int) []int {
	r := make([]int, len(l))
	for i, v := range l {
		r[i] = f(v)
	}
	return r
}
