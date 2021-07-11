# Meet Kafka

모든 데이터(로그 메시지, 유저 활동 정보 등)는 의미를 담고 있고 다음으로 나아가기 위한 중요한 정보를 알려준다. 이를 알아내기 위해선 해당 데이터를 분석할 수 있는 곳에 데이터를 모아놓아야 한다. 

데이터를 더 빨리 처리할 수 있다면, 보다 더 민첩하고 신뢰성있는 서비스를 만들 수 있다. 데이터를 저장, 이동시키는 일에 더 적은 노력을 쓸수록, 비지니스에 집중할 수 있게된다. 이것이 data-driven-enterprise에서 pipeline이 아주 중요한 요소가 된 이유이다.

## Publish/Subscribe Messaging

Kafka에 대해 논하기 전에 먼저 Publish/Subscribe Messaging에 대해 이해햐는 것이 중요하다. PubSub 패턴의 송신자는 수신자를 특정하지 않고 데이터를 전달한다. 대신 Pub은 전달할 데이터를 분류하고, Sub는 분류된 데이터중 자신이 관심있는 주제의 데이터를 전달받는다. 보통 이러한 기능을 원활하게 구현하기 위해 메시지가 게시되는 중앙 지점인 Broker를 갖는다.

### How It Starts
아무개가 모니터링 정보를 수집해야 하는 어플리케이션을 만든다고 가정해보자. 처음엔 생성된 metrics를 직접적인 연결을 통해 모니터링 서버로 전달하는 방식으로 구현했다.

- img : a single, direct metrics publisher

어플리케이션의 규모가 커짐에 따라 새로운 서비스가 추가되고 새로운 모니터링 정보가 필요할 것이다. 요구사항에 따라 변경된 어플리케이션 구조는 매우 복잡해진다. 

- img : many metrics publishers, using direct connedctions 

복잡한 데이터 흐름을 단순화하기 위해 모든 metrics를 전달받고 해당 정보가 필요한 서버들에게 데이터를 가져갈 수 있도록 하는 하나의 어플리케이션(Broker)을 둔다. 이를 통해 구조적 복잡도를 줄일 수 있다.  

- img : a metrics publish/subscribe system

### Individual Queue Systems

어플리케이션을 운영하기 위해선 metrics 정보 이외에도 서버 로그, 사용자 활동 추적 정보를 수집할 필요가 있다. 다른 정보들을 수집하기 위해 metrics 수집 구조와 유사한 개별적인 Pub/Sub 구조들을 갖게된다.  

- img : Multiple publish/subscribe systems

이 구조는 분명히 Pub/Sub 간 직접적인 연결보단 좋은 구조이다. 하지만 관리해야 할 자원의 중복이 많다. 이 구조는 개별적인 버그와 한계를 갖고 있는 여러개의 데이터 큐 시스템을 관리해야 한다. 서비스 규모에 따라 새로운 데이터 형식을 수집하게 될 수도 있다. 여기서 필요한 것은 generic(비지니스가 성장함에 따라 추가되는 데이터를 모두 관리할 수 있는) 데이터를 다루는 하나의 중앙화된 시스템이다.

## Enter Kafka


