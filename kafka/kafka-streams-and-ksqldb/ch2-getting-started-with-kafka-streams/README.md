# Getting Started with Kafka Streams

## Before Kafka Streams

`Kafka Streams` 이전에는, 스트림 처리 어플리케이션을 만드는 것이 필요 이상으로 복잡했었다(데이터 처리에 대한 라이브러리 부족). 초창기 카프카 생태계에서, 카프카 기반 스트림 처리 어플리케이션을 만드는 데에는 두 가지 옵션이 있었다. 

- Producer/Consumer API를 사용해서 직접 구현
- `Spark Streaming`, `Flink`같은 다른 스트림 처리 프레임워크 사용

Producer/Consumer API를 직접 이용하는 방법은 API를 구현한 다양한 언어로 구현할 수 있고, 어떤 종류의 처리 로직이라도 직접 구현할 수 있다. 하지만 그 대가로 엄청난 양의 코드를 작성해야 한다.

- 스트림 처리 어플리케이션에 필요한 것들
    - Local and fault-tolerant state
    - A rich set of operators for transforming streams of data: aggregate, join, group events(window, bucket)...
    - More advanced representations of streams
    - Sophisticated handling of time

나머지 옵션으로 Spark Streaming, Flink 같은 완전한 스트림 처리 플랫폼을 사용하면, 플랫폼의 다양하고 풍부한 기능을 직접 구현하지 않고 사용할 수 있지만, 해당 자원을 관리하기 위해서 추가적인 복잡도를 감당해야 한다.

카프카 생태계에는 프로세싱 클러스터의 오버헤드 없이, 스트림 처리 요소를 제공하는 단순하고 성능이 좋은 솔루션이 필요했다. 

## Enter Kafka Streams

2016년에 Kafka Streams의 첫 버전이 릴리즈되면서 카프카 생태계에 큰 변화가 있었다. 수동적인 기능에 의존하던 기존 스트림 어플리케이션(API를 통해 직접 구현한 앱, 다른 스트림 처리 플랫폼)들은 카프카 커뮤니티가 개발한 패턴과 실시간 이벤트 스트림 처리를 위한 추상화를 활용한 고오급 어플리케이션(Kafka Streams App)으로 대체되었다.

Producer/Consumer, Connect API는 단순히 카프카에 데이터를 저장하거나 읽어오는 작업에 집중하는 반면, Kafka Streams는 실시간 스트림 처리에 초점을 둔다. 데이터 파이프라인을 따라 흐르는 이벤트 스트림을 쉽게 컨슘하고, 풍부한 스트림 처리 연잔자를 통해 데이터 변환 로직을 적용하고, 선택적으로 새로운 형태의 데이터를 카프카에 저장하는 기능을 지원한다.

<img src="img/kafka-streams.png"> 

## Features at a Glance

Kafka Streams는 현대의 스트림 처리 어플리케이션에 적합한 많은 기능을 지원한다.

- Java의 streaming API 와 유사한 high-level DSL
- 세심한 작업을 지원하는 low-level Processor API
- streams, table 같은 데이터 모델링을 위한 편리한 추상화
- streams, table 간 join 연산 지원
- stateless, stateful 스트림 처리를 위한 연산자와 유틸리티(ex: RocksDB)
- windowing, periodic 기능을 포함한 시간 기반 연산자
- 간단하고 쉬운 설치(단순한 라이브러리임)
- Scalability, Reliability, Maintainability 

## Operational Characteristics 

`Martin Kleppmann`의 `Designing Data-Intensive Applications`에서 저자는 데이터 시스템에서 세가지 중요한 목적을 강조한다.

- scalability 확장성
- Reliability 신뢰성
- Maintainability 유지보수성

### Scalability

부하가 증가하더라도 대처할 수 있고 성능을 유지할 수 있는 시스템을 `scalable`하다고 한다. 카프카 토픽은 파티션 개수를 늘리거나 브로커를 추가하는 방식으로 스케일링할 수 있다.

