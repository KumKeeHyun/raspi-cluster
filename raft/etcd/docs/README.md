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
> `etcd/raft/README.md#Usage`에 나와있는 예시 코드가 소스 코드와 다른 점이 있어서 소스 코드를 기준으로 몇가지 수정해서 작성했습니다.

#### 1. Raft의 주요 Object인 Node를 생성, 시작
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

#### 2. 주기적으로 Node.Ready() 채널을 읽어서 스토리지를 업데이트하거나 네트워크를 통해 다른 노드로 메시지 전송
    1. Entries, HardState, Snapshot 순서대로 영구적인 스토리지에 저장
    2. Messages에 있는 모든 메시지들을 네트워크 계층을 통해 지정된 노드로 전달 
    3. Snapshot이나 CommitedEntries가 있는 경우 state-machine에 적용
    4. Node.Advance()를 호출해서 다음 배치에 대한 준비 상태를 알림

#### 3. 주기적으로 Node.Tick()을 호출해서 HeartbeatTimeout, ElectionTimeout이 발생되도록 함

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

#### 4. raft 모듈 외부에서 내부로 필요한 메시지를 전달
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

<br>

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

<br>

## raft.Node가 raftpb.Message를 처리하는 방법
raft.Node가 Application 또는 다른 peer 노드와 어떻게 소통하는지 알아보았으니 이제 이 메시지를 어떻게 처리하는지 살펴봐야 한다. 수많은 타입의 raftpb.Message는 모두 raft.raft.Step(msg) 함수에서 처리된다. 

> `raft.raft.Step`는 처음 raft(package 이름), 두번째 raft(object 이름)이 같아서 구분하기 위해 저렇게 표기했다. 다음부터는 `raft.Step`으로 표기할 것이다. 

다음 코드는 모든 종류의 raftpb.Message가 raft 프로토콜을 수행하기 위해 raft.Step 함수를 호출하는 것을 보여준다.

```go
// https://github.com/etcd-io/etcd/blob/master/raft/node.go#L300
func (n *node) run() {
	// ...
	for {
		// ...
		select {
		case pm := <-propc:
			m := pm.m
			m.From = r.id
			err := r.Step(m) // client write reqeust에 따라 생성된 proposal 메시지 
			// ...
		case m := <-n.recvc:
			if pr := r.prs.Progress[m.From]; pr != nil || !IsResponseMsg(m.Type) {
				r.Step(m) // 네트워크 계층을 통해 전달받은 메시지들
			}
		// ...
	}
}

// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L645
func (r *raft) tickElection() {
	r.electionElapsed++

	if r.promotable() && r.pastElectionTimeout() {
		r.electionElapsed = 0
		r.Step(pb.Message{From: r.id, Type: pb.MsgHup}) // 로컬 노드가 선거를 시작하도록 MsgHup 메시지를 생성한 후 바로 Step 함수를 통해 처리
	}
}
```

메시지가 client에 의해 생성되거나 네트워크 계층에서 전달받거나 같은 상황에 상관없이 모두 raft.Step 함수를 호출한다. 여기서 몇가지 의문이 생길 수 있다. raft 노드는 Leader, Candidate, Follower 상태에 따라 같은 메시지도 다르게 처리해야 하지만 그런 로직은 보이지 않고 무조건 하나의 함수를 호출하도록 추상화 되어있기 때문이다. ETCD는 이런 로직을 구현하기 위해서 각 상태에 따른 step 함수(소문자)를 따로 작성하고 이를 raft.Step 함수로 감싼다. 

다음 코드는 각 상태에 따른 step 함수, 이를 래핑한 raft.Step 함수, raft.Step 함수가 어떻게 노드 상태에 따라 다른 step 함수를 호출하는지 보여준다.

