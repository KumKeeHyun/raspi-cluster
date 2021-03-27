# raft

```go
type commit struct {
	data       []string
	applyDoneC chan<- struct{}
}

type raftNode struct {
	proposeC    <-chan string            // key-value 쓰기 연산 요청
	confChangeC <-chan raftpb.ConfChange // 클러스터 구성 변경 요청
	commitC     chan<- *commit           // 커밋된 엔트리들을 외부에 전달하는 채널. 
	errorC      chan<- error             // raft 모듈에서 발생된 에러를 전달하는 채널

	id          int      // client ID for raft session
	peers       []string // raft peer URLs
	join        bool     // node is joining an existing cluster
	waldir      string   // path to WAL directory
	snapdir     string   // path to snapshot directory
	getSnapshot func() ([]byte, error)

	confState     raftpb.ConfState
	snapshotIndex uint64
	appliedIndex  uint64

	// raft backing for the commit/error channel
	node        raft.Node
	raftStorage *raft.MemoryStorage
	wal         *wal.WAL

	snapshotter      *snap.Snapshotter // 스냅샷 파일 생성, 쓰기, 읽기 등을 담당하는 객체
	snapshotterReady chan *snap.Snapshotter // raft 모듈이 준비가 끝나면 외부로 스냅샷 관리 객체를 전달하기 위한 채널

	snapCount uint64
	transport *rafthttp.Transport // raft 클러스터의 네트워크 계층
	stopc     chan struct{} // signals proposal channel closed
	httpstopc chan struct{} // signals http server to shutdown
	httpdonec chan struct{} // signals http server shutdown complete

	logger *zap.Logger
}

var defaultSnapshotCount uint64 = 10000


// proposeC: raft 모듈에 로그 업데이트를 제안하기 위한 채널
// confChangeC: raft 모듈에 클러스터 구성 업데이트를 제안하기 위한 채널
// commit: raft 모듈의 로그에서 커밋된 엔트리들을 외부로 전달하기 위한 채널. nil이 먼저 전달되고 이후 커밋된 엔트리들이 전달됨.
// raft 모듈을 중단하려면 외부에서 proposeC 채널을 닫고 error 채널을 읽으면 됨.
func newRaftNode(id int, peers []string, join bool, getSnapshot func() ([]byte, error), proposeC <-chan string,
	confChangeC <-chan raftpb.ConfChange) (<-chan *commit, <-chan error, <-chan *snap.Snapshotter) {

	commitC := make(chan *commit)
	errorC := make(chan error)

	rc := &raftNode{
		proposeC:    proposeC,
		confChangeC: confChangeC,
		commitC:     commitC,
		errorC:      errorC,
		id:          id,
		peers:       peers,
		join:        join,
		waldir:      fmt.Sprintf("raftexample-%d", id),
		snapdir:     fmt.Sprintf("raftexample-%d-snap", id),
		getSnapshot: getSnapshot,
		snapCount:   defaultSnapshotCount,
		stopc:       make(chan struct{}),
		httpstopc:   make(chan struct{}),
		httpdonec:   make(chan struct{}),

		logger: zap.NewExample(),

		snapshotterReady: make(chan *snap.Snapshotter, 1),
		// rest of structure populated after WAL replay
	}
	go rc.startRaft()
	return commitC, errorC, rc.snapshotterReady
}

func (rc *raftNode) startRaft() {
	if !fileutil.Exist(rc.snapdir) {
		if err := os.Mkdir(rc.snapdir, 0750); err != nil {
			log.Fatalf("raftexample: cannot create dir for snapshot (%v)", err)
		}
	}
	rc.snapshotter = snap.New(zap.NewExample(), rc.snapdir)

	oldwal := wal.Exist(rc.waldir)
	rc.wal = rc.replayWAL()

	// signal replay has finished
	rc.snapshotterReady <- rc.snapshotter

	rpeers := make([]raft.Peer, len(rc.peers))
	for i := range rpeers {
		rpeers[i] = raft.Peer{ID: uint64(i + 1)}
	}
	c := &raft.Config{
		ID:                        uint64(rc.id),
		ElectionTick:              10,
		HeartbeatTick:             1,
		Storage:                   rc.raftStorage,
		MaxSizePerMsg:             1024 * 1024,
		MaxInflightMsgs:           256,
		MaxUncommittedEntriesSize: 1 << 30,
	}

	if oldwal || rc.join { // 이전에 실행됐던 기록이 있거나 클러스터에 join하는 노드라면 RestartNode
		rc.node = raft.RestartNode(c)
	} else { // 새로운 클러스터를 생성하는 노드라면 StartNode
		rc.node = raft.StartNode(c, rpeers)
	}

	rc.transport = &rafthttp.Transport{ // raft 모듈 네트워크 계층 구현부
		Logger:      rc.logger,
		ID:          types.ID(rc.id),
		ClusterID:   0x1000,
		Raft:        rc,
		ServerStats: stats.NewServerStats("", ""),
		LeaderStats: stats.NewLeaderStats(zap.NewExample(), strconv.Itoa(rc.id)),
		ErrorC:      make(chan error),
	}

	rc.transport.Start()
	for i := range rc.peers {
		if i+1 != rc.id {
			rc.transport.AddPeer(types.ID(i+1), []string{rc.peers[i]}) 
		}
	}

	go rc.serveRaft()
	go rc.serveChannels()
}


func (rc *raftNode) serveChannels() {
	snap, err := rc.raftStorage.Snapshot()
	if err != nil {
		panic(err)
	}
	rc.confState = snap.Metadata.ConfState
	rc.snapshotIndex = snap.Metadata.Index
	rc.appliedIndex = snap.Metadata.Index

	defer rc.wal.Close()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// 외부에서 제안한 엔트리를 raft 모듈에 전달
	// proposeC: state machine 관련 엔트리, confChangeC: 클러스터 구성 관련 엔트리
	go func() {
		confChangeCount := uint64(0)

		for rc.proposeC != nil && rc.confChangeC != nil { // 외부에서 proposeC, confChangeC 채널을 닫기 전까지
			select {
			case prop, ok := <-rc.proposeC:
				if !ok {
					rc.proposeC = nil
				} else {
					// blocks until accepted by raft state machine
					rc.node.Propose(context.TODO(), []byte(prop))
				}

			case cc, ok := <-rc.confChangeC:
				if !ok {
					rc.confChangeC = nil
				} else {
					confChangeCount++
					cc.ID = confChangeCount
					rc.node.ProposeConfChange(context.TODO(), cc)
				}
			}
		}
		// 외부에서 raft 모듈을 중단하기 위해 채널을 닫으면 모듈의 고루틴들에게 정지 신호 전달
		close(rc.stopc)
	}()

	// state machine 업데이트 이벤트 루프
	for {
		select {
		case <-ticker.C:
			rc.node.Tick()

		// 엔트리들을 wal에 저장하고 commit channel에 전달
		case rd := <-rc.node.Ready():
			rc.wal.Save(rd.HardState, rd.Entries)
			if !raft.IsEmptySnap(rd.Snapshot) {
				rc.saveSnap(rd.Snapshot)
				rc.raftStorage.ApplySnapshot(rd.Snapshot)
				rc.publishSnapshot(rd.Snapshot)
			}
			rc.raftStorage.Append(rd.Entries)
			rc.transport.Send(rd.Messages)
			// 커밋된 엔트리들을 state machine, 클러스터 구성에 적용
			applyDoneC, ok := rc.publishEntries(rc.entriesToApply(rd.CommittedEntries))
			if !ok {
				rc.stop()
				return
			}
			// applied된 엔트리 개수가 일정 수준 이상일 때 스냅샷 생성, 저장, 적용  
			rc.maybeTriggerSnapshot(applyDoneC)
			rc.node.Advance()

		case err := <-rc.transport.ErrorC:
			rc.writeError(err)
			return

		case <-rc.stopc:
			rc.stop()
			return
		}
	}
}

// 전달된 엔트리들의 인덱스를 검사해서 state machine에 적용할 수 있는 범위의 엔트리들만 리턴
func (rc *raftNode) entriesToApply(ents []raftpb.Entry) (nents []raftpb.Entry) {
	if len(ents) == 0 {
		return ents
	}
	firstIdx := ents[0].Index
	if firstIdx > rc.appliedIndex+1 { // 엔트리들과 적용된 엔트리들 사이에 공백이 있는 경우 일어날 수 없음. 프로세스 종료 
		log.Fatalf("first index of committed entry[%d] should <= progress.appliedIndex[%d]+1", firstIdx, rc.appliedIndex)
	}
	if rc.appliedIndex-firstIdx+1 < uint64(len(ents)) { 
		nents = ents[rc.appliedIndex-firstIdx+1:] // ents 중에서 appliedIndex 이후에 있는 엔트리들만 추출
	}
	return nents
}

// 엔트리들을 commit channel에 전달. 클러스터 구성 변경 엔트리의 경우 raft 모듈, 네트워크 계층에 적용
// 외부에서 커밋된 엔트리들을 모두 처리할 때까지 기다리는 채널 반환
func (rc *raftNode) publishEntries(ents []raftpb.Entry) (<-chan struct{}, bool) {
	if len(ents) == 0 {
		return nil, true
	}

	data := make([]string, 0, len(ents)) // state machine에 전달할 엔트리
	for i := range ents {
		switch ents[i].Type {
		case raftpb.EntryNormal: // state machine에 전달할 엔트리 타입인 경우
			if len(ents[i].Data) == 0 {
				// 메시지가 비어있다면 무시
				break
			}
			s := string(ents[i].Data)
			data = append(data, s)
		case raftpb.EntryConfChange: // 클러스터 구성 변경 엔트리 타입인 경우
			var cc raftpb.ConfChange
			cc.Unmarshal(ents[i].Data)
			rc.confState = *rc.node.ApplyConfChange(cc) // raft 모듈에 적용
			// 네트워크 계층에 적용
			switch cc.Type {
			case raftpb.ConfChangeAddNode:
				if len(cc.Context) > 0 {
					rc.transport.AddPeer(types.ID(cc.NodeID), []string{string(cc.Context)})
				}
			case raftpb.ConfChangeRemoveNode:
				if cc.NodeID == uint64(rc.id) {
					log.Println("I've been removed from the cluster! Shutting down.")
					return nil, false
				}
				rc.transport.RemovePeer(types.ID(cc.NodeID))
			}
		}
	}

	var applyDoneC chan struct{}

	if len(data) > 0 {
		applyDoneC = make(chan struct{}, 1)
		select {
		case rc.commitC <- &commit{data, applyDoneC}: // commitC 채널에 엔트리들 전달
		case <-rc.stopc:
			return nil, false
		}
	}

	// appliedIndex 업데이트
	rc.appliedIndex = ents[len(ents)-1].Index

	return applyDoneC, true
}

func (rc *raftNode) serveRaft() {
	url, err := url.Parse(rc.peers[rc.id-1])
	if err != nil {
		log.Fatalf("raftexample: Failed parsing URL (%v)", err)
	}

	ln, err := newStoppableListener(url.Host, rc.httpstopc) // httpstopc에 의해 중단될 수 있는 커스텀 TCPListener
	if err != nil {
		log.Fatalf("raftexample: Failed to listen rafthttp (%v)", err)
	}

	err = (&http.Server{Handler: rc.transport.Handler()}).Serve(ln) // 네트워크 계층 서버 실행
	select {
	case <-rc.httpstopc:
	default:
		log.Fatalf("raftexample: Failed to serve rafthttp (%v)", err)
	}
	close(rc.httpdonec)
}
```