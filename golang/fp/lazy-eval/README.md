# lazy evaluation

- 출처 : Oreilly Functional Thinking

표현의 평가를 가능한 나중으로 미루는 기법은 `FP`의 특징 중 하나이다. 예를 들어 `lazy collection`는 모든 요소들을 한꺼번에 미리 계산하는 것이 아니라 필요에 따라 하나씩 전달한다. 이러한 구조는 다음과 같은 이점이 있다.

- 시간이 많이 걸리는 연산을 최대한 뒤로 미룰 수 있다.
- 무한 수열을 모델링할 수 있다.
- `map`, `filter`같은 연산을 효율적으로 계산할 수 있다.

## 시간이 많이 걸리는 연산 뒤로 미루기

우선 시간이 많이 걸리는 연산의 예시로 소수를 설정했다. 다음은 특정 정수가 소수인지 판단하는 코드이다.

[구현한 소스코드](./prime)

```go
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
```

특정 양의 정수가 소수인지 판단하는 연산의 시간복잡도는 `O(n^1/2)`이다. Benchmark 테스트 결과 대략 3초동안 60000까지의 소수를 찾아낼 수 있다. <s>실행환경이 `raspi-4`여서 성능이 구리다.</s>

```
goos: linux
goarch: arm64
pkg: github.com/KumKeeHyun/raspi-cluster/golang/fp/lazy-eval/prime
BenchmarkPrimeIter-4   	   59116	     49547 ns/op	    3106 B/op	      41 allocs/op
PASS
ok  	github.com/KumKeeHyun/raspi-cluster/golang/fp/lazy-eval/prime	3.146s
```

Golang는 게으른 컬렉션을 기본으로 제공하지 않지만 여러가지 방법으로 흉내낼 수 있다. 가장 친숙한 개념인 `iterator`를 이용해서 필요할 때만 다음 소수를 계산하도록 구현해보자.

```go
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
```

`1000000000`보다 큰 소수 N개를 구하려 했을 때, 가장 간단한 방법은 반복문을 돌면서 N개의 소수를 찾는 것이다. 하지만 N이 특정되지 않은 상태에서 `1000000000`보다 큰 첫번째, 두번째 소수를 구해야 하는 상황이라면? N값을 어림잡아 반복문을 돈다면, 실제론 안쓸 수도 있는 소수까지 계산하게 될 수 있다. 

`iterator`로 구현한 소수 반복자는 최종적으로 얼마나 많은 소수를 구해야할지 고려하지 않고도 필요한 만큼 소수를 구할 수 있다. 다음 소수값이 지금 당장 필요없다면 해당 연산을 하지않고 뒤로 미룰 수 있게 된 것이다. 

```go
iter := NewPrimeIteratorFrom(1000000000)
iter.Next() // 1000000007
iter.Next() // 1000000009
iter.Next() // 1000000021

// 1000000021 다음의 소수는 필요없음 -> 계산 미룸
```

## 게으른 리스트 만들기

[구현한 소스코드](./lazylist)

`Golang`은 엄격한 언어이지만, 클로저와 재귀적 표현을 이용하면 게으른 리스트를 구현할 수 있다. 클로저 블록의 실행은 뒤로 연기할 수 있기 때문이다.

엄격한 자료구조를 클로저로 감싸면 게으른 자료구조가 만들어진다.

```go
lazy := func() int { 
    return NextPrimeFrom(100000000000000000)
}

lazy() // lazy를 호출하기 전까지는 100000000000000000보다 큰 소수를 계산하지 않는다.
```

계속해서 요소를 더하기 위해, 요소를 앞에 더하고 클로저로 감싸면 게으른 리스트를 만들 수 있다.

```go
type evaluatedList struct {
    item int
    nextEval evalFunc
}

type evalFunc func() *evaluatedList

list := func() *evaluatedList {
	return &evaluatedList{
		item: 1,
		nextEval: func() *evaluatedList {
			return &evaluatedList{
				item: 2,
				nextEval: func() *evaluatedList {
					return nil
				},
			}
		},
	}
} 

list().item // 1
list().nextEval().item // 2
list().nextEval().nextEval() // nil
```

