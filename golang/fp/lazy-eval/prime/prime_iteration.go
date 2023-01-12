package prime

type Iterator interface {
	HasNext() bool
	Next() int
}

type primeIterator struct {
	lastPrime int
}

func NewPrimeIterator() Iterator {
	return &primeIterator{1}
}

func NewPrimeIteratorFrom(num int) Iterator {
	return &primeIterator{num}
}

func (pi *primeIterator) HasNext() bool{
	return true
}

func (pi *primeIterator) Next() int{
	pi.lastPrime = NextPrimeFrom(pi.lastPrime)
	return pi.lastPrime
}

