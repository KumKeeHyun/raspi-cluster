package prime

import (
	"reflect"
	"testing"
)

func TestIsPrime(t *testing.T) {
	equalsThat(IsPrime(2), true, t)
	equalsThat(IsPrime(3), true, t)
	equalsThat(IsPrime(4), false, t)
	equalsThat(IsPrime(5), true, t)
	equalsThat(IsPrime(6), false, t)
	equalsThat(IsPrime(7), true, t)
	equalsThat(IsPrime(8), false, t)
	equalsThat(IsPrime(9), false, t)
	equalsThat(IsPrime(10), false, t)
	equalsThat(IsPrime(11), true, t)
}

func equalsThat(got interface{}, want interface{}, t *testing.T) {
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got : %v, want : %v", got, want)
	}
}

func BenchmarkNextPrimeFrom(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NextPrimeFrom(100000000000000000)
	}
}