# etcd raft 모듈 코드 분석

- 후속 시리즈: [etcd raft 모듈 사용해보기](./understanding-etcd-raft-2.md)

군대 입대 이전에 공허한 마음에 썻던 글이라 부족하기도 하고 전역한 시점에서 2년동안 업데이트 된 내용이 있어서 전체적으로 글을 다듬고 몇가지 부분을 수정했습니다.

소스코드는 '23.03.05 기준으로 최신 테그인 `v3.6.0-alpha.0`을 사용했습니다.


# TOC
<!--ts-->
- [etcd raft 모듈 코드 분석](#etcd-raft-모듈-코드-분석)
- [TOC](#toc)
- [서론](#서론)
	- [왜 이 코드를 분석하게 되었나](#왜-이-코드를-분석하게-되었나)
	- [들어가기 전에](#들어가기-전에)
	- [간단하게 raft란](#간단하게-raft란)
- [etcd raft Readme](#etcd-raft-readme)
	- [Features](#features)
		- [기본적인 raft 프로토콜 구현](#기본적인-raft-프로토콜-구현)
		- [성능 향상을 위한 추가 구현](#성능-향상을-위한-추가-구현)
	- [Notable Users](#notable-users)
	- [Usage](#usage)
		- [1. raft의 주요 Object인 node를 생성, 실행](#1-raft의-주요-object인-node를-생성-실행)
		- [2. node.Ready() 채널을 읽어서 스토리지를 업데이트하거나 네트워크를 통해 다른 노드로 메시지 전송](#2-nodeready-채널을-읽어서-스토리지를-업데이트하거나-네트워크를-통해-다른-노드로-메시지-전송)
		- [3. 주기적으로 node.Tick()을 호출해서 HeartbeatTimeout, ElectionTimeout 발생시키기](#3-주기적으로-nodetick을-호출해서-heartbeattimeout-electiontimeout-발생시키기)
		- [4. raft 모듈 내부로 필요한 메시지를 전달](#4-raft-모듈-내부로-필요한-메시지를-전달)
- [etcd raft inside](#etcd-raft-inside)
	- [raft.Node와 Transport 계층](#raftnode와-transport-계층)
		- [raftpb.Message](#raftpbmessage)
		- [raft.Node에서 Transport 계층으로](#raftnode에서-transport-계층으로)
		- [Transport 계층에서 raft.Node로](#transport-계층에서-raftnode로)
	- [raft.Node가 raftpb.Message를 처리하는 과정](#raftnode가-raftpbmessage를-처리하는-과정)
		- [Step 함수](#step-함수)
	- [raft.Node가 raftpb.Message를 Application에게 전달하는 과정](#raftnode가-raftpbmessage를-application에게-전달하는-과정)
		- [Ready를 통한 배치 처리](#ready를-통한-배치-처리)
		- [send 헬퍼 함수](#send-헬퍼-함수)
- [raft 알고리즘](#raft-알고리즘)
	- [Leader 선출 처리 과정](#leader-선출-처리-과정)
		- [1. ElectionTimeout 발생](#1-electiontimeout-발생)
		- [2. Step 함수에서 hup 함수 호출](#2-step-함수에서-hup-함수-호출)
		- [3. campaign 함수 동작](#3-campaign-함수-동작)
		- [4. MsgVote를 받은 다른 노드들의 동작](#4-msgvote를-받은-다른-노드들의-동작)
		- [5. MsgVoteResp를 받은 Candidate 노드의 동작](#5-msgvoteresp를-받은-candidate-노드의-동작)
	- [로그 복제 처리 과정](#로그-복제-처리-과정)
		- [1. 새로운 변동 사항을 raft 클러스터에 제안](#1-새로운-변동-사항을-raft-클러스터에-제안)
		- [2. MsgApp을 받은 Follower 노드의 동작](#2-msgapp을-받은-follower-노드의-동작)
		- [3. MsgAppResp을 받은 Leader 노드의 동작](#3-msgappresp을-받은-leader-노드의-동작)
		- [4. Leader가 Follower의 로그 복제 상황을 빠르게 수정하는 방법](#4-leader가-follower의-로그-복제-상황을-빠르게-수정하는-방법)
			- [case 1](#case-1)
			- [case 2](#case-2)
	- [raft.Node에서 채널 이벤트 기반으로 오케스트레이션 하기](#raftnode에서-채널-이벤트-기반으로-오케스트레이션-하기)
		- [이벤트 루프](#이벤트-루프)
		- [Chan을 이용한 몇가지 패턴](#chan을-이용한-몇가지-패턴)
			- [select case](#select-case)
			- [chan chan](#chan-chan)
- [마치면서](#마치면서)
	- [분석글에 더 추가해야 할 내용](#분석글에-더-추가해야-할-내용)
	- [감사합니다](#감사합니다)
<!--te-->

# 서론
## 왜 이 코드를 분석하게 되었나
코시국으로 갈피를 못잡고 있던 작년에 선배의 소개로 `Kafka`, `Elasticsearch`을 공부하면서 분산 환경을 처음 접했습니다. 여러 머신에서 요청을 분산해서 처리하고 하나의 머신이 동작을 멈춰도 복제된 내용을 통해 내구성을 갖는 시스템이 너무 멋있었고 이런 분산, 복제 시스템에 관심을 갖고 공부하게 되었습니다. 

저는 `Medium` 사이트에서 글을 찾아 읽는 것을 좋아하는데 마침 추천 리스트에 있는 `나는 어떻게 분산 환경을 공부했는가(영어로 쓰여있었음)`라는 글을 읽고 `raft 합의 알고리즘`을 알게되었습니다. 자연스럽게 raft에 대해 공부하면서 `In Search of an Understandable Consensus Algorithm-Diego` 논문과 여러 아티클을 읽게 되었습니다. 이후 근자감이 차올라 직접 구현을 해보려다가 raft가 얼마나 복잡하고 어려운 알고리즘인지만 깨닫고 결국 구현되어있는 코드를 분석해보는 목표를 세웠습니다. 

제가 알고있는 프로젝트중에 raft를 사용하는 프로젝트는 ~~`Zookeeper`~~, `etcd`정도였습니다.(수정: Zookeeper는 raft 가 아닌 zap 알고리즘을 사용함) 

Zookeeper는 Kafka를 사용해보면서 친숙함이 있었지만 Kafka 이후 버전에서는 자체적으로 raft를 구현하고 Zookeeper를 사용하지 않는다고 들어서 ~~미래가 밝아보이지 않다는 생각을 했습니다~~(수정: 하둡 생태계는 단단하다). 

또 ~~Java에 대한 지식이 깊지 않아서 제외 했습니다~~(수정: 군대에서 21개월 동안 스프링으로 구르고 구른 덕분에 얕진 않을 것). 

이에 반해서 etcd는 정말 핫한 `Kubernetes`에서 메타데이터를 저장하는 Key-Value Store로 사용하고 있고 etcd의 raft 모듈이 cockroachDB같은 다른 프로젝트에서도 사용할만큼 신뢰성이 높고 추상화도 잘되어있는 것 같아서 etcd 코드를 선택했습니다. 추가적으로 `Golang`으로 구현되어있기 때문에 Go를 공부하고있는 저에게 정말 알맞는 코드라 생각했습니다. 

## 들어가기 전에
저는 대학교 3학년을 수료한 상태이고 글을 써본적이 별로 없는 공대생입니다. 분산 시스템에 대한 강의를 수강한 것도 아니고 인터넷에서 여러 자료를 찾아가면서 공부했기 때문에 `raft 합의 알고리즘`에 친숙하지 않으신 분들을 위해 개념부터 설명해드리기엔 저의 능력이 부족합니다. 이런 큰 코드를 분석해본 경험도 적기 때문에 읽으시면서 글이 매끄럽지 않더라도 양해 부탁드립니다.

## 간단하게 raft란
raft는 여러 머신에서 복제된 상태를 유지할 수 있게 하는 합의 알고리즘입니다. raft는 state-machine 자체를 복제하는 것이 아니라 state-machine에 적용할 변동사항을 로그형태로 관리하고 이 로그를 복제합니다. 모든 데이터의 흐름(복제)은 Leader에서 Follower로 흐릅니다. 한 논리적인 클러스터에서 Leader는 1개만 존재하고 이러한 Leader는 선거를 통해 선출됩니다. 

- raft 설명 자료
	- [In Search of an Understandable Consensus Algorithm 논문 PDF](https://raft.github.io/raft.pdf)
	- [raft 동작 설명 Medium 글](https://codeburst.io/making-sense-of-the-raft-distributed-consensus-algorithm-part-1-3ecf90b0b361)

# etcd raft Readme

[README 바로가기](https://github.com/etcd-io/raft/blob/main/README.md)

## Features
먼저 etcd raft 라이브러리의 Readme 문서에 쓰여있는 특징은 다음과 같습니다. 

### 기본적인 raft 프로토콜 구현
- 리더 선출
- 로그 복제
- 로그 압축
- 클러스터 멤버십 변경
- 리더 변경 확장(???)
- 리더와 팔로워 모두에서 처리하는 효율적인 read-only 쿼리
  - 리더는 읽기 쿼리를 처리하기 전에 quorum 확인
  - 팔로워는 읽기 쿼리를 처리하기 전에 리더로부터 안전하게 읽을 수 있는 인덱스 확인
- 리더와 팔로워 모두에서 처리하는 효율적인 lease 기반 read-only 쿼리

### 성능 향상을 위한 추가 구현
- 로그 복제 지연을 줄이기 위한 pipelining
- 로그 복제를 위한 flow-control
- 네트워크 Sync I/O 부하를 줄이기 위한 배치 처리
- 디스크 Sync I/O 부하를 줄이기 위한 배치 처리
- follower가 받은 요청을 내부적으로 leader로 redirection
- leader가 quorum을 잃으면 자동으로 follower로 전환
- quorum을 잃었을 때 로그가 무한하게 자라는 것을 방지

대부분의 raft 구현은 Storage 처리, 로그 메시징 직렬화와 네트워크 전송등을 포함한 `Monolithic` 디자인을 갖고 있습니다. 대신 etcd의 raft 라이브러리는 raft의 핵심 알고리즘만 구현하여 `최소한의 디자인`만 따릅니다.

## Notable Users

- cockroachdb : 확장 가능하고 잘 살아남고 강한 일관성을 지원하는 SQL 데이터베이스
- dgraph : 확장 가능하고 지연시간이 적고 높은 처리량을 지원하는 Graph 데이터베이스
- etcd
- tikv : rust로 구현된 트랜잭션을 지원하는 분산 key-value 데이터베이스
- swarmkit : 분산 시스템 오케스트레이션을 위한 툴킷
- chain core : 블록체인 네트워크를 위한 프로젝트

<br>

## Usage

### 1. raft의 주요 Object인 node를 생성, 실행
- 3개의 노드(id:1,2,3)로 구성된 클러스터를 초기화하는 경우
```go
  // raftLog(entries, snapshot을 관리하는 서장소)를 외부에서 주입 
  storage := raft.NewMemoryStorage()
  c := &raft.Config{
    ID:              0x01,
    ElectionTick:    10,
    HeartbeatTick:   1,
    Storage:         storage,
    MaxSizePerMsg:   4096,
    MaxInflightMsgs: 256,
  }

  // 클러스터 내 다른 노드들의 리스트를 전달
  // 다른 노드들도 개별로 실행해야 함을 주의
  // 
  // 기존 문서에는 peers에 노드 자신의 정보는 전달하지 않는 것으로 설명함.
  // n := raft.StartNode(c, []raft.Peer{{ID: 0x02}, {ID: 0x03}})
  // 
  // 하지만 etcd clustering 문서와 소스코드를 찾아봐도 peers에 자신의 정보까지 포함해서 넘기고 있음. 
  // https://etcd.io/docs/v3.5/op-guide/clustering/
  n := raft.StartNode(c, []raft.Peer{{ID: 0x01}, {ID: 0x02}, {ID: 0x03}})
```

- 하나의 노드로 구성된 클러스터
```go
  // Create storage and config as shown above.
  // Set peer list to itself, so this node can become the leader of this single-node cluster.
  peers := []raft.Peer{{ID: 0x01}}
  n := raft.StartNode(c, peers)
```

- 기존 클러스터에 새로운 노드를 추가하는 경우(join)
```go
  // 기존 문서에는 peers를 비워서 raft.StartNode 함수를 호출하라고 함.
  // 하지만 이 함수는 peers가 비어있으면 panic을 던짐.
  // n := raft.StartNode(c, nil)
  // 
  // 새로운 노드가 클러스터에 추가되는 경우에도 RestartNode를 호출해야함.
  n := raft.RestartNode(c)
```

- 이전에 작동을 멈추었던 노드를 재시작하는 경우
```go
  storage := raft.NewMemoryStorage()
  // 영구적인 저장소에 저장되어있던 snapshot, entries, state를 로드
  storage.ApplySnapshot(snapshot)
  storage.SetHardState(state)
  storage.Append(entries)

  // Restart raft without peer information.
  // Peer information is already included in the storage.
  n := raft.RestartNode(c)
```

<br>

### 2. node.Ready() 채널을 읽어서 스토리지를 업데이트하거나 네트워크를 통해 다른 노드로 메시지 전송

1. ready.Entries, ready.HardState, ready.Snapshot 순서대로 영구적인 스토리지에 저장
   - 스토리지가 3개를 원자적으로 저장하는 것을 보장한다면 한번에 저장해도 상관 없음
2. ready.Messages에 있는 메시지들을 네트워크 계층을 통해 지정된 노드로 전달
   - 메시지 전송은 반드시 이전 ready 배치의 Entries와 현재 ready 배치의 HardState가 모두 저장된 이후에 수행해야 함
   - 메시지가 스냅샷인 경우 스냅샷이 전송된 뒤에 node.ReportSnaphost 함수를
3. Snapshot이나 CommitedEntries가 있는 경우 state-machine에 적용
   - CommitedEntries에 구성 변경 엔트리가 있는 경우 노드에 반영해야 함
   - `NodeID` 필드를 `0`으로 변경해서 구성 변경을 취소할 수 있지만 결국 어떠한 형식으로든 ApplyConfChange 함수를 통해 노드에 반영해야 함
   - 취소 결정은 전적으로 state-machine을 기반으로 해야 함. 노드 헬스 체크 정보같은 외부 요소를 기반으로 하면 안됨 
4. node.Advance()를 호출해서 다음 배치에 대한 준비 상태를 알림
   - 1단계를 수행한 뒤에 호출해도 상관 없지만 ready로 전달된 변경사항들은 항상 순서대로 처리되어야 함
   

### 3. 주기적으로 node.Tick()을 호출해서 HeartbeatTimeout, ElectionTimeout 발생시키기

```go
// Usage-2,3 을 처리하는 코드
  for {
    select {
    case <-s.Ticker: // 3. time.Ticker를 사용해서 주기적으로 Node.Tick()을 호출
      n.Tick()
    case rd := <-s.Node.Ready(): // 2. raft 모듈이 스토리지, 네트워크 계층에서 처리할 것들을 배치형식으로 전달
      saveToStorage(rd.HardState, rd.Entries, rd.Snapshot) // 2-1. Entries, HardState, Snapshot을 영구적인 스토리지에 저장
      send(rd.Messages) // 2-2. 네트워크 계층을 통해 Messages를 지정된 노드로 전달 
      if !raft.IsEmptySnap(rd.Snapshot) { // 2-3. Snapshot이 있다면 적용
        processSnapshot(rd.Snapshot)
      }
      for _, entry := range rd.CommittedEntries { // 2-3. CommitedEntries가 있다면 적용
        process(entry)
        if entry.Type == raftpb.EntryConfChange {
          var cc raftpb.ConfChange
          cc.Unmarshal(entry.Data)
          s.Node.ApplyConfChange(cc)
        }
      }
      s.Node.Advance() // 2-4. 다음 ready 배치를 받기 위해 Node.Advance() 호출
    case <-s.done:
      return
    }
  }
```

<br>

### 4. raft 모듈 내부로 필요한 메시지를 전달
- 네트워크 계층을 통해 다른 노드로부터 받은 메시지들은 Node.Step(ctx context.Context, m raftpb.Message)을 통해 모듈 내부로 전달
```go
  func recvRaftRPC(ctx context.Context, m raftpb.Message) {
      n.Step(ctx, m) // raft 모듈이 메시지를 처리할 수 있도록 전달
  }
```

- 어플리케이션에서 state-machine에 대한 변경을 제안
```go
  // 일반적인 쓰기 작업
  n.Propose(ctx, data)

  // 클러스터 멤버쉽 변경
  n.ProposeConfChange(ctx, cc)
```

<br>

# etcd raft inside

## raft.Node와 Transport 계층
<img src="https://user-images.githubusercontent.com/44857109/112468548-bd501c00-8dab-11eb-8b63-bf461cde45e4.png" width="100%" height="100%">

- 이 그림은 raft 라이브러리의 핵심인 raft.Node가 Application(Transport, Storage 계층)와 소통하는 과정을 정리한 그림이다.

### raftpb.Message
소스 코드를 읽기 전에는 리더 선출, 로그 복제 등의 작업에서 노드간 네트워크 통신을 분리할 정도로 추상화가 가능한지 의문이 들었다. 

etcd는 네트워크가 필요한 작업들(`requestVote`, `appendEntries`, `heartbeat`, `installSnapshot` 등)을 각각 요청, 응답, 제안 등의 형태로 정의(`raftpb.MessageType`)하고, raft 알고리즘 수행에 필요한 모든 작업을 `raftpb.Message`로 표현할 수 있도록 추상화 했다.

```protobuf
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raftpb/raft.proto#L44
// For description of different message types, see:
// https://pkg.go.dev/go.etcd.io/etcd/raft/v3#hdr-MessageType
enum MessageType {
	MsgHup             = 0; // 로컬 노드가 선거를 시작하도록 함. (Local)
	MsgBeat            = 1; // 로컬 노드가 모든 peer 노드들에게 MsgHeartBeat 메시지를 보내도록 함 (Leader -> Leader)
	MsgProp            = 2; // state-machine을 변경하기 위한 제안 메시지 (Local)
	MsgApp             = 3; // 로그 복제를 위해 Entries를 보냄 (Leader -> Follower)
	MsgAppResp         = 4; // MsgApp 응답 (Follower -> Leader)
	MsgVote            = 5; // peer 노드들에게 투표 요청을 보냄 (Candidate -> Follower)
	MsgVoteResp        = 6; // MsgVote 응답 (Follower -> Candidate)
	MsgSnap            = 7; // Follower에게 Snapshot을 보냄 (Leader -> Follower)
	MsgHeartbeat       = 8; // peer 노드들에게 heartbeat을 보냄 (Leader -> Follower)
	MsgHeartbeatResp   = 9; // MsgHeartbeat 메시지 응답 (Follower -> Leader)
	MsgUnreachable     = 10;
	MsgSnapStatus      = 11; // Follower가 Snapshot을 적용하던 도중 오류가 발생하면 Leader에게 알림. 정상적으로 처리된 경우에는 MsgAppResp를 보냄 (Follower -> Leader)
	MsgCheckQuorum     = 12;
	MsgTransferLeader  = 13; // Leader에게 새로운 선거를 시작하자고 제안. Leader는 해당 peer 노드의 상태를 보고 MsgTimeout 메시지를 전달 (? -> Leader)
	MsgTimeoutNow      = 14; // peer 노드에게 선거를 시작하라고 알림 (Leader -> ?)
	MsgReadIndex       = 15; // Follower가 client's read request를 처리하기 위해 Leader에게 ReadIndex를 요청 (Follower -> Leader)
	MsgReadIndexResp   = 16; // MsgReadIndex 응답 (Leader -> Follower)
	MsgPreVote         = 17; // preVote 요청
	MsgPreVoteResp     = 18; // preVote 응답
}

message Message {
	optional MessageType type        = 1  [(gogoproto.nullable) = false];
	optional uint64      to          = 2  [(gogoproto.nullable) = false];
	optional uint64      from        = 3  [(gogoproto.nullable) = false];
	optional uint64      term        = 4  [(gogoproto.nullable) = false];
	optional uint64      logTerm     = 5  [(gogoproto.nullable) = false];
	optional uint64      index       = 6  [(gogoproto.nullable) = false];
	repeated Entry       entries     = 7  [(gogoproto.nullable) = false];
	optional uint64      commit      = 8  [(gogoproto.nullable) = false];
	optional Snapshot    snapshot    = 9  [(gogoproto.nullable) = false];
	optional bool        reject      = 10 [(gogoproto.nullable) = false];
	optional uint64      rejectHint  = 11 [(gogoproto.nullable) = false];
	optional bytes       context     = 12;
}
```

위와 같이 raft 알고리즘을 수행하기 위한 모든 네트워크 작업과 내부에서 수행해야 하는 작업을 모두 raftpb.Message로 표현할 수 있다. 

etcd raft 모듈은 `node.Step`, `node.Propose` 함수를 통해 raftpb.Message를 전달받고, raft 알고리즘을 수행한 후에, 외부로 전달해야 하는 raftpb.Message는 `node.Ready` 채널을 통해 전달하는 방식으로 작동한다. 

네트워크를 통해 peers와 raftpb.Message를 주고 받는 작업, commit된 raftpb.Message를 스토리지에 반영하는 작업, 스냅샷을 생성하고 복구하는  작업들은 모두 외부 구현에게 맡긴다. 

<br>

### raft.Node에서 Transport 계층으로

etcd는 raftpb.Message를 다른 노드들과 주고 받기 위해 http를 사용한다. [etcd-io/etcd/server/etcdserver/api/rafthttp](https://github.com/etcd-io/etcd/tree/v3.6.0-alpha.0/server/etcdserver/api/rafthttp)에 구현되어 있으며 multiple-raft-group 같은 특별한 요구사항이 아니라면 그대로 사용해도 무방하다.

<img src="https://user-images.githubusercontent.com/44857109/112740861-47ba9a80-8fbb-11eb-9de8-5713cfb8ecd7.png" width="100%" height="100%">

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/etcdserver/raft.go#L158
func (r *raftNode) start(rh *raftReadyHandler) {
	// 실제 etcd의 raft 모듈 이벤트 처리 루프

	// ...
	go func() {
		// ...
		for {
			select {
			case <-r.ticker.C:
				r.tick()
			case rd := <-r.Ready():
				// 스토리지 계층에 CommittedEntries를 병렬로 처리

				// 리더는 CommittedEnrites를 처리하는 작업과 복제 작업을 동시에 수행할 수 있음
				// 따라서 병렬로 실행된 앞선 작업이 끝난 것을 확인하지 않고 Send 작업 수행 가능
				if islead {
					r.transport.Send(r.processMessages(rd.Messages))
				}

				// Snapshot이 있다면, 영구 저장소에 저장
				// HardState, Entries 영구 저장소에 저장
				// Snapshot이 있다면, 스토리지 계층에 적용

				if !islead {
					msgs := r.processMessages(rd.Messages)

					// 만약 CommittedEntries에 ConfChange가 있다면 entries가 처리될 때까지 기다림
					// 기다리지 않으면 raft state machine이 꼬일 수 있음
					r.transport.Send(msgs)
				}
				// ...

				r.Advance()
			case <-r.stopped:
				return
			}
		}
	}()
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/etcdserver/api/rafthttp/transport.go#L175
func (t *Transport) Send(msgs []raftpb.Message) {
	for _, m := range msgs {
		// ...
		to := types.ID(m.To)

		t.mu.RLock()
		p, pok := t.peers[to] // 네트워크 계층에서 관리하는 peers에서 m.To(id 값)에 해당하는 peer 검색
		g, rok := t.remotes[to]
		t.mu.RUnlock()

		if pok {
			// ...
			p.send(m) // 해당 peer 에게 메시지 전송
			continue
		}
    // ...
	}
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/etcdserver/api/rafthttp/peer.go#L236
func (p *peer) send(m raftpb.Message) {
  // ...
	writec, name := p.pick(m) // 실제 POST 요청 전송을 전담하는 고루틴으로 보낼 채널
	select {
	case writec <- m:
	default:
		// ...
	}
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/etcdserver/api/rafthttp/pipeline.go#L93
func (p *pipeline) handle() { 
	// pipeline을 담당하는 고루틴
	defer p.wg.Done()

	for {
		select {
		case m := <-p.msgc:
			start := time.Now()
			err := p.post(pbutil.MustMarshal(&m)) // 실제 POST 요청
			end := time.Now()
			// ...
		case <-p.stopc:
			return
		}
	}
}
```

<br>

### Transport 계층에서 raft.Node로

<img src="https://user-images.githubusercontent.com/44857109/112741090-4b4f2100-8fbd-11eb-8d14-d9819af530ee.png" width="100%" height="100%">

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/embed/etcd.go#L549
func (e *Etcd) servePeers() (err error) {
	ph := etcdhttp.NewPeerHandler(e.GetLogger(), e.Server)
	// ...

	for _, p := range e.Peers {
		u := p.Listener.Addr().String()
		gs := v3rpc.Server(e.Server, peerTLScfg, nil)
		m := cmux.New(p.Listener)
		go gs.Serve(m.Match(cmux.HTTP2()))
		srv := &http.Server{
			Handler:     grpcHandlerFunc(gs, ph),
			ReadTimeout: 5 * time.Minute,
			ErrorLog:    defaultLog.New(io.Discard, "", 0), // do not log user error
		}
		go srv.Serve(m.Match(cmux.Any()))
		// ...
	}

	// ...
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/etcdserver/api/etcdhttp/peer.go#L39
func NewPeerHandler(lg *zap.Logger, s etcdserver.ServerPeerV2) http.Handler {
	return newPeerHandler(lg, s, s.RaftHandler(), s.LeaseHandler(), s.HashKVHandler(), s.DowngradeEnabledHandler())
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/etcdserver/server.go#L632
func (s *EtcdServer) RaftHandler() http.Handler {
	return s.r.transport.Handler() // rafthttp의 transport
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/etcdserver/api/rafthttp/transport.go#L157
func (t *Transport) Handler() http.Handler {
	pipelineHandler := newPipelineHandler(t, t.Raft, t.ClusterID) // http.Handler 인터페이스 구현체
	// ...
	mux := http.NewServeMux()
	mux.Handle(RaftPrefix, pipelineHandler)
	// ...
	return mux
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/etcdserver/api/rafthttp/http.go#L95
func (h *pipelineHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ...
	b, err := ioutil.ReadAll(limitedr)
	// ...
	var m raftpb.Message
	if err := m.Unmarshal(b); err != nil {
		// ...
		return
	}
	// ...
	if err := h.r.Process(context.TODO(), m); err != nil { // raft 모듈로 메시지 전달
		// ...
		return
	}
	// ...
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/etcdserver/server.go#L686
func (s *EtcdServer) Process(ctx context.Context, m raftpb.Message) error {
	// ...
	return s.r.Step(ctx, m)
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/node.go#L429
func (n *node) Step(ctx context.Context, m pb.Message) error {
	// ...
	return n.step(ctx, m)
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/node.go#L454
func (n *node) step(ctx context.Context, m pb.Message) error {
	return n.stepWithWaitOption(ctx, m, false)
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/node.go#L464
func (n *node) stepWithWaitOption(ctx context.Context, m pb.Message, wait bool) error {
	if m.Type != pb.MsgProp {
		select {
		case n.recvc <- m:
			return nil
		// ...
		}
	}
	// ...
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/node.go#L303
func (n *node) run() {
	r := n.rn.raft // RawNode
    // ...
	for {
		// ...
		select {
		// ...
		case m := <-n.recvc:
			// 적절하지 않은 메시지 필터링
			if pr := r.prs.Progress[m.From]; pr != nil || !IsResponseMsg(m.Type) {
				r.Step(m) // 실제 raft 모듈이 메시지를 처리하도록 전달
				// -> https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L847
			}
    	// ...
		}
	}
}
```

<br>

## raft.Node가 raftpb.Message를 처리하는 과정
지금까지 raft 모듈과 전송 계층을 잇는 방법을 살펴보았다. 이제 실제 모듈 내부에서 raftpb.Message를 어떻게 처리하는지 살펴봐야 한다. 모든 raftpb.Message는 `raft.Step` 함수에서 처리된다. 

다음 코드는 raft 모듈 내부에서 raftpb.Message를 처리하기 위해 raft.Step 함수를 호출하는 것을 보여준다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/node.go#L303
func (n *node) run() {
	// ...
	for {
		// ...
		select {
		case pm := <-propc:
			m := pm.m
			m.From = r.id
			err := r.Step(m) // 로컬(client)에서 raft 모듈로 제안한 메시지
			// ...
		case m := <-n.recvc:
			if pr := r.prs.Progress[m.From]; pr != nil || !IsResponseMsg(m.Type) {
				r.Step(m) // 네트워크 계층(peers)을 통해 전달받은 메시지들
			}
		// ...
	}
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L645
func (r *raft) tickElection() {
	r.electionElapsed++

	if r.promotable() && r.pastElectionTimeout() {
		r.electionElapsed = 0
		r.Step(pb.Message{From: r.id, Type: pb.MsgHup}) // 로컬 노드가 선거를 시작하도록 MsgHup 메시지를 생성한 후 바로 Step 함수를 통해 처리
	}
}
```

<br>

### Step 함수

메시지가 client에 의해 생성되거나 네트워크 계층에서 전달받거나 같은 상황에 상관없이 모두 raft.Step 함수를 호출한다. 여기서 몇가지 의문이 생길 수 있다. raft 노드는 `Leader`, `Candidate`, `Follower` 상태에 따라 같은 메시지도 다르게 처리해야 하지만 그런 로직은 보이지 않고 무조건 하나의 함수를 호출하도록 추상화 되어있기 때문이다. etcd는 이런 로직을 구현하기 위해서 각 상태에 따른 stepXXX 함수를 따로 작성하고 이를 raft.Step 함수로 감싼다. 

다음 코드는 각 상태에 따른 stepXXX 함수, 이를 래핑한 raft.Step 함수, raft.Step 함수가 어떻게 노드 상태에 따라 다른 step 함수를 호출하는지 보여준다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L989
type stepFunc func(r *raft, m pb.Message) error // 각 상태에 따른 step 함수의 타입

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L991
func stepLeader(r *raft, m pb.Message) error {
	// ...
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L1421
func stepFollower(r *raft, m pb.Message) error {
	// ...
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L1376
func stepCandidate(r *raft, m pb.Message) error {

}

// --------------------------------------------------------

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L243
type raft struct { // raft 알고리즘을 수행하는 구조체
	// ...
	step stepFunc // 특정 상태의 step 함수를 저장하는 변수
	// ...
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L724
func (r *raft) becomeLeader() { // Leader 상태로 전환하는 함수
	// ...
	r.step = stepLeader // // r.step에 Leader 상태에서 메시지를 처리하는 함수를 등록
	// ...
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L686
func (r *raft) becomeFollower(term uint64, lead uint64) { // Follower 상태로 전환하는 함수
	r.step = stepFollower // r.step에 Follower 상태에서 메시지를 처리하는 함수를 등록
	// ...
}

// -------------------------------------------------------- 

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L847
func (r *raft) Step(m pb.Message) error {
	// 메시지의 term에 따라 내부 상태 알고리즘 수행
	switch {
	case m.Term == 0:
		// local message
	case m.Term > r.Term:
		// ...
	case m.Term < r.Term:
		// ...
	}
	
	switch m.Type { // Election에 관련된 메시지들은 따로 처리
	case pb.MsgHup:
		// ...

	case pb.MsgVote, pb.MsgPreVote:
		// ...
		
	default: // Election을 제외한 모든 타입의 메시지들
		err := r.step(r, m) // r.step에 등록된 step 함수 호출
		if err != nil {
			return err
		}
	}
	return nil
}
```

함수형 프로그래밍 언어에서는 함수를 변수, 값으로 사용할 수 있다. `stepLeader`, `stepFollower`, `stepCandidate` 와 같은 함수들을 변수에 등록하고 raft.Step에서 자동으로 알맞은 함수를 호출하도록 했다. 

이런 문법은 `tick`을 구현하는 곳에도 적용되었다. Application(raft 모듈 외부)에서 일정 간격마다 raft.Node.Tick() 함수를 호출하도록 되어있는데 이때 raft.Node.Tick() 또한 같은 방법으로 `tickElection`(Follower, Candidate 상태일 때 등록), `tickHeartbeat`(Leader 상태일 때 등록) 를 호출하도록 되어있다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L686
func (r *raft) becomeFollower(term uint64, lead uint64) {
	// ...
	r.tick = r.tickElection
	// ...
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L645
func (r *raft) tickElection() {
	r.electionElapsed++

	if r.promotable() && r.pastElectionTimeout() {
		r.electionElapsed = 0
		if err := r.Step(pb.Message{From: r.id, Type: pb.MsgHup}); err != nil {
			r.logger.Debugf("error occurred during election: %v", err)
		}
	}
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L724
func (r *raft) becomeLeader() {
	// ...
	r.tick = r.tickHeartbeat
	// ...
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L657
func (r *raft) tickHeartbeat() {
	r.heartbeatElapsed++
	r.electionElapsed++

	// ...

	if r.heartbeatElapsed >= r.heartbeatTimeout {
		r.heartbeatElapsed = 0
		if err := r.Step(pb.Message{From: r.id, Type: pb.MsgBeat}); err != nil {
			r.logger.Debugf("error occurred during checking sending heartbeat: %v", err)
		}
	}
}
```

step 함수는 각 노드 상태의 main 함수라 생각해도 무방하다. step 함수는 switch-case 문, 메시지 타입에 따라 처리할 로직들로 구성되어있다. 각 case문을 한번에 살펴보는 것은 너무 복잡해지기 때문에 raft 프로토콜의 핵심적인 흐름을 하나하나 뜯어서 살펴보려 한다.

<br>

## raft.Node가 raftpb.Message를 Application에게 전달하는 과정

### Ready를 통한 배치 처리
raft 프로토콜 구현을 위한 핵심 로직을 보기전에 살펴볼 것이 하나 남아있다. 바로바로 raft 모듈이 peer 노드로 전달할 메시지를 Application에게 전달하는 방법이다. 메시지들은 앞서 설명했던 것처럼 `버퍼에 저장`되어있다가 Node.Ready() 을 통해 `배치형식으로 전달`된다. 이에 관련된 함수들을 살펴보자.

<img src="https://user-images.githubusercontent.com/44857109/112741348-efd26280-8fbf-11eb-90e8-88115b2e007d.png" width="70%" height="70%">


```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L243
type raft struct { // raft 프로토콜의 핵심 로직을 처리하는 object
	// ...
	msgs []pb.Message // 외부로 전달할 메시지 버퍼
	// ...
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L386
func (r *raft) send(m pb.Message) {
	if m.From == None {
		m.From = r.id
	}
	// ...
	r.msgs = append(r.msgs, m) // 버퍼에 메시지 추가
}

// -------------------------------------------------------- 

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/node.go#L564
func newReady(r *raft, prevSoftSt *SoftState, prevHardSt pb.HardState) Ready {
	rd := Ready{
		Entries:          r.raftLog.unstableEntries(),
		CommittedEntries: r.raftLog.nextEnts(),
		Messages:         r.msgs, // 배치로 전달할 object에 r.msgs 넣기
	}
	// ...
	return rd
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/rawnode.go#L140
func (rn *RawNode) acceptReady(rd Ready) {
	// 배치로 전달한 메시지들이 Application에서 모두 처리되었다면 (Application Loop에서 Node.Advance 함수 호출)

	// ...

	rn.raft.msgs = nil // 메시지 버퍼 비우기
					   
}
```

raft.raft object에는 다른 peer 노드에게 전송할 메시지들을 임시로 저장해두는 `버퍼(r.msgs []raftpb.Message)`가 있다. raft.Step을 통해 여러 메시지를 처리하는 도중에 특정한 작업을 수행하기 위해 메시지를 전송해야 하는 경우(ex: 선거에서 투표를 요청할 때) `send 함수`를 통해 버퍼에 메시지를 추가한다.

 버퍼에 저장된 메시지들은 배치 형식으로 `Ready 구조체`에 담겨 Application으로 전달되는데 이때 Ready.Messages 필드에 메시지들을 담아서 전달한다. 이후 Application Loop에서 Ready에 대한 모든 작업(안정적인 저장소에 메타데이터 저장, CommitedEntries 적용, Snapshot 적용, 다른 peer에게 메시지 전송 등)을 수행하고 다음 배치를 받기 위해 Node.Advance() 를 호출하면 `acceptReady()` 를 통해서 메시지 버퍼가 비워지는 방식이다.

raft.Node가 배치형식으로 데이터를 Application에 전달하고 외부에서 들어오는 메시지를 처리하는 내부 로직은 차근차근 빌드업을 모두 마친뒤에 설명하려고 한다. 

<br>

### send 헬퍼 함수

메시지를 전송할 때 send 함수를 이용하는 것을 보았지만 이 함수만으로는 불편한 것이 있다. 예를 들어 모든 peer에게 특정 메시지를 전달하는 작업은 많이 반복되는 작업이기 때문에 이를 도와주는 몇가지 함수들이 있다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L423
func (r *raft) sendAppend(to uint64) { // appendEntries 작업을 래핑
	r.maybeSendAppend(to, true)
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L495
func (r *raft) sendHeartbeat(to uint64, ctx []byte) { // heartbeat 작업을 래핑
	commit := min(r.prs.Progress[to].Match, r.raftLog.committed)
	m := pb.Message{
		To:      to,
		Type:    pb.MsgHeartbeat,
		Commit:  commit,
		Context: ctx,
	}
	r.send(m)
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L515
func (r *raft) bcastAppend() { // 모든 peer 노드들에게 appendEntires 작업 수행
	r.prs.Visit(func(id uint64, _ *tracker.Progress) {
		if id == r.id {
			return
		}
		r.sendAppend(id)
	})
}


// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L534
func (r *raft) bcastHeartbeatWithCtx(ctx []byte) { // 모든 peer 노드들에게 heartbeat 작업 수행
	r.prs.Visit(func(id uint64, _ *tracker.Progress) {
		if id == r.id {
			return
		}
		r.sendHeartbeat(id, ctx)
	})
}
```

위와 같이 `appendEntries`, `heartbeat` 작업과 브로드캐스트하는 작업은 한번더 래핑이 되어있다.

지금까지 etcd의 rafthttp(Transport 계층), raftpb.Message의 구조, raftpb.Message가 모듈 내부로 전달되고 생성되는 과정을 살펴보았다. 이제 이 메시지들에 의해 raft 프로토콜이 작동하는 흐름만 알아보면 된다. 모든 메시지를 살펴보기엔 무리가 있기 때문에 `리더 선출`(MsgHup, MsgVote, MsgVoteResp)과 `로그 복제`(MsgProp, MsgApp, MsgAppResp, MsgHeartbeat, MsgHeartbeatResp)만 흐름에 따라 분석해보았다.

<br>

# raft 알고리즘

## Leader 선출 처리 과정

### 1. ElectionTimeout 발생
모든 raft 노드는 Follower 상태로 시작한다. Leader가 발견되지 않는 상태에서 주기적으로 raft.Node.Tick()에 의해 raft.tickElection() 이 호출되면, Follower는 결국 `ElectionTimeout`이 발생하고 Candidate 상태로 올라가기 위해 `MsgHup` 메시지를 생성하게 된다. 

> etcd raft 라이브러리는 ElectionTimeout을 물리적인 시간에 따라 직접적으로 호출하지 않고 `ElectionElapsed` 라는 `논리적인 시간 개념`을 통해 발생시킨다. 주기적으로 호출되는 tick 함수에서 ElectionElapsed를 증가시키고 이때 증가된 값이 일정 값보다 크다면 Timeout으로 판단하는 방식이다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L645
func (r *raft) tickElection() {
	r.electionElapsed++

	if r.promotable() && r.pastElectionTimeout() {
		r.electionElapsed = 0
		r.Step(pb.Message{From: r.id, Type: pb.MsgHup})
	}
}
```

<br>

### 2. Step 함수에서 hup 함수 호출
raft.Step 함수는 메시지가 생성되었던의 Term과  시점그 메시지를 처리하는 노드의 Term을 비교해서 상황에 따라 몇가지 작업을 한뒤에 step 함수들(stepLeader, stepCandidate, stepFollower)을 통해 특정한 작업을 수행한다. 이때 step 함수로 처리하지 않는 예외적인 메시지가 있는데 MsgHup(campaign 시작하는 메시지), MsgVote(투표를 요청하는 메시지)이다. 

raft.hup 함수는 현재 자신의 노드가 Candidate가 될 수 있는 상태인지 확인하는 과정을 거치고 최종적으로 campaign 함수를 호출한다. 만약 자신의 로그에서 커밋되었지만 아직 state-machine에 적용되지 않은 entries중에 configChange 가 있다거나 snapshot을 적용중이라면 선거를 시작할 수 없다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L847
func (r *raft) Step(m pb.Message) error {
	switch {
	case m.Term == 0:
		// local message
	case m.Term > r.Term:
		// ...
	case m.Term < r.Term:
		// ...
	}	

	switch m.Type {
	case pb.MsgHup:
		if r.preVote {
			r.hup(campaignPreElection)
		} else {
			r.hup(campaignElection) // <=== hup 함수 호출!
		}

	case pb.MsgVote, pb.MsgPreVote: // 해당 노드에게 투표를 할지 안할지 결정한 후에 MsgVoteResp 메시지 전송
		// ...

	default:
		err := r.step(r, m)
		if err != nil {
			return err
		}
	}
	return nil
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L760
func (r *raft) hup(t CampaignType) {
	// ...
	// 현재 이 노드가 선거를 시작할 수 있는 상태인지 검사
	// 1. 이미 리더인 경우
	// 2. 적용할 snapshot이 있는 경우
	// 3. 복제 대기중인 로그에 ConfChange entry가 있는 경우  
	// 선거 시작을 거부
	// ...

	r.campaign(t) // campaign 시작
}
```

<br>

### 3. campaign 함수 동작
campaign 함수의 동작은 어렵지 않다. 노드의 상태를 Candidate로 전환하고 자신에게 먼저 투표한 다음, 투표할 수 있는 노드들에게 투표를 요청하는 메시지를 전송한다. 

> etcd raft 라이브러리는 preVote, vote 두가지 기능을 모두 지원하지만 preVote는 안보고 넘어가려 합니다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L785
func (r *raft) campaign(t CampaignType) {
	// ...
	// 안전을 위해 적용해야 할 snapshot 이 있는지 한번더 검사 

	var term uint64
	var voteMsg pb.MessageType
	if t == campaignPreElection {
		// ...
	} else {
		r.becomeCandidate() // Candidate 상태로 전환 (tick, step 함수 등록, 새로운 term에 맞게 노드 상태 초기화)
		voteMsg = pb.MsgVote
		term = r.Term
	}

	// 먼저 자신에게 투표함 (r.poll 함수는 다음에 설명함)
	if _, _, res := r.poll(r.id, voteRespMsgType(voteMsg), true); res == quorum.VoteWon {
		// 클러스터에 노드가 자신밖에 없는 경우 MsgVote 를 전송하고 
		// MsgVoteResp 를 기다리는 작업을 수행할 필요가 없음. 바로 다음 작업으로 이동함.
		if t == campaignPreElection {
			r.campaign(campaignElection)
		} else {
			r.becomeLeader() // Leader 상태로 전환
		}
		return
	}

	// 현재 클러스터에 있는 노드들중에 투표할 수 있는 노드들의 id를 구함
	var ids []uint64
	{
		idMap := r.prs.Voters.IDs()
		ids = make([]uint64, 0, len(idMap))
		for id := range idMap {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	}

	// 노드들에게 MsgVote 메시지 전송. 이후 해당 노드들은 MsgVoteResp를 이 노드에게 전송함.
	// MsgVoteResp를 받은 이 노드는 받은 투표수를 확인한 후 Leader 상태로 올라갈지 결정함.
	for _, id := range ids {
		if id == r.id {
			continue
		}
		// ...
		r.send(pb.Message{Term: term, To: id, Type: voteMsg, Index: r.raftLog.lastIndex(), LogTerm: r.raftLog.lastTerm(), Context: ctx})
	}
}
```

이 라이브러리는 Transport 계층과 분리되어있기 때문에 모든 네트워크 작업은 비동기로 작동한다. 노드는 이후에 MsgVote 메시지들이 Node.Ready()를 통해 Application으로 전달되고, Transport 계층을 통해 다른 노드로 전달되고, MsgVoteResp 메시지를 네트워크 계층을 통해 전달받을 때까지 기다리지 않는다. 그냥 이어서 다른 메시지를 처리하고 있다가 나중에 MsgVoteResp 메시지가 도착하면 그떄 메시지를 알맞게 처리할 뿐이다. 이러한 작업이 오히려 로직의 복잡성을 줄이고 동시성 처리를 쉽게 해주는 것 같다.

<br>

### 4. MsgVote를 받은 다른 노드들의 동작
raft 라이브러리를 사용하는 Application은 Transport 계층을 통해 peers로부터 받은 raftpb.Message를 라이브러리 내부 로직에게 전달할 의무가 있다. 앞서서 이 메시지는 raft.Node 내부에서 raft.Step 에 의해 처리되는 것을 확인했기 때문에 raft.Step에서 해당 메시지를 처리하는 것만 보면 된다. 

MsgVote의 처리 동작도 복잡하지 않다. 자신이 현재 Term에서 누구에게 투표했는지 현황과 Candidate 로그의 진행 상태를 확인하고, MsgVoteResp 메시지를 reject 여부 정보를 담아서 전송한다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L847
func (r *raft) Step(m pb.Message) error {
	// ...

	switch m.Type {
	case pb.MsgHup:
		// ...

	case pb.MsgVote, pb.MsgPreVote:
		// 해당 노드에게 투표할 수 있는지 확인
		canVote := r.Vote == m.From || // 이미 해당 노드에게 투표를 한 경우(raft의 모든 네트워크 요청을 멱등성을 보장해야 함)
			(r.Vote == None && r.lead == None) || // 아직 아무에게도 투표하지 않았고 현재 Term에서 따로 알고있는 Leader가 없는 상태
			(m.Type == pb.MsgPreVote && m.Term > r.Term)

		// Candidate의 로그가 자신보다 최신인지 검사후 해당 노드에게 투표함
		if canVote && r.raftLog.isUpToDate(m.Index, m.LogTerm) {
			// ...

			// Candidate 노드에게 grant 의미로 MsgVoteResp 전송
			r.send(pb.Message{To: m.From, Term: m.Term, Type: voteRespMsgType(m.Type)})
			if m.Type == pb.MsgVote {
				// Only record real votes.
				r.electionElapsed = 0
				r.Vote = m.From
			}
		} else {
			// ...

			// 투표할 수 없다면 reject 의미로 MsgVoteResp 전송
			r.send(pb.Message{To: m.From, Term: r.Term, Type: voteRespMsgType(m.Type), Reject: true})
		}

	default:
		err := r.step(r, m)
		if err != nil {
			return err
		}
	}
	return nil
}
```

<br>

### 5. MsgVoteResp를 받은 Candidate 노드의 동작
이전에 campaign 함수를 통해 보냈던 MsgVote에 대한 응답 메시지가 도착했다면 선거 투표 현황을 업데이트하고, 선거 결과에 따라 Leader 상태로 올라간다. 

만약 이 선거에서 이기지 못한 경우, 노드는 Candidate 상태에서 주기적으로 발생되는 tick에 의해 다시 선거를 시작하거나, 선거에서 이긴 다른 노드에게 MsgApp 메시지를 받아 Follower 상태로 내려가게 된다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L1376
func stepCandidate(r *raft, m pb.Message) error {
	// Candidate 선거 타입에 따라 VoteResp 타입 설정
	var myVoteRespType pb.MessageType
	if r.state == StatePreCandidate {
		myVoteRespType = pb.MsgPreVoteResp
	} else {
		myVoteRespType = pb.MsgVoteResp
	}

	switch m.Type {
	// ...
	case myVoteRespType: // MsgVoteResp
		gr, rj, res := r.poll(m.From, m.Type, !m.Reject) // 응답 메시지에 따라 선거 투표 현황 업데이트
		
		switch res {
		case quorum.VoteWon: // 만약 투표에서 이겼다면
			if r.state == StatePreCandidate {
				r.campaign(campaignElection)
			} else {
				r.becomeLeader() // Leader 상태로 전환
				r.bcastAppend() // 모든 노드들에게 리더가 된 것을 알리고, 로그를 복제하기 위한 MsgApp 메시지 전송
			}
		case quorum.VoteLost:
			r.becomeFollower(r.Term, None)
		}
	// ...
	return nil
}
```

<br>

## 로그 복제 처리 과정
마지막으로 살펴볼 것은 로그를 복제하는 과정이다. 이 과정은 리더 선출처럼 하나의 줄기로만 처리되지 않고 여러개의 작은 흐름에 의해 처리되기 때문에 좀 복잡하다. 원활한 이해를 위해 여러 흐름중에서도 가장 핵심적인 흐름 하나만 골라서 콜스택을 따라갈 것이다. 로그에 새로운 entry를 추가하고, 복제를 위해 entries를 다른 노드에게 전송하고, 과반수로 복제가 이루어진 entires들이 commit되는 흐름을 살펴보자. 

### 1. 새로운 변동 사항을 raft 클러스터에 제안
새로운 변동 사항을 클러스터에 제안한다는 말은 결국 raft가 관리하는 `로그에 새로운 Entry를 추가`한다는 말이다. 

Application이 client 요청에 따라 복제 로그에 새로운 entry를 추가하고 싶다면, raft.Node.Propose 함수를 통해 MsgProp 메시지를 생성하고 raft.Step 함수에서 처리될 수 있도록 해야한다. 기본적으로 raft 알고리즘에서 데이터 흐름은 Leader -> Follower 로 흐르기 때문에 모든 `Proposal은 리더에서 수행`되어야 한다. 

Entry의 종류는 `EntryNormal`(쓰기 작업), `EntryConfChange`(클러스터 멤버쉽 변경 작업)이 있다. etcd raft 라이브러리의 EntryConfChange 구현은 논문에 서술된 raft 멤버쉽 변경과 차이가 있다. etcd 구현에서 ConfChange는 로그에 추가될 때 적용되지 않고, 해당 `entry가 applied 되었을 때 적용`된다(entry가 커밋되고 Application로 전달되면 App이 멤버쉽 변경 작업을 수행하는 방식). 또한 한번에 하나의 ConfChange만 커밋시키기 위해서 적용되지 않은 ConfChange가 로그에 존재하는 경우에는 새로운 ConfChange entry를 드랍시킨다.

> README Note: 이 접근 방식은 멤버가 2개인 클러스터에서 한 노드를 제거할 때 문제가 발생할 수 있습니다. ConfChange entry의 커밋을 받지 못한 상태로 노드중 하나가 죽으면 클러스터가 더이상 진행할 수 없게 됩니다. 따라서 클러스터의 멤버쉽을 항상 3개 이상으로 사용하는 것이 좋습니다.

> 이전 버전에서는 멤버쉽 변경에서 한번에 한 노드만 추가, 삭제가 가능했지만, `ConfChangeV2`를 통해 한번에 다수의 노드를 추가, 삭제하는 것이 가능해졌습니다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L991
func stepLeader(r *raft, m pb.Message) error {
	switch m.Type {
	// ...
	case pb.MsgProp:
		// ...

		for i := range m.Entries {
			e := &m.Entries[i]

			var cc pb.ConfChangeI
			if e.Type == pb.EntryConfChange {
				// ... 
				// 만약 Entry가 클러스터 멤버쉽 변경이라면 cc에 역직렬화
			}
			// 만약 Entry 가 클러스터 멤버쉽 변경 Entry 라면 이미 진행중인 confChange가 있는지 확인함. 
			// 적용할 수 있다면 pendingConfIndex에 기록. 적용할 수 없다면 해당 Entry를 EntryNormal 으로 바꾸어서 드랍시킴.
			if cc != nil { 
				// ...

				if refused != "" {
					m.Entries[i] = pb.Entry{Type: pb.EntryNormal} // confChange Entry 드랍시킴
				} else {
					// Entry가 commit 되고 실제로 적용될 때까지 시간이 걸리기 때문에
					// 현재 confChange가 진행중이라는 것을 알려야 함. 이 정보를 pendingConfIndex에 기록
					r.pendingConfIndex = r.raftLog.lastIndex() + uint64(i) + 1 
				}
			}
		}

		// unstable 로그에 entries를 추가함
		// 만약 unstable 로그의 용량이 꽉 찼다면 드랍시키고 false 리턴
		// Follower 들의 로그 복제 진행 상황을 확인한 뒤에 commitedIndex 업데이트 
		if !r.appendEntry(m.Entries...) { 
			return ErrProposalDropped
		}

		// 추가한 entries를 복제하기 위해 Follower 들에게 MsgApp 메시지 전송. 
		// 메시지에는 새로운 CommitedIndex, Entries 등이 포함됨.
		r.bcastAppend() 
		return nil
	}

	// ...
}
```

etcd raft 라이브러리는 내부적으로 Follower가 생성한 `MsgProp 메시지를 Leader로 Redirection` 해준다. 따라서 client의 쓰기 요청을 받은 Application가 자신의 raft 모듈의 상태가 Follower인지, Leader 인지 확인할 필요 없이 raft.Node.Propose 함수를 호출하면 된다.

> 단 Candidate 상태에서는 해당 제안이 드랍된다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L1421
func stepFollower(r *raft, m pb.Message) error {
	switch m.Type {
	case pb.MsgProp:
		// 알고있는 Leader가 없거나 Forwarding을 허락하지 않는 경우엔 드랍시킴.
		if r.lead == None { 
			return ErrProposalDropped
		} else if r.disableProposalForwarding {
			return ErrProposalDropped
		}

		// MsgProp 메시지를 그대로 Leader에게 전송
		m.To = r.lead
		r.send(m)
	// ...
	}
	return nil
}
```

### 2. MsgApp을 받은 Follower 노드의 동작
이 동작은 아주 간단하다. 우선 tick에 의해 electionTimeout이 발생하지 않도록 electionElapsed를 초기화한다. 그리고 전달된 Entries를 자신의 로그에 추가할 수 있는지 검사하고, 결과에 따라 로그를 업데이트한다. 만약 Entries가 거부된 경우에는 Leader가 자신의 로그 상태를 잘못 추적하고 있다는 뜻이기 때문에 바로잡을 수 있는 정보 `RejectHint`를 MsgAppResp에 담아서 전송한다.

이어서 Leader의 CommittedIndex를 자신의 로그에 적용한다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L1421
func stepFollower(r *raft, m pb.Message) error {
	switch m.Type {
	// ...
	case pb.MsgApp:
		r.electionElapsed = 0 // electionTimeout 초기화
		r.lead = m.From
		r.handleAppendEntries(m)
	// ...
	}
	return nil
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L1475
func (r *raft) handleAppendEntries(m pb.Message) {
	if m.Index < r.raftLog.committed {
		r.send(pb.Message{To: m.From, Type: pb.MsgAppResp, Index: r.raftLog.committed})
		return
	}

	// m.Index, m.LogTerm은 m.Entries(추가할 entries)의 바로 전 entry의 index, term 정보이다.
	// 로그 복제에 있어서 같은 index 위치의 entry 두쌍의 term이 같으면 일치하는 것으로 판단한다.
	// 즉 Follower의 로그에서 m.Index 위치의 entry의 Term이 m.LogTerm과 같다면 m.Entries를 추가할 수 있지만,
	// 같지 않다면 m.Entries를 추가할 수 없다.
	// 추가적으로 Follower의 로그가 m.Index 보다 뒤쳐저 있는 경우에도 m.Entries를 추가할 수 없다.
	if mlastIndex, ok := r.raftLog.maybeAppend(m.Index, m.LogTerm, m.Commit, m.Entries...); ok {
		r.send(pb.Message{To: m.From, Type: pb.MsgAppResp, Index: mlastIndex})
	} else {

		// Follower의 로그 상황이 Leader가 추적하고 있는 정보(NextIndex or MatchIndex)와 달라서 MsgApp 요청을 거부해야 할 때,
		// Leader가 자신의 로그 상태를 빠르게 바로잡게 하기 위해 자신이 원하는 Index 정보를 힌트로 전달한다.
		// 이러한 hintIndex가 어떤 상황에서 어떤 방식으로 구해지는지는 뒤에서 설명한다.
		hintIndex := min(m.Index, r.raftLog.lastIndex())
		hintIndex = r.raftLog.findConflictByTerm(hintIndex, m.LogTerm)
		hintTerm, err := r.raftLog.term(hintIndex)
		if err != nil {
			panic(fmt.Sprintf("term(%d) must be valid, but got %v", hintIndex, err))
		}

		// Reject와 함께 RejectHint를 전달한다.
		r.send(pb.Message{
			To:         m.From,
			Type:       pb.MsgAppResp,
			Index:      m.Index,
			Reject:     true,
			RejectHint: hintIndex,
			LogTerm:    hintTerm,
		})
	}
}
```

<br>

### 3. MsgAppResp을 받은 Leader 노드의 동작
Leader는 로그를 복제하기 위해 전송했던 메시지에 대한 응답 메시지를 받는다. Follower의 로그 상태에 따라 복제가 정상적으로 이루어질 수도 있고 복제 거부 메시지를 받을 수도 있다. Follower 측에서 `복제를 거부`했다는 뜻은 Leader가 Follower의 `로그 복제 상태를 잘못 파악`하고 있다는 뜻이기 때문에 이를 바로잡는 작업이 필요하다. 이 작업이 길어질 수록 네트워크 비용이 낭비되기 때문에 최대한 빨리 수정되어야 한다. 이 작업을 최적화하기 위해 Follower와 Leader가 모두 참여한다. 그 방법은 뒤에서 설명한다.

Follower로부터 로그를 정상적으로 복제했다는 응답을 받으면 해당 Follower의 MatchIndex를 증가시키고 CommittedIndex를 증가시킬 수 있는지 검사한다. 만약 CommittedIndex가 업데이트되면 이를 알리기 위해 bcastAppend 함수를 호출한다.

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/raft.go#L991
func stepLeader(r *raft, m pb.Message) error {
	// ...

	pr := r.prs.Progress[m.From] // Follower 로그 복제 상태를 추적하는 object
	if pr == nil {
		r.logger.Debugf("%x no progress available for %x", r.id, m.From)
		return nil
	}

	switch m.Type {
	case pb.MsgAppResp:
		pr.RecentActive = true

		if m.Reject {
			nextProbeIdx := m.RejectHint
			if m.LogTerm > 0 {
				// 복제를 위해 전달한 Entries가 거부되었다는 것은 Follower의 상태를 잘못 알고있다는 것이다.
				// (이 상황은 선거에서 이긴 Leader가 NextIndex 정보를 자신의 로그를 기준을 초기화하는 상황에서 
				// 발생됨. 또한 오랬동안 네트워크에 문제가 있던 Follower에게도 발생될 수 있음.
				// NextIndex를 옳바른 값으로 수정하기위해 여러번 MsgApp을 전송하면 네트워크 비용이 낭비되기 때문에
				// Leader는 이를 최대한 빠르게 수정해야 할 책임이 있음.)
				// 
				// Follower에서 이를 위해 전달한 RejectHintIndex, RejectHintTerm와 자신의 로그 정보를 바탕으로
				// 옳바른 NextIndex를 빠르게 구해낼 수 있다. 이러한 작업을 수행하는 방법은 뒤에서 예제와 함께 설명한다.
				nextProbeIdx = r.raftLog.findConflictByTerm(m.RejectHint, m.LogTerm)
			}

			// 만약 Follower에게 복제해야할 Entry가 이미 Snapshot에 포함되어 전달할 수 없을 때,
			// Follower의 추적 상태를 Probe로 내리고 Snapshot을 전송하기 위해 sendAppend 함수를 호출한다.
			if pr.MaybeDecrTo(m.Index, nextProbeIdx) {
				if pr.State == tracker.StateReplicate {
					pr.BecomeProbe()
				}
				r.sendAppend(m.From)
			}
		} else {
			// 해당 Follower에게 Snapshot을 전달하고 있었거나, 너무 많은 
			// Entries를 전송한 상태라서 로그 복제를 멈추고 있었는지 확인
			oldPaused := pr.IsPaused()

			// Follower의 로그 복제 상태(MatchIndex)를 업데이트
			if pr.MaybeUpdate(m.Index) {
				switch {
				case pr.State == tracker.StateProbe:
					pr.BecomeReplicate()

				case pr.State == tracker.StateSnapshot && pr.Match >= pr.PendingSnapshot:
					// Snapshot을 전달했던 Follower에게 MsgAppResp를 받으면 Snapshot 적용이 끝났다고 판단함.
					// 로그 복제를 이어서 하기 위해 Replicate 상태로 올림.
					pr.BecomeProbe()
					pr.BecomeReplicate()

				case pr.State == tracker.StateReplicate:
					// flow control을 위한 작업 수행
					pr.Inflights.FreeLE(m.Index)
				}

				// 로그 복제가 이루어졌기 때문에 CommittedIndex를 증가시키기 위해 maybeCommit 함수 호출
				// 만약 CommittedIndex가 없데이트되었다면 그 정보를 모든 Follower에게 전달함.
				if r.maybeCommit() {
					releasePendingReadIndexMessages(r)
					r.bcastAppend()
				} else if oldPaused {
					// 로그 복제를 하고있지 않았던 Follower의 경우 이전의 CommittedIndex 업데이트를
					// 전달받지 않았을 수 있기 때문에 CommittedIndex를 전달함.
					r.sendAppend(m.From)
				}

				for r.maybeSendAppend(m.From, false) {
				}

				// ...
			}
		}
	// ...
	}
	return nil
}
```

위 코드의 `pr.BecomeReplicate, pr.BecomeProbe`을 보면 pr(Progress: Follower의 로그 복제 상황을 추적하는 역할)가 몇가지 상태를 갖고있는 것을 알 수 있다. 이것은 특정 Follower에게 전송하는 로그 복제의 속도(Snapshot을 적용중이거나 Flow Control을 조절하기 위한 경우 등)를 제어하기 위해 추가한 etcd의 추가적인 구현이다. Progress 상태는 `Replicate`, `Probe`, `Snapshot` 상태를 갖고 이에 대한 StateMachine은 다음과 같다.

<img src="https://user-images.githubusercontent.com/44857109/112720119-dc7fb280-8f3f-11eb-9492-f1b2038d14b9.png" width="100%" height="100%">

- Probe: HeartbeatTimeout 간격동안 최대 하나의 복제 메시지를 전송합니다. Leader는 해당 노드의 실제 진행 상황을 정확하게 알지 못하기 때문에 실제 진행 상황을 조사할 때까지 최대한 느리게 복제 메시지를 전송합니다.
- Replicate: 복제 메시지를 보낼 때마다 낙관적으로 다음에 보낼 로그의 크기를 증가시킵니다. Follower에게 로그 항목을 빠르게 복제하기 위한 상태입니다.
- Snapshot: 이 상태일 때 Leader는 Follower에게 복제 메시지를 전송하지 않습니다.

<br>

### 4. Leader가 Follower의 로그 복제 상황을 빠르게 수정하는 방법

새로 선출된 Leader는 Follower들의 진행 상태를 `MatchIndex = 0`, `NextIndex = lastLogIndex`로 설정합니다. Follower의 상태를 알지 못하기 때문에 Progress를 StateProbe 상태로 설정해두고 MsgHeartbeat 메시지를 보내면서 진행 상황을 조사합니다. 이때 단순하게 NextIndex를 일정 수준씩 감소시키면서 찾게 되면 로그 상태에 따라 너무 많은 시간이 걸릴 수 있습니다. ETCD 구현은 Follower의 `RejectHint`와 Leader의 `findConflictByTerm`을 통해 최대 2번의 메시지 교환을 통해 진행 상황을 조사할 수 있습니다.

#### case 1

<img src="https://user-images.githubusercontent.com/44857109/112720943-b3155580-8f44-11eb-8563-c6d332a26377.png" width="100%" height="100%">

위 그림과 같은 상황을 가정해보자.

1. Leader는 `Index=9, Term=5` 으로 로그 복제 메시지를 전송한다. 
2. Follower의 로그는 `Index=6, Term=2`가 최대이기 때문에 이 정보를 RejectHint로 Leader에게 전달한다.
3. Leader는 `NextIndex = 6` 으로 수정하고 `Index=6, Term=5` 으로 로그 복제 메시지를 전송한다. 
4. Follower는 자신의 `Index=6` Entry의 Term과 일치하지 않기 때문에 이 Entry를 드랍하기위해 `Index=5, Term=2`을 RejectHint로 Leader에게 전달한다.
5. 3, 4 반복...

이 경우에서 Leader는 불필요한 로그 복제 메시지를 반복하면서 NextIndex를 6, 5, 4, ..., 1 까지 감소시킬 것이 자명하다. 여기서 한가지 로그의 특성을 이용한다.

- 로그 복제가 Reject되는 상황에서는 항상 Leader의 Term은 Follower의 Term보다 크다.

즉 처음에 받았던 RejectHint인 `Index=6, Term=2`에 있는 정보인 `Term=2`를 이용해서, Leader의 로그 중에서 Term이 2 보다 작거나 같은 Index를 선택하면(`Index=1, Term=1`) 빠르게 Probe를 성공시킬 수 있다. 수정된 상황에서 로그 복제 진행은 다음과 같다.

1. Leader는 `Index=9, Term=5` 으로 로그 복제 메시지를 전송한다. 
2. Follower의 로그는 `Index=6, Term=2`가 최대이기 때문에 이 정보를 RejectHint로 Leader에게 전달한다.
3. Leader는 `Term=2` 보다 작거나 같은 Entry인 `Index=1, Term=1`을 찾고 이 정보로 로그 복제 메시지(`Index=[2,9]인 Entries`)를 전송한다.
4. 즉-시 성공

Leader의 `findConflictByTerm`는 이러한 로직으로 수행된다.

<br>

#### case 2
`case 1`처럼 Leader만 노력을 해서는 완전하게 최적화할 수 없다. Follower도 노력을 해야한다.

<img src="https://user-images.githubusercontent.com/44857109/112721577-ead1cc80-8f47-11eb-9d33-833fcadb11dd.png" width="100%" height="100%">

위 그림과 같은 상황을 가정해보자.

1. Leader는 `Index=9, Term=7` 으로 로그 복제 메시지를 전송한다. 
2. Follower의 로그는 `Index=9` Entry의 Term이 6 이기 때문에 로그 복제가 일치하지 않다고 판단하고, `Index=9, Term=6`를 RejectHint로 Leader에게 전달한다.
3. Leader는 `findConflictByTerm`을 이용해서 `Term=6` 보다 작거나 같은 Entry인 `Index=8, Term=3`을 찾고 이 정보로 로그 복제 메시지를 전송한다.
4. Follower의 로그는 `Index=8` Entry의 Term이 5 이기 때문에 로그 복제가 일치하지 않다고 판단하고, `Index=8, Term=5`를 RejectHint로 Leader에게 전달한다.
5. Leader는 `findConflictByTerm`을 이용해서 `Term=5` 보다 작거나 같은 Entry인 `Index=7, Term=3`을 찾고 이 정보로 로그 복제 메시지를 전송한다.
6. 4, 5 반복...

이처럼 Leader의 노력에도 불구하고 옳바른 NextIndex를 찾기위해 Reject 당할 로그 복제 메시지를 반복해서 전송하게 된다. `3`의 상황에서 Leader가 보낸 로그 복제 메시지를 통해 Follower는 'Leader의 로그에서 `Index=8` Entry의 Term이 3 이기 때문에 그 이전의 Entries 또한 Term이 3보다 같거나 작을 것' 이라는 정보를 얻을 수 있다. 따라서 Follower는 자신의 로그에서 Term이 3보다 작거나 같은 Entry(`Index=3, Term=3`)를 RejectHint로 사용하는 방식으로 최적화할 수 있다. 수정된 상황에서 로그 복제 진행은 다음과 같다.

1. Leader는 `Index=9, Term=7` 으로 로그 복제 메시지를 전송한다. 
2. Follower의 로그는 `Index=9` Entry의 Term이 6 이기 때문에 로그 복제가 일치하지 않다고 판단하고, `Index=9, Term=6`를 RejectHint로 Leader에게 전달한다.
3. Leader는 `findConflictByTerm`을 이용해서 `Term=6` 보다 작거나 같은 Entry인 `Index=8, Term=3`을 찾고 이 정보로 로그 복제 메시지를 전송한다.
4. Follower는 자신의 로그에서 `Term=3` 보다 작거나 같은 Entry인 `Index=3, Term=3`를 RejectHint로 Leader에게 전달한다.
5. Leader는 `Index=3, Term=3`으로 로그 복제 메시지(`Index=[4,9]인 Entries`)를 전송한다.
6. 즉-시 성공

<br>

## raft.Node에서 채널 이벤트 기반으로 오케스트레이션 하기
raft 프로토콜의 핵심 로직을 구현한 `raft.go`을 읽으면 동시성 처리를 위한 Lock 처리가 하나도 없는 것을 알 수 있다. 즉 raft.go에 있는 코드들은 동시에 실행되지 않는다. 하지만 첫 부분에서 설명한 것처럼 raft.Node object는 외부로 raftpb.Message를 전달하면서, 주기적으로 Tick에 대한 기능도 수행해야 하고, client의 요청에 따라 새로운 제안 사항을 로그에 추가시켜야 한다. etcd 구현은 이 모든 작업을 하나의 Goroutine에서 채널 이벤트 기반으로 수행한다. 

### 이벤트 루프

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/raft/node.go#L303
func (n *node) run() {
	var propc chan msgWithResult
	var readyc chan Ready
	var advancec chan struct{}
	var rd Ready

	r := n.rn.raft
	lead := None

	for {
		if advancec != nil { 
			// Application Loop에서 아직 Node.Advance()를 호출하지 않은 경우
			// 즉 App이 아직 이전에 보냈던 Ready를 모두 수행하지 않아서 다음 Ready를 받을 준비가 안된 경우
			// readyc = nil 을 통해서 case readyc <- rd: 이 수행되지 않도록 한다.
			readyc = nil
		} else if n.rn.HasReady() {
			// Application Loop에서 다음 Ready를 받을 준비가 되었고
			// raft 모듈 또한 외부로 전달할 변동사항이 있는 경우
			// readyc = n.readyc 을 통해 외부로 Ready를 전달할 수 있도록 한다.
			rd = n.rn.readyWithoutAccept()
			readyc = n.readyc
		}

		// readyc와 같은 메커니즘으로 propc를 적절한 값으로 설정한다.
		if lead != r.lead {
			if r.hasLeader() {
				// ...
				propc = n.propc
			} else {
				propc = nil
			}
			lead = r.lead
		}

		// 이벤트 처리 select 문
		select {
		case pm := <-propc:
			m := pm.m
			m.From = r.id
			err := r.Step(m)
			if pm.result != nil {
				pm.result <- err
				close(pm.result)
			}
		case m := <-n.recvc:
			if pr := r.prs.Progress[m.From]; pr != nil || !IsResponseMsg(m.Type) {
				r.Step(m)
			}
		case cc := <-n.confc:
			// ...
		case <-n.tickc:
			n.rn.Tick()
		case readyc <- rd:
			n.rn.acceptReady(rd)
			advancec = n.advancec
		case <-advancec:
			n.rn.Advance(rd)
			rd = Ready{}
			advancec = nil
		case c := <-n.status:
			c <- getStatus(r)
		case <-n.stop:
			close(n.done)
			return
		}
	}
}
```

> `go-ethereum`도 메인 루프가 이런 형식으로 되어있는 것으로 알고있다.


이벤트 루프에서 `readyc`만 추가적으로 보고 넘어가야 한다. 상식적으로 외부로 Ready를 보낼 준비가 되었다면 바로 `n.readyc <- rd` 를 수행해서 바로바로 Ready를 전달해야 한다. 하지만 위 코드에서는 `readyc = n.readyc` 로 그저 select-case 문을 통해 Ready를 보낼 수 있도록 준비만 하고 있다. 따라서 채널을 준비했다고해서 해당 루프에서 바로 Ready가 보내지는 것이 아니다. 이것은 Ready가 한번 보내질 때 최대한 덜 자주, 많은 정보를 포함한 상태로 App에 전달하기 위해서 일부러 이렇게 설계한 것이다.

### Chan을 이용한 몇가지 패턴
진짜 마지막으로 raft.Node.run의 이벤트 루프에 사용된 몇가지 패턴을 살펴보자.

#### select case
select-case 문에는 수신 채널 작업 이외에도 송신 채널 작업을 등록할 수 있다. 만약 다른 모든 case문들의 수신 채널이 준비되지 않았을 때, 송신하는 case문이 실행된다. 이때 송신할 채널이 nil이라면 해당 case문은 실행되지 않는다.

```go
func main() {
	nodeReq := make(chan string) // 실제로 Application에 전달할 채널
	var req chan string          // nodeReq를 가리킬 채널
	isReady := false

	go func() {
		readyTick := time.NewTicker(3 * time.Second) // 3초마다 nodeReq 채널 준비
		for range readyTick.C {
			isReady = true
		}
	}()

	go func() {
		work := time.NewTicker(2 * time.Second) // 2초마다 특정한 작업 수행

		for {
			// ready 상태에 따라 req 채널 초기화
			if isReady {
				req = nodeReq
			} else {
				req = nil
			}

			select {
			case <-work.C:
				log.Println("working!")

			case req <- "send Ready": // case <-work.C 를 수행할 수 없을 때 수행, req가 nil이면 실행되지 않음
				log.Println("send string to out chan!")
				isReady = false
			}
		}
	}()

	getReqChan := func() chan string {
		return nodeReq
	}
	for {
		select {
		case s := <-getReqChan(): // Ready 수신
			log.Println("recv", s)
		}
	}
}
```

```
2021/03/27 21:43:31 App: Start Loop                    // 31초 시작. 아직 req가 준비되지 않아서 req case문이 trigger되지 않음
2021/03/27 21:43:33 Node: working!                     // work.C 채널 수신
2021/03/27 21:43:35 Node: working!
2021/03/27 21:43:35 Node: send Ready to nodeReq chan!  // req가 준비된 후 nodeReq 채널로 "send Ready" 문자열 송신
2021/03/27 21:43:35 App: recv send Ready
2021/03/27 21:43:37 Node: working!
2021/03/27 21:43:37 Node: send Ready to nodeReq chan!
2021/03/27 21:43:37 App: recv send Ready
```

#### chan chan
status 채널의 타입을 보면 `chan chan Status` 타입이다. 이 채널은 `chan Status`을 송수신하는 채널이다. 즉 채널에 채널을 송수신하는 것이다. 2단게 채널을 이용하면 두개의 고루틴의 작업을 동기화 시킬 수있다.

```go
func main() {
	reqChan := make(chan chan string)

	go func() {
		// Node Loop
		for {
			select {
			case req := <-reqChan:
				log.Println("Node: recv request. start process")
				time.Sleep(2 * time.Second) // 2초 동안 작업 수행
				req <- "process done"       // 작업이 끝났다고 알림
			}
		}
	}()

	// App Loop
	for { // 무한 루프로 요청 전송
		log.Println("App: send request")
		done := make(chan string)
		reqChan <- done // chan chan 으로 요청을 보냄
		log.Println("App: recv response ", <-done) // 요청이 모두 처리될 때까지 기다림
	}
}
```

```
2021/03/27 22:03:36 App: send request
2021/03/27 22:03:36 Node: recv request. start process
2021/03/27 22:03:38 App: recv response  process done // App Loop가 실제로 Node Loop와 동기화됨
2021/03/27 22:03:38 App: send request
2021/03/27 22:03:38 Node: recv request. start process
2021/03/27 22:03:40 App: recv response  process done
```

<br>

# 마치면서
사실 상용화된 프로젝트의 코드를 읽고 분석해본 적이 처음이었습니다. 옛날에는 '와 저런 프로그램의 코드는 내가 이해할 수 없을 정도로 수준이 높은 어썸한 코드겠지?' 라고 생각했었는데, 이번 etcd의 raft 라이브러리 코드를 분석하면서 많이 놀랐습니다. 정말 핫하고 좋은 오픈소스 프로젝트일 수록 가독성이 좋은 코드와 친절한 주석, 명확한 실행 로직을 갖고 있다는 것을 알게되었습니다. 역시 좋은 코드를 작성하는 개발자가 되는 것을 힘든 길인가 봅니다.

## 분석글에 더 추가해야 할 내용
코드를 분석하기로 결정하기 이전에, raft를 직접 구현하면서 제일 힘들었던 부분이 동시성 처리, Snapshot 생성, 멤버쉽 변경 부분이었습니다. 논문에도 '어떤 어떤 식으로 수행하면 될거에요~'라고 말만 해주고 정확한 명세는 서술되어있지 않아서 구글링을 해보다가 결국 포기했습니다. 그런데 놀랍게도, 분석하는 글을 1300줄이나 쓰는 와중에 Snapshot, 멤버쉽 변경에 대한 핵심 로직은 설명되어있지 않습니다. Snapshot은 어느정도 분석은 끝났지만 글로 풀어내기에는 아직 이해를 다 하지 못했고, 멤버쉽 변경은 아직 코드도 다 읽지 못했기 때문입니다.

## 감사합니다
이 코드 범벅이인 길고 지루한 글을 읽어주셔서 정말 감사합니다.
