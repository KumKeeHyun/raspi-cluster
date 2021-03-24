# ETCD Raft 모듈 코드 분석
## 서론
### 왜 이 코드를 분석하게 되었나?
코시국으로 갈피를 못잡고 있던 작년에 선배의 소개로 `Kafka`, `Elasticsearch`을 공부하면서 분산 환경을 처음 접했습니다. 여러 머신에서 요청을 분산해서 처리하고 하나의 머신이 동작을 멈춰도 복제된 내용을 통해 내구성을 갖는 시스템이 너무 멋있었고 이런 분산, 복제 시스템에 관심을 갖고 공부하게 되었습니다. 

저는 `Medium` 사이트에서 글을 찾아 읽는 것을 좋아하는데 마침 추천 리스트에 있는 `나는 어떻게 분산 환경을 공부했는가(영어로 쓰여있었음)`라는 글을 읽고 `Raft 합의 알고리즘`을 알게되었습니다. 자연스럽게 Raft에 대해 공부하면서 `In Search of an Understandable Consensus Algorithm-Diego` 논문과 여러 아티클을 읽게 되었습니다. 이후 근자감이 차올라 직접 구현을 해보려다가 Raft가 얼마나 복잡하고 어려운 알고리즘인지만 깨닫고 결국 구현되어있는 코드를 분석해보는 목표를 세웠습니다. 

제가 알고있는 프로젝트중에 Raft를 사용하는 프로젝트는 `Zookeeper`, `ETCD`정도였습니다. Zookeeper는 Kafka를 사용해보면서 친숙함이 있었지만 Kafka 이후 버전에서는 자체적으로 Raft를 구현하고 Zookeeper를 사용하지 않는다고 들어서 미래가 밝아보이지 않다는 생각을 했습니다. 또 Java에 대한 지식이 깊지 않아서 제외했습니다. 이에 반해서 ETCD는 정말 핫한 `Kubernetes`에서 메타데이터를 저장하는 Key-Value Store로 사용하고 있고 ETCD의 Raft 모듈이 cockroachDB같은 다른 프로젝트에서도 사용할만큼 신뢰성이 높고 추상화도 잘되어있는 것 같아서 ETCD 코드를 선택했습니다. 추가적으로 `Golang`으로 구현되어있기 때문에 Go를 공부하고있는 저에게 정말 알맞는 코드라 생각했습니다. 

### 들어가기 전에
저는 대학교 3학년을 수료한 상태이고 글을 써본적이 별로 없는 공대생입니다. 분산 시스템에 대한 강의를 수강한 것도 아니고 인터넷에서 여러 자료를 찾아가면서 공부했기 때문에 `Raft 합의 알고리즘`에 친숙하지 않으신 분들을 위해 개념부터 설명해드리기엔 저의 능력이 부족합니다. 이런 큰 코드를 분석해본 경험도 적기 때문에 읽으시면서 글이 매끄럽지 않더라도 양해 부탁드립니다.

### 간단하게 Raft란?
Raft는 여러 머신에서 복제된 상태를 유지할 수 있게 하는 합의 알고리즘입니다. Raft는 state-machine 자체를 복제하는 것이 아니라 state-machine에 적용할 변동사항을 로그형태로 관리하고 이 로그를 복제합니다. 모든 데이터의 흐름(복제)은 Leader에서 Follower로 흐릅니다. 한 논리적인 클러스터에서 Leader는 1개만 존재하고 Leader가 사라지면 여러 Follower들중 한 Follower가 Election을 시작하면서 Candidate가 되고 선거에서 이기면 최종적으로 Leader가 됩니다.

## ETCD Raft Readme

### Features
먼저 ETCD Raft Readme 문서에 쓰여있는 특징은 다음과 같습니다.

#### 기본적인 Raft 프로토콜 구현
- 리더 선출
- 로그 복제
- 스냅샷
- 클러스터 멤버십 변경
- read-only 쿼리 성능 향상을 위한 처리 방식
    - read-only 쿼리를 leader와 follower 모두 처리
    - leader가 요청을 받으면 quorum을 확인하고 엔트리 로그 연산을 건너 뛰고 쿼리 처리
    - follower가 요청을 받으면 leader로부터 safe-read-index를 확인하고 쿼리 처리

#### 성능 향상을 위한 추가 구현
- 로그 복제 지연을 줄이기 위한 pipelining
- 로그 복제 flow-control
- 네트워크 I/O(raft module 내부 메시지 교환) 부하를 줄이기 위한 배치 처리
- 디스크 I/O(log entries) 부하를 줄이기 위한 배치 처리
- follower가 받은 요청을 내부적으로 leader로 redirection
- leader가 quorum을 잃으면 자동으로 follower로 전환됨
- quorum을 잃었을 때 로그가 무한하게 자라는 것을 방지

