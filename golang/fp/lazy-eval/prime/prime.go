package prime

import "math"

func NextPrimeFrom(num int) int {
	num += 1
	for !IsPrime(num) {
		num += 1
	}
	return num
}

func IsPrime(num int) bool {
	return sumOfFactors(num) == num+1
}

func sumOfFactors(num int) int {
	sum := 0
	for _, factor := range factorsOf(num) {
		sum += factor
	}
	return sum
}

func factorsOf(num int) []int {
	result := make([]int, 0)

	sqrtNum := int(math.Sqrt(float64(num))) + 1
	for i := 1; i < sqrtNum; i++ {
		if isFactor(num, i) {
			result = append(result, i, num/i)
		}
	}
	return result
}

func isFactor(num, potential int) bool {
	return num%potential == 0
}
