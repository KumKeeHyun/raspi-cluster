# node

## Ready Struct
```go
// Ready는 안정적인 저장소에 저장할 것들(HardState, Entries, Snapshot), 
// 다른 노드로 전송할 메시지, state-machine에 적용할 것들(CommittedEntries, Snapshot)
// 을 캡슐화하여 라이브러리 외부로 전달한다.
// 모든 필드는 read-only이다.
type Ready struct {
    // 안전하지 보관하지 않아도 되는 raft node의 현재 상태(Lead, CurrentState).
    // 없데이트되어야 할 상태가 없다면 nil이 전달됨.
	*SoftState

    // 메시지들이 전송되기 전에 안정적인 저장소에 저장되어야 할 raft node의 상태(term, voted, commited).
    // 업데이트되어야 할 상태가 없다면 empty state가 전달됨.
	pb.HardState

    // ReadState에 있는 인덱스보다 appliedIndex가 더 클때 클라이언트의 읽기 요청을 처리할 수 있다.
    // ReadState는 raft 모듈이 MsgReadIndex 메시지를 받았을 때 리턴된다.
    // 리턴된 값은 해당 메시지를 보냈던 클라이언트 읽기 요청에만 유효하다.
	ReadStates []ReadState

    // 메시지들이 전송되기 전에 안정적인 저장소에 저장되어야 할 Entries.
	Entries []pb.Entry

    // 안정적인 저장소에 저장되어야 할 snapshot.
	Snapshot pb.Snapshot

    // state-machine에 적용되어야 할 entries.
    // 해당 entires들은 이미 안정적인 저장소에서 커밋되었음.
	CommittedEntries []pb.Entry

    // Entires들이 안정적인 저장소에 저장된 후에 외부로 전송해야하는 메시지들
    // 만약 MsgSnap 메시지가 포함되어있다면, 어플리케이션에서는 스냅샷을 전달받았거나, 
    // ReportSnapshot 호출이 실패했을 때 raft모듈로 다시 보고해야 함.
	Messages []pb.Message

    // HardState와 Entries가 디스크에 동기적으로 써야하는지, 비동기 쓰기가 허용되는지 나타냄.
	MustSync bool
}
```

## Node Interface
```go
type Node interface {
    // raft 모듈 내부의 논리적인 클락을 증가시킴. election timeout(candidate, follower), heartbeat timeout(leader)이 tick에 의해 발생됨.
	Tick()

    // 노드의 상태를 candidate로 바꾸고 leader가 되기위한 선거를 시작하도록 함.
	Campaign(ctx context.Context) error

    // raft log에 새로운 데이터를 추가하는 것을 제안함. 해당 제안은 어떠한 알림없이 드랍될 수 있음. 제안을 제시도하는 것은 어플리케이션의 책임임.  
	Propose(ctx context.Context, data []byte) error

	// ProposeConfChange proposes a configuration change. Like any proposal, the
	// configuration change may be dropped with or without an error being
	// returned. In particular, configuration changes are dropped unless the
	// leader has certainty that there is no prior unapplied configuration
	// change in its log.
	//
	// The method accepts either a pb.ConfChange (deprecated) or pb.ConfChangeV2
	// message. The latter allows arbitrary configuration changes via joint
	// consensus, notably including replacing a voter. Passing a ConfChangeV2
	// message is only allowed if all Nodes participating in the cluster run a
	// version of this library aware of the V2 API. See pb.ConfChangeV2 for
	// usage details and semantics.
	ProposeConfChange(ctx context.Context, cc pb.ConfChangeI) error

    // 메시지를 raft 모듈로 전달해서 적절하게 처리함. ctx.Err()가 반환될 수 있음.
	Step(ctx context.Context, msg pb.Message) error

    // 현재 시점 상태를 반환하는 채널을 리턴함. 어플리케이션은 Ready를 전달받고 적절히 처리한 후에 반드시 Advance를 호출해야 함. 
    // 주의: 이전의 Ready항목이 완전히 처리될 때까지 다음 Ready의 commited entries나 snapshot은 처리될 수 없음.
	Ready() <-chan Ready

    // Ready에 캡슐화해서 전달한 모든 진행사항이 어플리케이션에 의해 처리되었다는 것을 raft node에 전달함. 다음으로 전달할 진행사항을 Ready에 담아 전달하도록 준비한다.
    // 어플리케이션은 일반적으로 Ready의 모든 항목을 로그 저장소, 메타데이터 저장소, state-machine, 네트워크계층에 전달하여 처리한 뒤에 Advance를 호출해야 함.
    // 하지만 최적화를 위해 Ready의 항목을 처리하는 동안 Advance를 호출할 수 있음. 예를 들어, Ready에 스냅샷이 포함된 경우 해당 데이터를 처리하는 데 시간이 오래걸릴 수 있음. 이런 경우 raft 모듈의 진행을 중지하지 않고 다음 Ready를 수신받기 위해 Advance를 호출할 수 있음.
	Advance()

	// ApplyConfChange applies a config change (previously passed to
	// ProposeConfChange) to the node. This must be called whenever a config
	// change is observed in Ready.CommittedEntries, except when the app decides
	// to reject the configuration change (i.e. treats it as a noop instead), in
	// which case it must not be called.
	//
	// Returns an opaque non-nil ConfState protobuf which must be recorded in
	// snapshots.
	ApplyConfChange(cc pb.ConfChangeI) *pb.ConfState

	// TransferLeadership attempts to transfer leadership to the given transferee.
	TransferLeadership(ctx context.Context, lead, transferee uint64)

	// ReadIndex request a read state. The read state will be set in the ready.
	// Read state has a read index. Once the application advances further than the read
	// index, any linearizable read requests issued before the read request can be
	// processed safely. The read state will have the same rctx attached.
	ReadIndex(ctx context.Context, rctx []byte) error

	// Status returns the current status of the raft state machine.
	Status() Status
    
	// ReportUnreachable reports the given node is not reachable for the last send.
	ReportUnreachable(id uint64)

	// ReportSnapshot reports the status of the sent snapshot. The id is the raft ID of the follower
	// who is meant to receive the snapshot, and the status is SnapshotFinish or SnapshotFailure.
	// Calling ReportSnapshot with SnapshotFinish is a no-op. But, any failure in applying a
	// snapshot (for e.g., while streaming it from leader to follower), should be reported to the
	// leader with SnapshotFailure. When leader sends a snapshot to a follower, it pauses any raft
	// log probes until the follower can apply the snapshot and advance its state. If the follower
	// can't do that, for e.g., due to a crash, it could end up in a limbo, never getting any
	// updates from the leader. Therefore, it is crucial that the application ensures that any
	// failure in snapshot sending is caught and reported back to the leader; so it can resume raft
	// log probing in the follower.
	ReportSnapshot(id uint64, status SnapshotStatus)

	// Stop performs any necessary termination of the Node.
	Stop()
}
```