조금 복잡하지만 재귀적으로 요소를 추가하고 클로저로 감싸는 구조이다. 요소를 추가하기 쉽도록 `cons()`를 구현해보자.

> 현실적인 구현은 재귀를 피하지만, `lazy evaluation`을 이해하기 위해 간단하게 구현한 모델이다. 

### LazyList cons(prepend) 구현

우선 클로저를 감싸는 구조체를 만들고, 비어있는 리스트를 정의해야한다. 빈 리스트는 평가할 때 값이 `nil`이 나오도록 정의했다.

```go
type lazyList struct {
	eval evalFunc
}

func empty() *lazyList {
	return &lazyList{
        eval: func() *evaluatedList {
		    return nil
	    },
    }
}

func (l *lazyList) isEmpty() bool {
	return l.eval() == nil
}
```

빈 리스트의 앞에 요소를 추가해보자. 기존의 게으른 리스트(클로저)에 값을 추가한 뒤, 클로저로 감싸서 새로운 리스트를 생성한다. 

```go
func (l *lazyList) cons(item int) *lazyList {
	return &lazyList{
        // 값이 추가된 새로운 리스트
		eval: func() *evaluatedList { 
			return &evaluatedList{
                // 새로 추가한 값
				item:     item, 
                // 기존의 게으른 리스트
				nextEval: l.eval, 
			}
		},
	}
}
```

이렇게 만들어진 리스트는 요소가 추가된 순서의 역순으로 값을 반환한다.

```go
// 1 -> 2 -> 3 순서로 추리한 리스트
list := empty().cons(1).cons(2).cons(3)

list.eval().item // 3
list.eval().eval().item // 2
list.eval().eval().eval().item // 1 
```

`evaluatedList`구조체(평가된 리스트)를 보면, `evaluatedList.item`은 리스트의 첫 요소이고 `evaluatedList.nextEval`은 리스트의 첫 요소를 제외한 부분 리스트인 것을 알 수 있다. 해당 요소에 명시적으로 접근할 수 있도록 `head(), tail()`을 구현해보자.

```go
func (l *lazyList) head() int {
	if l.isEmpty() {
		panic("empty list!!")
	}
    // 값을 평가한 후 요소 반환
	return l.eval().item
}

func (l *lazyList) tail() *lazyList {
	if l.isEmpty() {
		return l
	}
    return &lazyList{
        // 값을 평가한 후 다음으로 평가할 리스트 반환
        eval: l.eval().nextEval,
    }
}

list := empty().cons(1).cons(2).cons(3)

list.head() // 3
list.tail().head() // 2
list.tail().tail().head() // 1 
```

나머지 메소드들은 리스트를 조작할 수 있는 고계함수들이다.

### LazyList reduce 구현

`reduce`연산을 통해 게으른 리스트의 전체 요소를 평가할 수 있다. 주어진 `초기값`과 `평가된 리스트 요소`를 순차적으로 연산하여 하나의 값을 반환하는 함수이다. 

`reduce`를 이용해서 리스트 요소의 합을 구해보자.

```go
func (l *lazyList) reduceAllItem(acc int, f func(a, b int) int) int {
	if l.isEmpty() {
		return acc
	}
	return l.tail().reduceAllItem(f(acc, l.Head()), f)
}

list := empty().cons(1).cons(2).cons(3)
list.reduceAllItem(0, func(a, b int) int {
    return a + b
}) // 1 + 2 + 3 = 6
```

다음은 리스트의 모든 요소를 평가하여 `slice`로 반환하는 `reduce`연산이다.

```go
func (l *lazyList) reduceAllList(acc []int, f func(a []int, b int) []int) []int {
	if l.isEmpty() {
		return acc
	}
	return l.tail().reduceAllList(f(acc, l.head()), f)
}

func (l *lazyList) takeAll() []int {
	return l.reduceAllList([]int{}, func(a []int, b int) []int {
		a = append(a, b)
		return a
	})
}

list := empty().cons(1).cons(2).cons(3)
list.takeAll() // []int{3, 2, 1}
```

### LazyList map 구현

일반적인 golang의 map함수 예시는 다음과 같다.

```go
func mapFunc(src []int, f func(a int) int) []int {
    dst := make([]int, len(src))
    for i, v := range src {
        dst[i] = f(v)
    }
    return dst
}
```

