package lazylist

import (
	"reflect"
	"testing"
)

func TestCons(t *testing.T) {
	list := Empty().Cons(1).Cons(2)
	
	equalsThat(list.Head(), 2, t)
	equalsThat(list.Tail().Head(), 1, t)

	emptyList := list.Tail().Tail()
	equalsThat(emptyList.IsEmpty(), true, t)
}

func equalsThat(got interface{}, want interface{}, t *testing.T) {
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got : %v, want : %v", got, want)
	}
}

func TestRange(t *testing.T) {
	list := Range(4)

	equalsThat(list.Head(), 0, t)
	equalsThat(list.Tail().Head(), 1, t)
	equalsThat(list.Tail().Tail().Head(), 2, t)
	equalsThat(list.Tail().Tail().Tail().Head(), 3, t)
	equalsThat(list.Tail().Tail().Tail().Tail().IsEmpty(), true, t)

	emptyList := Range(0)
	equalsThat(emptyList.IsEmpty(), true, t)
	
	emptyList = Range(-1)
	equalsThat(emptyList.IsEmpty(), true, t)
}

func TestRangeFromTo(t *testing.T) {
	list := RangeFromTo(1, 5)

	equalsThat(list.Head(), 1, t)
	equalsThat(list.Tail().Head(), 2, t)
	equalsThat(list.Tail().Tail().Head(), 3, t)
	equalsThat(list.Tail().Tail().Tail().Head(), 4, t)
	equalsThat(list.Tail().Tail().Tail().Tail().IsEmpty(), true, t)

	emptyList := RangeFromTo(1, 1)
	equalsThat(emptyList.IsEmpty(), true, t)

	emptyList = RangeFromTo(5, 1)
	equalsThat(emptyList.IsEmpty(), true, t)
}

func TestReduceAll(t *testing.T) {
	list := RangeFromTo(1, 11)

	got := list.ReduceAllItem(0, func(a, b int) int {
		return a + b
	})
	want := 55

	equalsThat(got, want, t)
}

func TestTakeAll(t *testing.T) {
	list := RangeFromTo(0, 3)

	got := list.TakeAll()
	want := []int{0, 1, 2}

	equalsThat(got, want, t)
}

func TestFilter(t *testing.T) {
	list := RangeFromTo(1, 11)

	got := list.Filter(func(a int) bool {
		return a % 2 == 0
	}).TakeAll()
	want := []int{2, 4, 6, 8, 10}

	equalsThat(got, want, t)
}

func TestMap(t *testing.T) {
	list := RangeFromTo(1, 6)

	got := list.Map(func(a int) int {
		return a * 10
	}).TakeAll()
	want := []int{10, 20, 30, 40, 50}

	equalsThat(got, want, t)
}

func TestFilterMap(t *testing.T) {
	list := RangeFromTo(1, 11)
	
	got := list.Filter(func(a int) bool {
		return a % 2 == 0
	}).Map(func(a int) int {
		return a * 10
	}).TakeAll()
	want := []int{20, 40, 60, 80, 100}

	equalsThat(got, want, t)
}

func TestLazyEval(t *testing.T) {
	list := RangeFromTo(1, 11)

	got := list.Filter(func(a int) bool {
		return a % 2 == 0
	}).Map(func(a int) int {
		return a * 10
	}).Head()
	want := 20

	equalsThat(got, want, t)
}