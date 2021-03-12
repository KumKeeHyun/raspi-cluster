# etcd
`etcd-io/etcd` 소스코드를 분석하는 곳

## 왜 분석하게 되었는가
나름 `In Search of an Understandable Consensus Algorithm` 논문을 다 읽고 이해했다고 생각하고 구현을 해보려 했으나 능력이 부족함을 느낌. 알고리즘을 이해하는 것과 실제 구현하는 것은 너무나 큰 괴리가 있었음. 동시성 처리를 신경쓰는 것이 까다로웠고 스냅샷을 추가하려니까 논문에서 말했던 것보다 더 복잡해지고 생각할 상황이 많아진다는 것을 느낌. 

그래서 Raft 구현체들을 분석해보기로 함. 제일 유명한 프로젝트가 `kafka`,  `kubernetes`가 메타데이터를 저장하기 위해 사용하는 `zookeeper`,  `etcd`가 있었음. zookeeper는 Java로 구현되어 있고 Kafka 2.7 버전부터는 더이상 사용하지 않는다고 해서 뭔가 미래가 밝아보이지 않았음. etcd는 내가 공부하고있는 Golang으로 구현되어 있고 해당 raft 모듈이 `cockroachDB` 등의 다른 프로젝트에서도 사용되고 있는 것을 보아 공부하기에 아주 적합해 보였음. 

## etcd raft 모듈특징
Raft 프로토콜 구현을 위한 기본 특징

- 리더 선출
- 로그 복제
- 스냅샷
- 클러스터 멤버십 변경
- read-only 쿼리 성능 향상을 위한 처리 방식
    - read-only 쿼리를 leader와 follower 모두 처리
    - leader가 요청을 받으면 quorum을 확인하고 엔트리 로그 연산을 건너 뛰고 쿼리 처리
    - follower가 요청을 받으면 leader로부터 safe-read-index를 확인하고 쿼리 처리

기능 향상을 위한 추가 구현 특징

- 로그 복제 지연을 줄이기 위한 파이프라이닝
- 로그 복제 플로우컨트롤
- 네트워크 I/O(client request) 부하를 줄이기 위한 배치 처리
- 디스크 I/O(state machine) 부하를 줄이기 위한 배치 처리
- leader의 디스크에 병렬 쓰기(이건 무슨 뜻인지 정확히 모르겠음)
- follower가 받은 요청을 내부적으로 leader로 리다이렉션
- leader가 quorum을 잃으면 자동으로 follower로 전환됨
- quorum을 잃었을 때 로그가 무한하게 자라는 것을 방지