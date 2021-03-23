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

    // 클러스터 구성 변경을 제안함. Propose와 마찬가지고 구성 변경 요청은 에러가 반환되던지 안되던지 드랍될 수 있음. 리더가 이전에 제안된 구성변경 적용하기 위해 기다리고 있는경우 이 구성 변경 요청은 리더에 의해 드랍됨.
    // cc는 pb.ConfChange, pb.ConfChangeV2 메시지를 허용함. pb.ConfChangeV2는 유권자 교체를 포함한 임의의 구성 변경을 할 수 있음.   
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

    // 클러스터 구성 변경을 노드에 적용함. Ready.CommitedEntries에서 구성 변경 entry가 발견될 때마다 호출되어야 함. 하지만 어플리케이션이 구성 변경을 거부하기로 결정한 경우에는 호출하면 안됨. 스냅샷에 저장될 ConfState protobuf를 리턴함. 해당 값은 항상 nil이 아님.
	ApplyConfChange(cc pb.ConfChangeI) *pb.ConfState

    // 주어진 transferee를 리더로 전환하는 것을 시도함. 리더에게 MsgTransferLeader 메시지를 전송함.
	TransferLeadership(ctx context.Context, lead, transferee uint64)

    // ReadState를 요청함. ReadState는 이후에 반환될 Ready.ReadStates에 담겨 전달됨. 리더에게 MsgReadIndex 메시지를 전송함.  
    // TODO: readIndex까지는 follower가 읽기 요청을 처리할 수 있다는 의미?
	ReadIndex(ctx context.Context, rctx []byte) error

    // 현재 raft state machine에 대한 상태를 반환함
	Status() Status

    // 주어진 id의 노드에게 해당 노드로부터 온 send에 도달할 수 없다고 보고함. 해당 노드에게 MsgUnreachable 메시지를 전송함.
	ReportUnreachable(id uint64)

    // 전달받은 snapshot의 상태를 보고함. id는 스냅샷을 받은 팔로워의 id이고, status는 SnapshotFinish이나 SnapshotFailure를 의미함.
    // SnapshotFinish으로 해당 함수를 호출하면 아무런 기능을 수행하지 않음. 하지만 스냅샷 적용이 실패하면 리더에게 SnapshotFinish을 보고해야 함.
    // 리더가 팔로워에게 스냅샷을 보내면 팔로워가 스냅샷 적용을 끝낼 때까지 해당 probe를 중지시킴. 팔로워가 스냅샷 적용을 실패하고 이를 리더에게 보고하지 않는다면 팔로워는 리더의 업데이트 정보를 전달받지 못할 수 있음. 따라서 어플리케이션이 스냅샷 전송 오류나 적용 오류를 리더에게 다시 보고하도록 하는 것이 중요함.
	ReportSnapshot(id uint64, status SnapshotStatus)

	// Stop performs any necessary termination of the Node.
	Stop()
}
```

## node.run
```go
func (n *node) run() {
	var propc chan msgWithResult
	var readyc chan Ready
	var advancec chan struct{}
	var rd Ready

	r := n.rn.raft

	lead := None

	for {
		if advancec != nil { // readyc 채널로 Ready를 보내고 Advance가 호출되지 않은 상태라면
			readyc = nil // 'case ready <- rd'가 작동하지 않도록 readyc를 nil로 유지
		} else if n.rn.HasReady() {
			// readyc 채널을 준비했을 뿐이지 실제로 바로 보내질지는 보장하지 않음. 다른 채널들에 대한 루프를 처리한 뒤에 Ready가 처리될 수도 있음. 일반적으로 더큰 Ready를 내보내고 단순화(덜 자주, 더 예측 가능하게)하기 위해 이런식으로 설정함.
			rd = n.rn.readyWithoutAccept()
			readyc = n.readyc
		}

        // 리더가 변경된 경우(새로운 리더 발견, 리더 잃어버림) proposalChannel 세팅
		if lead != r.lead {
			if r.hasLeader() { 
				if lead == None { 
					r.logger.Infof("raft.node: %x elected leader %x at term %d", r.id, r.lead, r.Term)
				} else { // 
					r.logger.Infof("raft.node: %x changed leader from %x to %x at term %d", r.id, lead, r.lead, r.Term)
				}
				propc = n.propc
			} else {
				r.logger.Infof("raft.node: %x lost leader %x at term %d", r.id, lead, r.Term)
				propc = nil
			}
			lead = r.lead
		}

		select {
		case pm := <-propc: // proposal
			m := pm.m
			m.From = r.id
			err := r.Step(m) 
			if pm.result != nil { // result에 동기화된 요청이라면
				pm.result <- err // step 결과 알려줌
				close(pm.result)
			}
		case m := <-n.recvc:
            // 알고있는 peer로부터 온 메시지거나 Resq가 아닌 메시지는 필터링함.
			if pr := r.prs.Progress[m.From]; pr != nil || !IsResponseMsg(m.Type) {
				r.Step(m)
			}
		case cc := <-n.confc:
			_, okBefore := r.prs.Progress[r.id]
			cs := r.applyConfChange(cc)

			// 새로운 클러스터 구성에서 제거된 경우 들어오는 제안을 차단함(propc = nil). 노드의 로그가 최신이 아니고 리더를 따라잡고 있는 경우에 최신의 클러스터 구성을 갖고있지 않을 수 있기 때문에 최신 구성에서 노드가 존재할 경우 채널을 차단하고 싶지 않다고 함.
			// NB(주목): propc는 리더가 변경되면 재설정되지만 이것이 무엇을 암시하는지 정확히 알 수 없기 때문에 버그가 발생할 수 있음.
			if _, okAfter := r.prs.Progress[r.id]; okBefore && !okAfter {
				var found bool
			outer:
				for _, sl := range [][]uint64{cs.Voters, cs.VotersOutgoing} { // voters, votersOutgoing에서 자신을 찾음
					for _, id := range sl {
						if id == r.id {
							found = true
							break outer
						}
					}
				}
				if !found {
					propc = nil
				}
			}
			select {
			case n.confstatec <- cs: // 외부로 변경된 구성을 전달. (ApplyConfChange에서 채널을 수신하고 있음.)
			case <-n.done:
			}
		case <-n.tickc: // tick 실행 (tickLeader, tickCandidiate, tickFollower)
			n.rn.Tick()
		case readyc <- rd: // readyc != nil(Ready가 준비가 되었다면)
			n.rn.acceptReady(rd)
			advancec = n.advancec // 이후 반복문에서는 Advance가 호출되기 전까지 readyc는 nil로 유지됨
		case <-advancec:
			n.rn.Advance(rd)
			rd = Ready{}
			advancec = nil // 다시 readyc가 n.readyc로 세팅될 수 있도록 함.
		case c := <-n.status:
			c <- getStatus(r)
		case <-n.stop:
			close(n.done)
			return
		}
	}
}
```

## node.Propose && node.ProposeConfChange && node.Step
```go 
// 쓰기 연산을 제안
func (n *node) Propose(ctx context.Context, data []byte) error {
	return n.stepWait(ctx, pb.Message{Type: pb.MsgProp, Entries: []pb.Entry{{Data: data}}})
}

