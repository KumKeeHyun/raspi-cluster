package lazylist

func FibonacciList() InfiniteLazyList {
	return &infiniteLazyList{
		l: fibonacci(0, 1),
	}
}

func fibonacci(a, b int) *lazyList {
	return newLazyList(func() *evaluatedList {
		return &evaluatedList{
			item: b,
			nextEval: func() *evaluatedList {
				return fibonacci(b, a+b).eval()
			},
		}
	})
}