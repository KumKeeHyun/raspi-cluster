# Advanced State Management

## Persistent Store Disk Layout



## Fault Tolerance



### Changelog Topics



### Standby Replica



## Rebalancing: Enemy of the State

changelog topics는 어플리케이션이 state를 잃었을 때 다시 복구할 수 있도록 하고, standby replicas는 state를 복구하는 시간을 감출 수 있게 한다. 

하지만 Kafka Streams에 장애 내구성이 있다 하더라도, state store를 잃는 것이 아주 치명적이라는 사실은 변하지 않는다. 만약 복구해야 하는 state의 크기가 큰 경우, 토픽을 읽고 다시 적제하는 데에 수 분에서 극단적인 경우 수 시간까지 걸릴 수 있다. 

state를 복구해야 하는 상황의 가장 큰 원인은 파티션 재할당이다. 카프카는 컨슈머 멤버쉽에 변화가 있으면 자동으로 파티션을 재분배 한다. 이때 파티션 재할당은 크게 두가지 요소에 의해 이뤄진다. 

- group coordinator: 카프카 브로커에서 담당하며, 컨슈머 그룹의 멤버쉽을 관리(heartbeats 수신, 멤버쉽 변경을 감지하고 재할당 프로세스를 트리거)한다.
- group leader: 카프카 컨슈머에서 담당하며, 어떤 컨슈머에 어떤 파티션을 할당할 지를 담당한다. 

Kafka Streams에서는 파티션 재할당 이슈를 다루기 위해 두가지 전략을 사용한다.

- 가능하다면 state store가 다른 인스턴스로 옮겨지는 것을 막는다. -> 파티션 할당 변경을 최소화한다.
- 불가피하게 state store를 다른 인스턴스에 복귀해야 하는 경우, 최대한 빨리 이뤄지도록 한다. -> state store의 크기를 최대한 작게 유지한다.
 
 
## Preventing State Migration


### Sticky Assignment


### Static Membership



## Reducing the Impacts of Rebalances


### Incremental Cooperative Rebalancing


### Controlling State Size


#### Tombstones


#### Window retention


#### Aggressive topic compaction


## State Store Monitoring

### Adding State Listeners

Kafka Streams 어플리케이션은 `Created`, `Running`, `Rebalancing`, `Pending shutdown`, `Not runngin` 상태를 갖는다. 

<img src="img/application-state.png">

파티션 재할당은 Stateful Kafka Streams 어플리케이션에 특히 큰 영향을 주기 때문에, 어플리케이션이 언제, 얼마나 자주 재할당 상태가 되는지 모니터링하는 것이 중요하다. Kafka Streams는 상태가 변경될 때 호출될 callback을 등록할 수 있도록 `State Listener`를 제공한다.

```java
streams.setStateListener(
    (oldState, newState) -> {
      if (newState.equals(State.REBALANCING)) {
        // do something
      }
    });
``` 

### Adding State Resotre Listeners

Kafka Streams는 state store가 파티션 재할당에 의해 다시 초기화될 때 호출될 `State Restore Listener`를 제공한다. 

```java
KafkaStreams streams = new KafkaStreams(...);

streams.setGlobalStateRestoreListener(new RestoreListenerImpl()); 


// StateRestoreListener 인터페이스를 구현.
// onRestoreStart, onRestoreEnd, onBatchRestored 3가지 메소드를 구현해야 함.
class RestoreListenerImpl implements StateRestoreListener {

  private static final Logger log =
      LoggerFactory.getLogger(RestoreListenerImpl.class);
  
  // startingOffset이 0이라면, 전체 상태를 다시 초기화 해야하는 상황을 나타냄.
  @Override
  public void onRestoreStart(
      TopicPartition topicPartition,
      String storeName,
      long startingOffset,
      long endingOffset) {
    log.info("The following state store is being restored: {}", storeName);
  }
  
  @Override
  public void onRestoreEnd(
      TopicPartition topicPartition,
      String storeName,
      long totalRestored) {
    log.info("Restore complete for the following state store: {}", storeName);
  }
  
  // 이 callback method는 매우 빈번하게 호출될 수 있기 때문에, 동기적 연산을 사용하면 성능에 영향이 있을 수 있음.
  // 로깅하는 것조차 매우 부담스러울 수 있기 때문에, 일반적으로는 비워둠.
  @Override
  public void onBatchRestored(
      TopicPartition topicPartition,
      String storeName,
      long batchEndOffset,
      long numRestored) {
  }
} 
```

## Interactive Queries

Kafka Streams 2.5 버전 이전까지는, 어플리케이션이 죽거나 파티션 재할당 이슈가 발생하면, 해당 어플리케이션에 대한 interactive queries 요청은 실패했다. `rolling update`같은 상황에서 조차 가용성 이슈가 있었기 때문에, 고가용성을 요구하는 마이크로서비스를 구축하는 데에 있어서 큰 어려움이 있었다. 

```java
KeyQueryMetadata metadata =
  streams.queryMetadataForKey(storeName, key, Serdes.String().serializer());

String remoteHost = metadata.activeHost().host(); // 해당 호스트가 재할당 상태라면 요청은 실패함.
int remotePort = metadata.activeHost().port();
```

2.5 버전 이후부터는, standby replicas가 interactive queries 요청을 처리할 수 있게 되었다. 이를 통해 Kafka Streams 어플리케이션이 재할당 상태일 때도 고가용성의 API를 제공할 수 있게 되었다. 

```java
KeyQueryMetadata metadata =
    streams.queryMetadataForKey(storeName, key, Serdes.String().serializer());

// isAlive는 StateListeners를 통해 직접 구현해야 함.
if (isAlive(metadata.activeHost())) {
  // activeHost로 요청.
} else {
  // activeHost가 요청을 처리할 수 없는 상태라면
  // standby replicas로 요청.
  Set<HostInfo> standbys = metadata.standbyHosts();
}
```


## Custom State Stores