대부분의 Raft 구현은 Storage 처리, 로그 메시징 직렬화와 네트워크 전송등을 포함한 Monolithic 디자인을 갖고 있습니다. 대신 ETCD의 Raft 라이브러리는 Raft의 핵심 알고리즘만 구현하여 최소한의 디자인만 따릅니다. (역주: 스토리지, 네트워크 계층은 이 라이브러리를 사용하는 사용자가 구현해야 함.)

### Usage
`etcd/raft/README.md#Usage`에 나와있는 예시 코드가 소스 코드와 다른 점이 있어서 소스 코드를 기준으로 몇가지 수정해서 작성했습니다.

1. Raft의 주요 Object인 Node를 생성, 시작
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

  // ------------------------------------------------
  // Set peer list to the other nodes in the cluster.
  // Note that they need to be started separately as well.
  // ------------------------------------------------
  // 수정: StartNode에서 peer에는 자신의 노드 정보도 함께 넘겨주어야 함. {ID: 0x01} 추가
  n := raft.StartNode(c, []raft.Peer{{ID: 0x01}, {ID: 0x02}, {ID: 0x03}})
```

- 하나의 노드로 구성된 클러스터
```go
  // Create storage and config as shown above.
  // Set peer list to itself, so this node can become the leader of this single-node cluster.
  peers := []raft.Peer{{ID: 0x01}}
  n := raft.StartNode(c, peers)
```

- 기존 클러스터에 새로운 노드를 추가하는 경우
```go
  // Create storage and config as shown above.
  // ------------------------------------------------
  // n := raft.StartNode(c, nil)
  // ------------------------------------------------
  // 수정: 새로운 노드가 클러스터에 추가되는 경우에도 RestartNode를 호출해야함.
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

2. 주기적으로 Node.Ready() 채널을 읽어서 스토리지를 업데이트하거나 네트워크를 통해 다른 노드로 메시지 전송
    1. Entries, HardState, Snapshot 순서대로 영구적인 스토리지에 저장
    2. Messages에 있는 모든 메시지들을 네트워크 계층을 통해 지정된 노드로 전달 
    3. Snapshot이나 CommitedEntries가 있는 경우 state-machine에 적용
    4. Node.Advance()를 호출해서 다음 배치에 대한 준비 상태를 알림

3. 주기적으로 Node.Tick()을 호출해서 HeartbeatTimeout, ElectionTimeout이 발생되도록 함

```go
// Usage-2,3 을 통합한 코드
  for {
    select {
    case <-s.Ticker: // 3. time.Ticker를 사용해서 주기적으로 Node.Tick()을 호출
      n.Tick()
    case rd := <-s.Node.Ready(): // 2. raft 모듈이 스토리지, 네트워크 계층에서 처리할 것들을 배치형식으로 전달
      saveToStorage(rd.HardState, rd.Entries, rd.Snapshot) // 2-1. 특정한 정보를 영구적인 스토리지에 저장
      send(rd.Messages) // 2-2. 네트워크 계층을 통해 지정된 노드로 전달 
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
      s.Node.Advance() // 2-4. 다음 배치(Node.Ready())를 받기 위해 Node.Advance() 호출
    case <-s.done:
      return
    }
  }
```

4. raft 모듈 외부에서 내부로 필요한 메시지를 전달
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

  // 클러스터 구성 변경
  n.ProposeConfChange(ctx, cc)