```go
// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L983
type stepFunc func(r *raft, m pb.Message) error // 각 상태에 따른 step 함수의 타입

func stepLeader(r *raft, m pb.Message) error {
	// ...
}

func stepFollower(r *raft, m pb.Message) error {
	// ...
}

// --------------------------------------------------------

// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L243
type raft struct { // raft 프로토콜의 핵심 로직을 처리하는 object
	// ...
	step stepFunc // 특정 상태의 step 함수를 저장하는 변수
	// ...
}

// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L680
func (r *raft) becomeFollower(term uint64, lead uint64) { // Follower 상태로 전환하는 함수
	r.step = stepFollower // r.step에 Follower 상태에서 메시지를 처리하는 함수를 등록
	// ...
}

// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L718
func (r *raft) becomeLeader() { // Leader 상태로 전환하는 함수
	// ...
	r.step = stepLeader // // r.step에 Leader 상태에서 메시지를 처리하는 함수를 등록
	// ...
}

// -------------------------------------------------------- 

// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L841
func (r *raft) Step(m pb.Message) error {
	// ...
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

함수형 프로그래밍 언어에서는 함수를 변수, 값으로 사용할 수 있다. stepLeader, stepFollower, stepCandidate 와 같은 함수들을 변수에 등록하고 raft.Step에서 자동으로 알맞은 함수를 호출하도록 했다. 

이런 문법은 tick을 구현하는 곳에도 적용되었다. Application(raft 모듈 외부)에서 일정 간격마다 raft.Node.Tick() 함수를 호출하도록 되어있는데 이때 raft.Node.Tick() 또한 같은 방법으로 tickElection(Follower, Candidate 상태일 때 등록), tickHeartbeat(Leader 상태일 때 등록) 를 호출하도록 되어있다.

step 함수는 각 노드 상태의 main 함수라 생각해도 무방하다. step 함수는 switch-case 문, 메시지 타입에 따라 처리할 로직들로 구성되어있다. 각 case문을 한번에 살펴보는 것은 너무 복잡해지기 때문에 raft 프로토콜의 핵심적인 흐름을 하나하나 뜯어서 살펴보려 한다.

<br>

## raft.Node가 네트워크 계층을 통해 전송할 raftpb.Message를 Application에게 전달하는 방법
Raft 프로토콜 구현을 위한 핵심 로직을 보기전에 살펴볼 것이 하나 남아있다. 바로바로 raft 모듈이 peer 노드로 전달할 메시지를 Application에게 전달하는 방법이다. 메시지들은 앞서 설명했던 것처럼 버퍼에 저장되어있다가 Node.Ready() 을 통해 배치형식으로 전달된다. 이에 관련된 함수들을 살펴보자.

```go
// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L243
type raft struct { // raft 프로토콜의 핵심 로직을 처리하는 object
	// ...
	msgs []pb.Message // 외부로 전달할 메시지 버퍼
	// ...
}

// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L386
func (r *raft) send(m pb.Message) {
	if m.From == None {
		m.From = r.id
	}
	// ...
	r.msgs = append(r.msgs, m) // 버퍼에 메시지 추가
}

// -------------------------------------------------------- 

// https://github.com/etcd-io/etcd/blob/master/raft/node.go#L559
func newReady(r *raft, prevSoftSt *SoftState, prevHardSt pb.HardState) Ready {
	rd := Ready{
		Entries:          r.raftLog.unstableEntries(),
		CommittedEntries: r.raftLog.nextEnts(),
		Messages:         r.msgs, // 배치로 전달할 object에 r.msgs 넣기
	}
	// ...
	return rd
}

// https://github.com/etcd-io/etcd/blob/master/raft/rawnode.go#L140
func (rn *RawNode) acceptReady(rd Ready) {
	// ...
	rn.raft.msgs = nil // 배치로 전달한 메시지들이 Application에서 모두 처리되었다면 (Application Loop에서 Node.Advance 함수 호출)
					   // 메시지 버퍼 비우기
}
```

raft.raft object에는 다른 peer 노드에게 전송할 메시지들을 임시로 저장해두는 버퍼 필드([]raftpb.Message)가 있다. raft.Step을 통해 여러 메시지를 처리하는 도중에 특정한 작업을 수행하기 위해 메시지를 전송해야 하는 경우(ex: 선거에서 투표를 요청할 때) send 함수를 통해 버퍼에 메시지를 추가한다. 버퍼에 저장된 메시지들은 배치형식으로 Ready 구조체에 담겨 Application으로 전달되는데 이때 Ready.Messages 필드에 메시지들을 담아서 전달한다. 이후 Application Loop에서 Ready에 대한 모든 작업(안정적인 저장소에 메타데이터 저장, CommitedEntries 적용, Snapshot 적용, 다른 peer에게 메시지 전송 등)을 수행하고 다음 배치를 받기 위해 Node.Advance() 를 호출하면 acceptReady() 를 통해서 메시지 버퍼가 비워지는 방식이다.

raft.Node가 배치형식으로 데이터를 Application에 전달하고 외부에서 들어오는 메시지를 처리하는 내부 로직은 차근차근 빌드업을 모두 마친뒤에 설명하려고 한다.

메시지를 전송할 때 send 함수를 이용하는 것을 보았지만 이 함수만으로는 불편한 것이 있다. 예를 들어 모든 peer에게 특정 메시지를 전달하는 작업은 많이 반복되는 작업이기 때문에 이를 도와주는 몇가지 함수들이 있다.

```go
// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L423
func (r *raft) sendAppend(to uint64) { // appendEntries 작업을 래핑
	r.maybeSendAppend(to, true)
}

// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L494
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

// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L515
func (r *raft) bcastAppend() { // 모든 peer 노드들에게 appendEntires 작업 수행
	r.prs.Visit(func(id uint64, _ *tracker.Progress) {
		if id == r.id {
			return
		}
		r.sendAppend(id)
	})
}


// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L534
func (r *raft) bcastHeartbeatWithCtx(ctx []byte) { // 모든 peer 노드들에게 heartbeat 작업 수행
	r.prs.Visit(func(id uint64, _ *tracker.Progress) {
		if id == r.id {
			return
		}
		r.sendHeartbeat(id, ctx)
	})
}
```

위와 같이 appendEntries, heartbeat 와 이 것들을 브로드캐스트하는 작업은 한번더 래핑이 되어있다. 함수 기능 자체는 어려울 것이 없지만 함수형 프로그래밍의 기법이 들어가있어서 한번 짚고 넘어가려 한다.

bcastAppend 함수를 보면 반복문이 아닌 r.prs.Visit 함수를 호출하고 있다. r.prs는 클러스터를 구성하는 peer들을 관리하는 object이다. 로그 복제 진행 상황, 스냅샷 전송 여부 같은 정보를 기록하는 등의 작업을 수행한다. prs.Visit 함수는 모든 peer에 대해서 특정한 작업을 수행하도록 내부에서 반복문을 돌고 수행할 작업을 주입받는다. 모든 peer에게 같은 일을 수행해야 하는 작업를 추상화한 것이다. ETCD는 이러한 추상화를 통해서 bcastAppend, bcastHeartbeat 뿐만 아니라 peer들의 복제 진행 상황 초기화, 클러스터 구성 초기화 등의 작업을 수행한다.

다음 코드는 간단한 예제입니다.

```go
// 실제 코드와는 차이가 있습니다.
type ProgressTracker struct {
	peers map[uint64]*Progress
}

func (prs *ProgressTracker) Visit(task func (id uint64, p *Progress)) {
	for pID, pr := range prs.peers {
		task(pID, pr)
	}
}
```

<br>

## Leader 선출 처리 과정
지금까지 ETCD의 raft 라이브러리가 스토리지, 네트워크 게층과 소통하는 방법, raft 프로토콜 구현에 필요한 로직을 raftpb.Message로 추상화하고 메시지가 처리되기까지의 과정을 살펴보았다. 이제 메시지들에 의해 raft 프로토콜이 작동하는 방법만 알아보면 된다. 모든 메시지를 살펴보기엔 무리가 있기 때문에 리더 선출(MsgHup, MsgVote, MsgVoteResp)과 로그 복제(MsgProp, MsgApp, MsgAppResp, MsgHeartbeat, MsgHeartbeatResp)만 흐름에 따라 살펴볼 것이다.


### 1. ElectionTimeout 발생
모든 raft 노드는 Follower 상태로 시작한다. Leader가 발견되지 않는 상태에서 주기적으로 raft.Node.Tick()에 의해 raft.tickElection() 이 호출되면, Follower는 결국 `ElectionTimeout`이 발생하고 Candidate 상태로 올라가기 위해 `MsgHup` 메시지를 생성하게 된다. 

> ETCD Raft 라이브러리는 ElectionTimeout을 물리적인 시간에 따라 직접적으로 호출하지 않고 `ElectionElapsed` 라는 논리적인 시간 개념을 통해 발생시킨다. 주기적으로 호출되는 tick 함수에서 ElectionElapsed를 증가시키고 이때 증가된 값이 일정 값보다 크다면 Timeout으로 판단하는 방식이다.

```go
// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L645
func (r *raft) tickElection() {
	r.electionElapsed++

	if r.promotable() && r.pastElectionTimeout() {
		r.electionElapsed = 0
		r.Step(pb.Message{From: r.id, Type: pb.MsgHup})
	}
}
```

### 2. Step 함수에서 hup 함수 호출
raft.Step 함수는 메시지가 생성되었던 시점의 Term과 그 메시지를 처리하는 노드의 Term을 비교해서 상황에 따라 몇가지 작업을 한뒤에 step 함수들(stepLeader, stepCandidate, stepFollower)을 통해 특정한 작업을 수행한다. 이때 step 함수로 처리하지 않는 예외적인 메시지가 있는데 MsgHup(campaign 시작하는 메시지), MsgVote(투표를 요청하는 메시지)이다. 

raft.hup 함수는 현재 자신의 노드가 Candidate가 될 수 있는 상태인지 확인하는 과정을 거치고 최종적으로 campaign 함수를 호출한다. 만약 자신의 로그에서 커밋되었지만 state-machine에 적용되지 않은 entries중에 snapshot, configChange 가 있다면 선거를 시작할 수 없다.

```go
// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L917
func (r *raft) Step(m pb.Message) error {
	// Handle the message term, which may result in our stepping down to a follower.
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
			r.hup(campaignElection) // <=== hup 함수 호출!!!!!!
		}

	case pb.MsgVote, pb.MsgPreVote: // 해당 노드에게 투표를 할지 안할지 결정한 후에 MsgVoteResp 메시지 전송
		// ...

	default:
		err := r.step(r, m) // 대부분의 메시지가 여기서 처리됨
		if err != nil {
			return err
		}
	}
	return nil
}

// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L754
func (r *raft) hup(t CampaignType) {
	// ...
	// 현재 이 노드가 선거를 시작할 수 있는 상태인지 검사
	// 로그에 configChange entry나 snapshot entry가 대기중인 경우 선거 시작을 거부

	r.campaign(t) // 실제 선거 시작
}
```

### 3. campaign 함수 동작
campaign 함수의 동작은 어렵지 않다. 노드의 상태를 Candidate로 전환하고 자신에게 먼저 투표한 다음, 투표할 수 있는 노드들에게 투표를 요청하는 메시지를 전송한다. 

> ETCD Raft 라이브러리는 preVote, vote 두가지 기능을 모두 지원하지만 preVote는 안보고 넘어가려합니다.

이 라이브러리는 앞서 설명했던 것처럼 네트워크 계층과 분리되어있기 때문에 모든 네트워크 작업은 비동기적으로 작동한다. 노드는 이후에 MsgVote 메시지들이 Node.Ready()를 통해 Application으로 전달되고, 네트워크 계층을 통해 다른 노드로 전달되고, MsgVoteResp 메시지를 네트워크 계층을 통해 전달받을 때까지 기다리지 않는다. 그냥 다른 메시지를 처리하고 있다가 나중에 MsgVoteResp 메시지가 도착하면 그떄 메시지를 알맞게 처리할 뿐이다. 이러한 작업이 오히려 로직의 복잡성을 줄이고 동시성 처리를 쉽게 해주는 것 같다.

```go
// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L779
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

### 4. MsgVote를 받은 다른 노드들의 동작
raft 라이브러리를 사용하는 Application은 네트워크 계층을 통해 받은 메시지를 라이브러리 내부 로직에게 전달할 의무가 있다. 앞서서 이 메시지는 raft.Node 내부에서 raft.Step 에 의해 처리되는 것을 확인했기 때문에 raft.Step에서 해당 메시지를 처리하는 것만 보면 된다. 

MsgVote의 처리 동작도 복잡하지 않다. 자신이 현재 Term에서 누구에게 투표했는지 현황과 Candidate 로그의 진행상태를 확인하고 MsgVoteResp 메시지를 reject 여부 정보를 담아서 전송한다.

```go
// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L924
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

### 5. MsgVoteResp를 받은 Candidate 노드의 동작
이전에 campaign 함수를 통해 보냈던 MsgVote에 대한 응답 메시지가 도착했다면 선거 투표 현황을 업데이트하고, 선거 결과에 따라 Leader 상태로 올라간다. 

만약 이 선거에서 이기지 못한 경우, 노드는 Candidate 상태에서 주기적으로 tick이 발생하고 른 노드또한 다들로 부터 온 메시지를 처리한다.  tickElection에 의해 다시 선거를 시작하거나, 선거에서 이긴 다른 노드에게 MsgApp 메시지를 받아 Follower 상태로 내려가게 된다.

```go
// https://github.com/etcd-io/etcd/blob/master/raft/raft.go#L1393
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


<br>

## raft.Node에서 채널 이벤트 기반으로 오케스트레이션 하기