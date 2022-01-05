package lazylist

import (
	"reflect"
	"testing"
)

func TestCons(t *testing.T) {
	list := empty().cons(1).cons(2)
	
	equalsThat(list.head(), 2, t)
	equalsThat(list.tail().head(), 1, t)

	emptyList := list.tail().tail()
	equalsThat(emptyList.isEmpty(), true, t)
}

func equalsThat(got interface{}, want interface{}, t *testing.T) {
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got : %v, want : %v", got, want)
	}
}

func TestRange(t *testing.T) {
	list := rangeFunc(4)

	equalsThat(list.takeAll(), []int{0, 1, 2, 3}, t)

	emptyList := rangeFunc(0)
	equalsThat(emptyList.isEmpty(), true, t)
	
	emptyList = rangeFunc(-1)
	equalsThat(emptyList.isEmpty(), true, t)
}

func TestRangeFromTo(t *testing.T) {
	list := rangeFromTo(1, 5)

	equalsThat(list.takeAll(), []int{1, 2, 3, 4}, t)

	emptyList := rangeFromTo(1, 1)
	equalsThat(emptyList.isEmpty(), true, t)

	emptyList = rangeFromTo(5, 1)
	equalsThat(emptyList.isEmpty(), true, t)
}

func TestSliceToLazyList(t *testing.T) {
	list := sliceToLazyList([]int{10, 2, 4, 3})

	equalsThat(list.takeAll(), []int{10, 2, 4, 3}, t)
	equalsThat(list.takeN(2), []int{10, 2}, t)

	emptyList := sliceToLazyList([]int{})
	equalsThat(emptyList.isEmpty(), true, t)
}

func TestMemorization(t *testing.T) {
	list := rangeFromTo(1, 6)

	evalCnt := make(chan struct{}, 10)
	list = list.mapFunc(func(a int) int {
		evalCnt <- struct{}{}
		return a + 5
	})
	equalsThat(list.head(), 6, t)
	equalsThat(list.head(), 6, t)
	equalsThat(list.head(), 6, t)
	// 3번 평가해도 내부에선 한번만 평가하고 2번은 캐시된 값 반환
	equalsThat(len(evalCnt), 1, t)

	list.takeAll()
	// 리스트 전체(5개) 평가했다면 총 5번만 연산
	equalsThat(len(evalCnt), 5, t)

	close(evalCnt)
}

func TestReduceAll(t *testing.T) {
	list := rangeFromTo(1, 11)

	got := list.reduceAllItem(0, func(a, b int) int {
		return a + b
	})
	want := 55

	equalsThat(got, want, t)
}

func TestTakeAll(t *testing.T) {
	list := rangeFromTo(0, 3)

	got := list.takeAll()
	want := []int{0, 1, 2}

	equalsThat(got, want, t)
}

func TestFilter(t *testing.T) {
	list := rangeFromTo(1, 11)

	got := list.filterFunc(func(a int) bool {
		return a % 2 == 0
	}).takeAll()
	want := []int{2, 4, 6, 8, 10}

	equalsThat(got, want, t)
}

func TestMap(t *testing.T) {
	list := rangeFromTo(1, 6)

	got := list.mapFunc(func(a int) int {
		return a * 10
	}).takeAll()
	want := []int{10, 20, 30, 40, 50}

	equalsThat(got, want, t)
}

func TestFilterMap(t *testing.T) {
	list := rangeFromTo(1, 11)
	
	got := list.filterFunc(func(a int) bool {
		return a % 2 == 0
	}).mapFunc(func(a int) int {
		return a * 10
	}).takeAll()
	want := []int{20, 40, 60, 80, 100}

	equalsThat(got, want, t)
}

func TestLazyEval(t *testing.T) {
	list := rangeFromTo(1, 11)

	got := list.filterFunc(func(a int) bool {
		return a % 2 == 0
	}).mapFunc(func(a int) int {
		return a * 10
	}).head()
	want := 20

	equalsThat(got, want, t)
}