Kafka Streams에서는 한 파티션을 작업 단위로 처리하며, 컨슈머 그룹을 통해 자동으로 작업이 분산된다. 토픽이 파티션을 추가하는 방식으로 확장할 수 있는 것 처럼, Kafka Streams도 여러 인스턴스를 실행시켜서 확장할 수 있다.

Kafka Streams 어플리케이션은 대부분 다수의 인스턴스로 배포된다. 각 인스턴트는 컨슈머 그룹에 의해 전체 작업의 일부를 할당 받아 처리하게 된다. 만약 소스 토픽에 32개의 파티션이 있고 4개의 인스턴스를 실행시킨다면, 한 인스턴스당 8개의 파티션(8 * 4 = 32)을 처리한다. 인스턴스 개수를 늘려 16개를 실행시킨다면 각 인스턴스당 2개의 파티션(16 * 2 =32)을 처리한다.

### Reliablility

데이터 시스템에서 신뢰성은 엔지니어 관점(새벽에 서비스 장애로 일을 하고 싶지 않음)뿐만 아니라 고객의 관점(서비스를 이용할 수 없는 경우, 데이터 손실 등을 겪고 싶지 않음)에서도 중요한 요소이다. Kafka Streams에는 Consumer Groups을 통해 내결함성 기능을 갖는다. 

여러개의 인스턴스를 배포한 상황에서 장애로 한 개가 죽은 경우, 카프카는 자동으로 다른 인스턴스들에게 파티션을 재분배한다. 장애가 해결되고 다시 인스턴스가 실행되면(or k8s같은 시스템에 의해 새로 실행), 자동으로 카프카로부터 작업(파티션)을 할당받을 것이다.

### Maintainablility

Kafka Streams는 자바 라이브러리이기 때문에 버그수정이나 트러블슈팅이 비교적 쉽다. 또한 자바 어플리케이션을 모니터링하는 패턴(로그 수집 및 분석, JVM 성능 등)은 잘 확립되어 있다. 게다가 Kafka Streams API는 간결하고 직관적이기 때문에 code-level 유지보수가 비교적 쉽다.

## Comparison to Other Systems

### Deployment Model

Kafka Streams는 Spark Streaming, Flink같은 기술과 다른 배포 전략을 갖고 있다. 후자는 스트림 처리 프로그램을 실행하기 위해 각각 전용의 처리 클러스터를 구축해야 한다. 이것은 매우 큰 복잡도와 부담을 줄 수 있다. 잘 알려진 기업(Netflix)의 숙련된 엔지니어(Nitin Sharma) 조차 처리 클러스터의 오버헤드는 무시할 수 없다고 인정하고 있다. 

반면에 Kafka Streams 어플리케이션은 라이브러리를 통해 구현되는 standalone 어플리케이션이기 때문에 클러스터를 구축하지 않다고 된다. 또한 모니터링, 패키징, 배포하는 방법에 있어서 많은 자유도가 있다. 

### Processing Model

Spark Streaming, Trident의 처리 모델은 `micro-batching` 이다. micro-batching은 이벤트를 작은 그룹(batch)로 묶어서 메모리에 버퍼링하고 특정 주기마다 처리하는 방식이다.

Kafka Streams는 `event-at-a-time processing`(이벤트가 들어오는 즉시 처리)으로 구현되어서, micro-batching 보다 지연율이 낮다.

<img src="img/processing-model.png">

> micro-batching을 사용하는 프레임워크들은 보통 지연율이 높은 대신에 처리량에 최적화 되어있다. Kafka Streams에서는 파티션의 개수를 늘려서 데이터를 분할 처리하는 방식으로 낮은 지연율을 유지하면서 높은 처리량을 보장할 수 있다.

### Kappa Architecture


## Use Cases

## Processor Topologies

## Sub Topologies

### Depth First Processing

### Benefits of Dataflow Programming

### Tasks and Stream Threads

## High Level DSL Versus Low Level Processor API