func (n *node) ProposeConfChange(ctx context.Context, cc pb.ConfChangeI) error {
	msg, err := confChangeToMsg(cc) // ConfChange를 담은 MsgProp 메시지 생성
	if err != nil {
		return err
	}
	return n.Step(ctx, msg) // n.run에서 처리될 때까지 기다리지 않음
}

// 다른 노드들로부터 전달받은 메시지들을 처리
func (n *node) Step(ctx context.Context, m pb.Message) error {
	// 네트워크로 전달받은 메시지들에서 로컬 메시지가 있다면 드랍
	if IsLocalMsg(m.Type) {
		return nil
	}
	return n.step(ctx, m)
}

// waitOption = false
func (n *node) step(ctx context.Context, m pb.Message) error {
	return n.stepWithWaitOption(ctx, m, false)
}

// waitOption = true
func (n *node) stepWait(ctx context.Context, m pb.Message) error {
	return n.stepWithWaitOption(ctx, m, true)
}

func (n *node) stepWithWaitOption(ctx context.Context, m pb.Message, wait bool) error {
	if m.Type != pb.MsgProp { // proposal을 제외한 나머지(다른 노드에서 전달받은 메시지)는 recvc에서 처리되도록 넘김
		select {
		case n.recvc <- m:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-n.done:
			return ErrStopped
		}
	}
	ch := n.propc
	pm := msgWithResult{m: m} // propc 채널에 전달할 메시지 래핑
	if wait {
		pm.result = make(chan error, 1) // n.run에서 메시지가 처리될 때까지 기다리는 채널
	}
	select {
	case ch <- pm:
		if !wait { // wait == false 라면 바로 리턴
			return nil
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-n.done:
		return ErrStopped
	}
	select { // wait == true 라면 result 채널을 수신
			 // 만약 제안이 드랍됐다면 raft.ErrProposalDropped 리턴 
	case err := <-pm.result:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-n.done:
		return ErrStopped
	}
	return nil
}
```

## node.ApplyConfChange
```go 
func (n *node) ApplyConfChange(cc pb.ConfChangeI) *pb.ConfState {
	var cs pb.ConfState
	select {
	case n.confc <- cc.AsV2(): // n.run에서 새로운 구성을 적용하도록(raft.applyConfChange(cc)) 채널에 전달
	case <-n.done:
	}
	select {
	case cs = <-n.confstatec: // n.run에서 구성 적용이 완료되면 새로운 구성이 채널로 전달됨
	case <-n.done:
	}
	return &cs
}
```