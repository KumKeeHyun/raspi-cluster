# lazy evaluation

- 출처 : Oreilly Functional Thinking

표현의 평가를 가능한 나중으로 미루는 기법은 `FP`의 특징 중 하나이다. 예를 들어 `lazy collection`는 모든 요소들을 한꺼번에 미리 계산하는 것이 아니라 필요에 따라 하나씩 전달한다. 이러한 구조는 다음과 같은 이점이 있다.

- 시간이 많이 걸리는 연산을 최대한 뒤로 미룰 수 있다.
- 무한 수열을 모델링할 수 있다.
- `map`, `filter`같은 연산을 효율적으로 계산할 수 있다.

## 시간이 많이 걸리는 연산 뒤로 미루기

나중에 예시 추가할 예정

## 게으른 리스트 만들기

`Golang`은 엄격한 언어이지만, 클로저와 재귀적 표현을 이용하면 게으른 리스트를 구현할 수 있다. 클로저 블록의 실행은 뒤로 연기할 수 있기 때문이다.

엄격한 자료구조를 클로저로 감싸면 게으른 자료구조가 만들어진다.

```go
lazy := func() int { 
    return 1
}

lazy() // return 1
```

계속해서 요소를 더하기 위해, 요소를 앞에 더하고 클로저로 감싸면 게으른 리스트를 만들 수 있다.

```go
type value struct {
    item int
    nextEval evalFunc
}

type evalFunc func() *value

list := &value{
    item: 1,
    nextEval: func() *value {
        return &value{
            item: 2,
            nextEval: func() *value {
                return &value{
                    item: 3,
                    nextEval: func() *value {
                        return nil
                    },
                }
            },
        }
    },
}

list.item // 1
list.nextEval().item // 2
list.nextEval().nextEval().item // 3
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
        eval: func() *value {
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
		eval: func() *value { 
			return &value{
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

`value`구조체(평가된 리스트)를 보면, `value.item`은 리스트의 첫 요소이고 `value.eval`은 리스트의 첫 요소를 제외한 부분 리스트인 것을 알 수 있다. 해당 요소에 명시적으로 접근할 수 있도록 `head(), tail()`을 구현해보자.

```go
func (l *lazyList) head() int {
	if l.IsEmpty() {
		panic("empty list!!")
	}
    // 값을 평가한 후 요소 반환
	return l.eval().item
}

func (l *lazyList) tail() *lazyList {
	if l.IsEmpty() {
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
	if l.IsEmpty() {
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
	if l.IsEmpty() {
		return acc
	}
	return l.tail().reduceAllList(f(acc, l.Head()), f)
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

```go
func (l *lazyList) mapFunc(f func(a int) int) *lazyList {
	if l.IsEmpty() {
		return l
	}
	return newLazyList(func() *value {
		return &value{
			item: f(l.Head()),
			nextEval: func() *value {
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
	if l.IsEmpty() {
		return l
	}

	if f(l.Head()) {
		return newLazyList(func() *value {
			return &value{
				item: l.Head(),
				nextEval: func() *value {
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

```go
func NaturalNumList() *lazyList {
	return naturalNumList(1)
}

func naturalNumList(n int) *lazyList {
	return newLazyList(func() *value {
		return &value{
			item: n,
			nextEval: func() *value {
				return naturalNumList(n + 1).eval()
			},
		}
	})
}
```

