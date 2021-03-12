# kvstore

```go
// example을 위한 간단한 key-value store
type kvstore struct {
	proposeC    chan<- string // channel for proposing updates
                              // raft.Node.Propose에 엔트리를 전달할 데이터 채널
	mu          sync.RWMutex
	kvStore     map[string]string
	snapshotter *snap.Snapshotter
}

func newKVStore(snapshotter *snap.Snapshotter, proposeC chan<- string, commitC <-chan *commit, errorC <-chan error) *kvstore {
	s := &kvstore{proposeC: proposeC, kvStore: make(map[string]string), snapshotter: snapshotter}
	snapshot, err := s.loadSnapshot()
	if err != nil {
		log.Panic(err)
	}
	if snapshot != nil {
		log.Printf("loading snapshot at term %d and index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
		if err := s.recoverFromSnapshot(snapshot.Data); err != nil {
			log.Panic(err)
		}
	}
	// read commits from raft into kvStore map until error
	go s.readCommits(commitC, errorC)
	return s
}

// raft 모듈로 복제할 엔트리를 전달
func (s *kvstore) Propose(k string, v string) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(kv{k, v}); err != nil {
		log.Fatal(err)
	}
	s.proposeC <- buf.String()
}

// raft 모듈에서 전달받은 commit channel을 읽어서 state machine에 적용
// commit은 entry들을 배치형식으로 전달
// type commit struct {
// 	data       []string
// 	applyDoneC chan<- struct{}
// }
func (s *kvstore) readCommits(commitC <-chan *commit, errorC <-chan error) {
	for commit := range commitC {
		if commit == nil { // commit이 nil이라면 snapshot을 적재하라는 뜻
                           // 지정된 디렉토리에서 스냅샷을 읽어 statemachine에 적재
			snapshot, err := s.loadSnapshot()
			if err != nil {
				log.Panic(err)
			}
			if snapshot != nil {
				log.Printf("loading snapshot at term %d and index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
				if err := s.recoverFromSnapshot(snapshot.Data); err != nil {
					log.Panic(err)
				}
			}
			continue
		}

		for _, data := range commit.data { // commit이 쓰기연산을 위한 entries라면 하나씩 디코딩해서 적재
			var dataKv kv
			dec := gob.NewDecoder(bytes.NewBufferString(data))
			if err := dec.Decode(&dataKv); err != nil { 
				log.Fatalf("raftexample: could not decode message (%v)", err)
			}
			s.mu.Lock()
			s.kvStore[dataKv.Key] = dataKv.Val
			s.mu.Unlock()
		}
		close(commit.applyDoneC) // 채널을 닫아 동기화
	}
	if err, ok := <-errorC; ok {
		log.Fatal(err)
	}
}

// 스냅샷을 생성
// 스냅샷을 만드는 동안 다른 고루틴은 접근 못함
func (s *kvstore) getSnapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s.kvStore)
}

// snapshotter에 지정된 디렉토리에서 최신의 스냅샷 파일을 읽어 역직렬화한 스냅샷 리턴
// 스냅샷 파일이 없다면 nil 리턴 
func (s *kvstore) loadSnapshot() (*raftpb.Snapshot, error) {
	snapshot, err := s.snapshotter.Load()
	if err == snap.ErrNoSnapshot {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

// 스냅샷 내용을 kvstore에 적재
func (s *kvstore) recoverFromSnapshot(snapshot []byte) error {
	var store map[string]string
	if err := json.Unmarshal(snapshot, &store); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kvStore = store
	return nil
}
```