# etcd raft 모듈 사용해보기

[etcd raft 모듈 분석 링크](./understanding-etcd-raft-1.md)

## TOC
<!--ts-->
- [etcd raft 모듈 사용해보기](#etcd-raft-모듈-사용해보기)
  - [TOC](#toc)
  - [서론](#서론)
  - [etcd는 어떻게 사용하는가?](#etcd는-어떻게-사용하는가)
    - [bootstrap](#bootstrap)
  - [etcd-examples는 어떻게 사용하는가?](#etcd-examples는-어떻게-사용하는가)
  - [마치며](#마치며)
<!--te-->

## 서론

## etcd는 어떻게 사용하는가?

### bootstrap

```go
// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/etcdserver/bootstrap.go#L473
func bootstrapRaft(cfg config.ServerConfig, cluster *bootstrapedCluster, bwal *bootstrappedWAL) *bootstrappedRaft {
	switch {
	case !bwal.haveWAL && !cfg.NewCluster:
        // WAL이 없고 새로운 클러스터를 초기화하는 것도 아님 -> 기존 클러스터에 새롭게 join 하는 노드
        // peers를 비움 -> raft.RestartNode
		return bootstrapRaftFromCluster(cfg, cluster.cl, nil, bwal)
	case !bwal.haveWAL && cfg.NewCluster:
        // WAL이 없고 새로운 클러스터를 초기화함 -> 새로운 클러스터를 구성하는 노드 
        // peers를 전달 -> raft.StartNode
		return bootstrapRaftFromCluster(cfg, cluster.cl, cluster.cl.MemberIDs(), bwal)
	case bwal.haveWAL:
        // WAL이 있음 -> 충돌 or 중단되었다가 재시작하는 노드
		return bootstrapRaftFromWAL(cfg, bwal)
	default:
		cfg.Logger.Panic("unsupported bootstrap config")
		return nil
	}
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/etcdserver/bootstrap.go#L487
func bootstrapRaftFromCluster(cfg config.ServerConfig, cl *membership.RaftCluster, ids []types.ID, bwal *bootstrappedWAL) *bootstrappedRaft {
    // ...
    peers := make([]raft.Peer, len(ids))
    // ...
    return &bootstrappedRaft{
		// ...
		peers:     peers,
        // ...
	}
}

// https://github.com/etcd-io/etcd/blob/v3.6.0-alpha.0/server/etcdserver/bootstrap.go#L537
func (b *bootstrappedRaft) newRaftNode(ss *snap.Snapshotter, wal *wal.WAL, cl *membership.RaftCluster) *raftNode {
	var n raft.Node
	if len(b.peers) == 0 {
		n = raft.RestartNode(b.config)
	} else {
		n = raft.StartNode(b.config, b.peers)
	}
	// ...
}
```

## etcd-examples는 어떻게 사용하는가?


## 마치며