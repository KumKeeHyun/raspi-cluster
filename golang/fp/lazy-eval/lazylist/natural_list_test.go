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
		return a % 2 == 0
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
		return a % 2 == 0
	}).Map(func(a int) int {
		return a + 1
	}).ReduceNItem(5, 0, func(a, b int) int {
		return a + b
	})
	want := 35
	
	equalsThat(got, want, t)
}
