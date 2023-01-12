package prime

import (
	"testing"
)

func TestPrimeIter(t *testing.T) {
	iter := NewPrimeIterator()

	want := []int{2, 3, 5, 7, 11, 13, 17, 19}
	for i := 0; i < len(want); i++ {
		equalsThat(iter.Next(), want[i], t)
	}
}

func TestPrimeIterFrom(t *testing.T) {
	iter := NewPrimeIteratorFrom(1000000000)
	equalsThat(0, iter.Next(), t)
	equalsThat(0, iter.Next(), t)
	equalsThat(0, iter.Next(), t)
}

func BenchmarkPrimeIter(b *testing.B) {
	iter := NewPrimeIterator()
	
	for i := 0; i < b.N; i++ {
		iter.Next()
	}
}