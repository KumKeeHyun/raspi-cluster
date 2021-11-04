# Chapter 4. Kafka Consumers: Reading Data from Kafka

## Kafka Consumer Concepts

### Consumers and Consumer Groups

### Consumer Groups and Partition Rebalance

한 그룹 내의 consumer들은 topic의 partitions의 소유권을 공유한다. 그룹에 새로운 consumer가 추가되면, 이전에 다른 consumer가 담당하던 partitions을 새로운 consumer가 받아서 처리한다. 같은 방법으로 그룹 내에 충돌이나 중지로 인해 한 consumer가 중단되면, 해당 consumer가 담당하던 partitions는 남아있는 consumers 중 하나에서 받아 이어서 처리한다. 파티션 재할당은 해당 그룹이 처리하는 topic이 변경(topic에 partition 추가 -> partition 개수 변동)될 때도 발생한다. 

파티션의 소유권이 한 consumer에서 다른 consumer로 이동하는 것을 재할당이라고 한다. 재할당은 consumer group의 고가용성과 확장성에 큰 영향을 미치는 동작이다. 재할당 동작은 파티션 할당 전략에 따라 2가지 방식이 있다.

- Eager Rebalances
    - 재할당이 진행될 동안, 모든 consumers가 동작을 멈추고, 모든 paritions에 대한 소유권을 포기한 후, 그룹 내에서 완전히 새로운 파티션 소유권을 갖는다.
    <img src="img/eager.png">

- Cooperative Rebalances 
    - 파티션 소유권에 변동이 있는 consumer group의 부분 집합만 재할당에 참여한다.
    - 재할당에 참여하지 않는 consumer들은 계속해서 데이터를 처리할 수 있다.
    - Cooperative는 두 맥락을 통해 진행된다.
        - 1. consumer group leader가 재할당되어야 하는 파티션을 소유하고 있는 consumer 들에게 해당 정보를 알린다. 정보를 받은 consumer들은 해당 파티션에 대한 작동을 멈추고 소유권을 포기한다.
        - 2. consumer group leader가 버려진 파티션들을 새로운 소유자들에게 배분한다.
    - 이 과정은 파티션이 안정적으로 할당될 때까지 여러번 반복될 수 있다.
    - 이 방식의 장점은 그룹 내의 모든 consumer들이 작동을 멈추는 상황을 피할 수 있다는 것이다. 그룹의 규모가 아주 큰 경우, Eager 정책을 사용하면 상당한 시간동안 모든 데이터 처리가 멈출 수 있다.   
    <img src="img/cooperative.png">>

### Static Group Membership


## Creating a Kafka Consumer

## Subscribing to Topics

## The Poll Loop

## Configuring Consumers


## Commits and Offsets

### Automatic Commit 

### Commit Current Offset

### Asynchronous Commit

### Combining Synchronous and Asynchronous Commits

### Commit Specified Offset


## Rebalance Listeners


## Consuming Records with Specific Offsets