위의 함수는 리스트 전체를 순회하면서 각 요소를 계산하고 결과를 새로운 리스트로 반환한다. 

하지만 지금까지 구현한 리스트는 게으른 리스트다. 필수적이지 않은 연산은 최대한 뒤로 미루고 필요할 때 값을 평가한다. map연산은 리스트의 요소를 평가하기 전까지 최대한 뒤로 미룰 수 있다. 

다음은 인자로 주어진 연산을 값을 평가하는 시점에 계산하도록 하는 `mapFunc` 구현이다. 

```go
func (l *lazyList) mapFunc(f func(a int) int) *lazyList {
	if l.isEmpty() {
		return l
	}
	return newLazyList(func() *evaluatedList {
		return &evaluatedList{
			item: f(l.head()),
			nextEval: func() *evaluatedList {
				return l.tail().mapFunc(f).eval()
			},
		}
	})
}
```

새로 구현한 map함수를 보면 요소를 평가하는 방법을 수정할 뿐, 실제 리스트의 값을 계산하지 않는다.

### LazyList filter 구현

filter함수 구현도 map과 동일하다. 리스트의 값을 평가하지 않고 평가하는 방법만 수정한다.

```go
func (l *lazyList) filterFunc(f func(a int) bool) *lazyList {
	if l.isEmpty() {
		return l
	}

	if f(l.head()) {
		return newLazyList(func() *evaluatedList {
			return &evaluatedList{
				item: l.head(),
				nextEval: func() *evaluatedList {
					return l.tail().filterFunc(f).eval()
				},
			}
		})
	} else {
		return l.tail().filterFunc(f)
	}
}
```

### 고계함수 합성

map, filter, reduce를 합성해보기 전에 리스트를 쉽게 초기화하기 위한 함수를 작성해보자.

```go
// rangeFunc(5) -> []int{0, 1, 2, 3, 4}로 초기화된 게으른 리스트
func rangeFunc(n int) *lazyList {
	res := empty()
	if n < 1 {
		return res
	}
	return rangeFuncRec(res, n)
}

func rangeFuncRec(l *lazyList, n int) *lazyList {
	if n == 0 {
		return l
	}
	return rangeFuncRec(l.cons(n-1), n-1)
}

// rangeFromTo(1, 6) -> []int{1, 2, 3, 4, 5}로 초기화된 게으른 리스트
func rangeFromTo(f, t int) *lazyList {
	return rangeFunc(t - f).mapFunc(func(a int) int {
		return a + f
	})
}
```

이제 무한으로 합성해보자.

```go
// []int{2, 4, 6, 8, 10}
rangeFromTo(1, 11).filterFunc(func(a int) bool {
    return a % 2 == 0
}).takeAll()

// 80
rangeFromTo(1, 11).filterFunc(func(a int) bool {
    return a % 2 == 0
}).mapFunc(func(a int) int {
    return a + 10
}).reduceAllItem(0, func(a, b int) int {
    return a + b
})

// 20
// filter, map을 모든 요소에 적용하지 않고 첫 요소만 계산
rangeFromTo(1, 11).filterFunc(func(a int) bool {
		return a % 2 == 0
	}).mapFunc(func(a int) int {
		return a * 10
	}).head()
```

### 무한수열 모델링

`lazy evaluation`의 장점이라 했던 무한수열 모델링을 해보자. 다음은 자연수 집합을 게으른 리스트로 표현한 코드이다.

- 무한수열에서는 `reduceAll`을 사용할 수 없다.

```go
func NaturalList() *lazyList {
	return increasingList(1)
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

NaturalList().takeN(5) // []int{1, 2, 3, 4, 5}
NaturalList().takeN(10000) // []int{1, 2, 3, ..., 10000}

// []int{2, 4, 6, 8, 10}
NaturalList().filterFunc(func(a int) bool){
	return a % 2 == 0
}.takeN(5) 
```

같은 방식으로 피보나치 수열을 구현할 수 있다.

```go 
func FibonacciList() *lazyList {
	return fibonacci(0, 1)
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

// []int{1, 1, 2, 3, 5, 8, 13, 21, 34, 55}
FibonacciList().TakeN(10)
```

