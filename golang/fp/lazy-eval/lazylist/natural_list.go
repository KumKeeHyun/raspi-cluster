package lazylist

func NaturalList() InfiniteLazyList {
	return &infiniteLazyList{
		l: increasingList(1),
	}
}

func increasingList(n int) *lazyList {
	return newLazyList(func() *evaluatedList {
		return &evaluatedList{
			item: n,
			nextEval: func() *evaluatedList {
				return increasingList(n + 1).eval()
			},
		}
	})
}
