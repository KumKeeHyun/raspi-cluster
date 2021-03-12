# Snapshotter
- `etcd/server/etcdserver/api/snap`

## 구조체
```go
type Snapshotter struct {
	lg  *zap.Logger
	dir string // 스냅샷 파일을 저장할 디렉토리
}
```

## 메소드, 함수
```go
func (s *Snapshotter) SaveSnap(snapshot raftpb.Snapshot) error {
	if raft.IsEmptySnap(snapshot) {
		return nil
	}
	return s.save(&snapshot)
}

func (s *Snapshotter) save(snapshot *raftpb.Snapshot) error {
	start := time.Now()

	fname := fmt.Sprintf("%016x-%016x%s", snapshot.Metadata.Term, snapshot.Metadata.Index, snapSuffix)
	b := pbutil.MustMarshal(snapshot)   // raftpb.Snapshot(실제 데이터 + index, term, config 같은 메타데이터)를 []byte로 직렬화 
                                        // Marscal() 호출한 뒤에 에러가 있다면 panic
	crc := crc32.Update(0, crcTable, b) // error detection을 위한 checksum 연산
	snap := snappb.Snapshot{Crc: crc, Data: b} 
	d, err := snap.Marshal() // crc를 포함한 데이터를 다시 직렬화
	if err != nil {
		return err
	}

	spath := filepath.Join(s.dir, fname)
	err = pioutil.WriteAndSyncFile(spath, d, 0666) // spath 파일을 열고 wrtie, fsync 연산

	if err != nil {
		s.lg.Warn("failed to write a snap file", zap.String("path", spath), zap.Error(err))
		rerr := os.Remove(spath)
		if rerr != nil {
			s.lg.Warn("failed to remove a broken snap file", zap.String("path", spath), zap.Error(err))
		}
		return err
	}

	return nil
}

// 가장 최근의 스냅샷 파일을 역직렬화한뒤 리턴
func (s *Snapshotter) Load() (*raftpb.Snapshot, error) {
	return s.loadMatching(func(*raftpb.Snapshot) bool { return true })
}

// 스냅샷 파일들에서 최근 순서대로 matchFn을 만족한 파일을 역직렬화한뒤 리턴 
func (s *Snapshotter) loadMatching(matchFn func(*raftpb.Snapshot) bool) (*raftpb.Snapshot, error) {
	names, err := s.snapNames() // 디렉토리에 있는 .snap 파일들을 최근 순서대로 정렬해서 리턴
	if err != nil {
		return nil, err
	}
	var snap *raftpb.Snapshot
	for _, name := range names {
		if snap, err = loadSnap(s.lg, s.dir, name); err == nil && matchFn(snap) { // 스냅샷 파일들 중에서 원하는 조건에 맞는 파일을 역직렬화해서 리턴
			return snap, nil
		}
	}
	return nil, ErrNoSnapshot
}

// 스냅샷 파일을 읽어서 raft 모듈이 사용하는 스냅샷 타입으로 역직렬화
// 스냅샷 파일에 오류가 있다면 파일 이름에 .broken suffix 추가
func loadSnap(lg *zap.Logger, dir, name string) (*raftpb.Snapshot, error) {
	fpath := filepath.Join(dir, name)
	snap, err := Read(lg, fpath)
	if err != nil {
		brokenPath := fpath + ".broken"
		if lg != nil {
			lg.Warn("failed to read a snap file", zap.String("path", fpath), zap.Error(err))
		}
		if rerr := os.Rename(fpath, brokenPath); rerr != nil {
			if lg != nil {
				lg.Warn("failed to rename a broken snap file", zap.String("path", fpath), zap.String("broken-path", brokenPath), zap.Error(rerr))
			}
		} else {
			if lg != nil {
				lg.Warn("renamed to a broken snap file", zap.String("path", fpath), zap.String("broken-path", brokenPath))
			}
		}
	}
	return snap, err
}

// 파일을 읽어서 error detection 한뒤에 raft 모듈이 사용하는 스냅샷 타입으로 역직렬화
func Read(lg *zap.Logger, snapname string) (*raftpb.Snapshot, error) {
	b, err := ioutil.ReadFile(snapname)
	if err != nil {
		if lg != nil {
			lg.Warn("failed to read a snap file", zap.String("path", snapname), zap.Error(err))
		}
		return nil, err
	}

	if len(b) == 0 {
		if lg != nil {
			lg.Warn("failed to read empty snapshot file", zap.String("path", snapname))
		}
		return nil, ErrEmptySnapshot
	}

	var serializedSnap snappb.Snapshot
	if err = serializedSnap.Unmarshal(b); err != nil { // crc를 포함한 스냅샷 데이터 역직렬화
		if lg != nil {
			lg.Warn("failed to unmarshal snappb.Snapshot", zap.String("path", snapname), zap.Error(err))
		}
		return nil, err
	}

	if len(serializedSnap.Data) == 0 || serializedSnap.Crc == 0 {
		if lg != nil {
			lg.Warn("failed to read empty snapshot data", zap.String("path", snapname))
		}
		return nil, ErrEmptySnapshot
	}

	crc := crc32.Update(0, crcTable, serializedSnap.Data) // crc를 통한 error detection
	if crc != serializedSnap.Crc {
		if lg != nil {
			lg.Warn("snap file is corrupt",
				zap.String("path", snapname),
				zap.Uint32("prev-crc", serializedSnap.Crc),
				zap.Uint32("new-crc", crc),
			)
		}
		return nil, ErrCRCMismatch
	}

	var snap raftpb.Snapshot
	if err = snap.Unmarshal(serializedSnap.Data); err != nil { // raft 모듈에서 사용하는 스냅샷 타입으로 역직렬화
		if lg != nil {
			lg.Warn("failed to unmarshal raftpb.Snapshot", zap.String("path", snapname), zap.Error(err))
		}
		return nil, err
	}
	return &snap, nil
}
```