```

## 네트워크, 스토리지 계층과 raftpb.Message
<img src="https://user-images.githubusercontent.com/44857109/112468548-bd501c00-8dab-11eb-8b63-bf461cde45e4.png" width="70%" height="70%">

- 이 그림은 raft 라이브러리의 핵심 object인 raft.Node가 어떻게 다른 사용자가 구현한 Application(네트워크, 스토리지 계층)과 소통하는지 정리한 그림이다.

소스 코드를 읽기 전에는 리더 선출, 로그 복제 등의 작업에서 노드간 네트워크 통신을 분리하는 정도의 추상화가 가능한지 의문이 들었다. 선거에서 투표를 요청하거나 복제할 Entries를 전달할 때는 peer 노드의 개수만큼 스레드를 생성해서 RPC나 Rest API 등의 네트워크 요청을 보내는 것이 당연하다고 생각했었다. ETCD는 이러한 작업을 추상화 하기 위해 requestVote, appendEntries heartbeat, installSnapshot 등의 네트워크 작업을 raftpb.Message로 나타낼 수 있도록 추상화하였다.

```protobuf
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
	MsgSnapStatus      = 11; // Follower가 Snapshot을 적용하던 도중 오류가 발생하면 Leader에게 알림. 정상적으로 처리된 경우에는 MsgAppResq를 보냄 (Follower -> Leader)
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
- [https://github.com/etcd-io/etcd/blob/master/raft/raftpb/raft.proto#L38](https://github.com/etcd-io/etcd/blob/master/raft/raftpb/raft.proto#L38)

위와 같이 Raft 알고리즘을 구현하기 위한 모든 네트워크 작업과 내부에서 수행해야 하는 작업을 모두 raftpb.Message로 표현할 수 있다. raft.Node가 다른 peer 노드에게 전송할 메시지가 있다면 Message 버퍼에 저장되어있다가 Node.Ready() 채널을 통해 배치형식으로 Application에 전달된다. Application은 이 메시지들을 구현한 네트워크 계층을 통해 실제로 전송하게 된다. 마찬가지로 스토리지에 새로 커밋된 Entries를 적용해야 하거나 Snapshot을 적용해야할 때 unstable한 저장소에 저장해두었다가 Node.Ready() 채널을 통해 배치형식으로 Appication에 전달된다. 

다음 코드는 raft 라이브러리를 활용한 예제인 [raftexample](https://github.com/etcd-io/etcd/blob/master/contrib/raftexample) 코드에서 raftpb.Message를 다른 노드로 전달하기 위해 수행된 콜스택이다. raftexample의 네트워크 계층은 etcd-server을 위한 http 구현체를 사용한다. 

```go
// https://github.com/etcd-io/etcd/blob/master/contrib/raftexample/raft.go#L459
for {
		select {
		case <-ticker.C:
			rc.node.Tick()
		case rd := <-rc.node.Ready():
			// ...
			rc.transport.Send(rd.Messages) // raft.Node가 배치형식으로 전달한 메시지를 네트워크 계층을 통해 전송
			// ...
		case <-rc.stopc:
			rc.stop()
			return
		}
	}

// https://github.com/etcd-io/etcd/blob/master/server/etcdserver/api/rafthttp/transport.go#L175
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

// https://github.com/etcd-io/etcd/blob/master/server/etcdserver/api/rafthttp/peer.go#L236
func (p *peer) send(m raftpb.Message) {
  // ...
	writec, name := p.pick(m) // 실제 POST 요청 전송을 전담하는 고루틴으로 보낼 채널
	select {
	case writec <- m:
	default:
		// ...
	}
}

// https://github.com/etcd-io/etcd/blob/master/server/etcdserver/api/rafthttp/pipeline.go#L92
func (p *pipeline) handle() { // pipeline을 담당하는 고루틴
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

다음 코드는 raftexample에서 네트워크 계층을 통해 전달받은 raftpb.Message를 로컬 노드에서 처리하도록 전달하기 위해 수행된 콜스택이다.

```go
// https://github.com/etcd-io/etcd/blob/master/contrib/raftexample/raft.go#L479
func (rc *raftNode) serveRaft() {
  // ...
	err = (&http.Server{Handler: rc.transport.Handler()}).Serve(ln) // 네트워크 계층을 구현한 HTTP Server 시작
	// ...
}

// https://github.com/etcd-io/etcd/blob/master/server/etcdserver/api/rafthttp/transport.go#L157
func (t *Transport) Handler() http.Handler {
	pipelineHandler := newPipelineHandler(t, t.Raft, t.ClusterID) // http.Handler(ServeHTTP) 인터페이스를 구현한 구조체 
  // ...
	mux := http.NewServeMux()
	mux.Handle(RaftPrefix, pipelineHandler)
  // ...
	return mux
}

// https://github.com/etcd-io/etcd/blob/master/server/etcdserver/api/rafthttp/http.go#L95
func (h *pipelineHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// ...
	limitedr := pioutil.NewLimitedBufferReader(r.Body, connReadLimitByte)
	b, err := ioutil.ReadAll(limitedr) // HTTP Request Body 읽기
	// ...
	var m raftpb.Message
	if err := m.Unmarshal(b); err != nil { // raftpb.Message으로 역직렬화
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

// https://github.com/etcd-io/etcd/blob/master/contrib/raftexample/raft.go#L499
func (rc *raftNode) Process(ctx context.Context, m raftpb.Message) error {
	return rc.node.Step(ctx, m) // raft.Node가 메시지를 처리하도록 호출
}

// https://github.com/etcd-io/etcd/blob/master/raft/node.go#L300
func (n *node) run() {
  // ...
	for {
		// ...
		select {
		// ...
		case m := <-n.recvc:
      // 적절하지 않은 메시지 필터링
			if pr := r.prs.Progress[m.From]; pr != nil || !IsResponseMsg(m.Type) {
				r.Step(m) // 실제 raft 모듈이 메시지를 처리하도록 전달
			}
    // ...
		}
	}
}
```

## raft.Node가 raftpb.Message를 처리하는 방법
이제 raft.Node가 Application 또는 다른 peer 노드와 어떻게 소통하는지 알았으면 이 메시지를 어떻게 처리하는지 알아봐야 한다. 수많은 타입의 raftpb.Message를 처리하는 함수는 raft.raft.Step(msg) 이다. 

> `raft.raft.Step`는 처음 raft(package 이름), 두번째 raft(object 이름)이 같아서 구분하기 위해 저렇게 표기했다. 다음부터는 `raft.Step`으로 표기할 것이다. 


## 

