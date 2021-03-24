# bootstrap
```go
// Bootstrap initializes the RawNode for first use by appending configuration
// changes for the supplied peers. This method returns an error if the Storage
// is nonempty.
//
// It is recommended that instead of calling this method, applications bootstrap
// their state manually by setting up a Storage that has a first index > 1 and
// which stores the desired ConfState as its InitialState.

// 로그에 ConfChange entry를 추가하고 log.commit을 설정, raft에 적용하는 것으로 peer에 대한 설정을 초기화함.
func (rn *RawNode) Bootstrap(peers []Peer) error {
	if len(peers) == 0 {
		return errors.New("must provide at least one peer to Bootstrap")
	}
	lastIndex, err := rn.raft.raftLog.storage.LastIndex()
	if err != nil {
		return err
	}

	if lastIndex != 0 {
		return errors.New("can't bootstrap a nonempty Storage")
	}

	// We've faked out initial entries above, but nothing has been
	// persisted. Start with an empty HardState (thus the first Ready will
	// emit a HardState update for the app to persist).
	rn.prevHardSt = emptyState

	// TODO(tbg): remove StartNode and give the application the right tools to
	// bootstrap the initial membership in a cleaner way.
	rn.raft.becomeFollower(1, None)
	ents := make([]pb.Entry, len(peers))
	for i, peer := range peers {
		cc := pb.ConfChange{Type: pb.ConfChangeAddNode, NodeID: peer.ID, Context: peer.Context}
		data, err := cc.Marshal()
		if err != nil {
			return err
		}

		ents[i] = pb.Entry{Type: pb.EntryConfChange, Term: 1, Index: uint64(i + 1), Data: data}
	}
	rn.raft.raftLog.append(ents...)

	// StartNode 직후에 Campaign을 호출할 수 있도록 엔트리를 적용해야 함.
	// applyConfChange는 이곳, Ready 루프에서 총 두번 실행된다.
	// Ready 루프에서 해당 entry가 실행되려면 (in Follower-stepFollower-case MsgApp-handleAppendEntries-raftLog.maybeAppend), (in Leader-stepLeader-case MsgAppResq-progress.MaybeUpdate)이 모두 실행되어야 하기 때문에 progress.next는 이러한 bootstrap entry 뒤에 알맞게 설정됨. Ready.CommitedEntries를 통해 모든 클러스터 구성 변경 사항을 관찰할 수 있도록 raftLog.applied는 설정하지 않음.
	rn.raft.raftLog.committed = uint64(len(ents))
	for _, peer := range peers {
		rn.raft.applyConfChange(pb.ConfChange{NodeID: peer.ID, Type: pb.ConfChangeAddNode}.AsV2())
	}
	return nil
}
```