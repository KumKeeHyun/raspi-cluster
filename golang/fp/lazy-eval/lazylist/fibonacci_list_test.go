package lazylist

import (
	"testing"
)

func TestFibonacciList(t *testing.T) {
	list := FibonacciList()
	
	got := list.TakeN(10)
	want := []int{1, 1, 2, 3, 5, 8, 13, 21, 34, 55}

	equalsThat(got, want, t)
}
