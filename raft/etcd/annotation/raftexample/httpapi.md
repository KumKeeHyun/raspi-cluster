# httpapi
```go
// 클라이언트 요청을 처리하는 HTTP API
type httpKVAPI struct {
	store       *kvstore
	confChangeC chan<- raftpb.ConfChange
}

// http 패키지 핸들러
func (h *httpKVAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.RequestURI
	defer r.Body.Close()
	switch {
	case r.Method == "PUT": // key-value 쓰기 연산
		v, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read on PUT (%v)\n", err)
			http.Error(w, "Failed on PUT", http.StatusBadRequest)
			return
		}

		h.store.Propose(key, string(v)) // Raft 로그에 엔트리 추가

        // example 편의 사항 : 해당 엔트리가 커밋되는 것을 기다리지 않음
        // GET 요청에서 이전의 값이 반환될 수 있음
		w.WriteHeader(http.StatusNoContent)

	case r.Method == "GET": // key-value 읽기 연산
		if v, ok := h.store.Lookup(key); ok {
			w.Write([]byte(v))
		} else {
			http.Error(w, "Failed to GET", http.StatusNotFound)
		}

	case r.Method == "POST": // 클러스터 구성 변경 - 노드 추가
                             // key(r.RequestURI)는 추가할 노드의 ID, r.body는 추가할 노드의 url
		url, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read on POST (%v)\n", err)
			http.Error(w, "Failed on POST", http.StatusBadRequest)
			return
		}

		nodeId, err := strconv.ParseUint(key[1:], 0, 64)
		if err != nil {
			log.Printf("Failed to convert ID for conf change (%v)\n", err)
			http.Error(w, "Failed on POST", http.StatusBadRequest)
			return
		}

		cc := raftpb.ConfChange{
			Type:    raftpb.ConfChangeAddNode,
			NodeID:  nodeId,
			Context: url,
		}
		h.confChangeC <- cc // raft 모듈로 노드 추가 엔트리 전달

		// example 편의 사항 : 해당 엔트리가 커밋되는 것을 기다리지 않음
		w.WriteHeader(http.StatusNoContent)

	case r.Method == "DELETE": // 클러스터 구성 변경 - 노드 제거
		nodeId, err := strconv.ParseUint(key[1:], 0, 64)
		if err != nil {
			log.Printf("Failed to convert ID for conf change (%v)\n", err)
			http.Error(w, "Failed on DELETE", http.StatusBadRequest)
			return
		}

		cc := raftpb.ConfChange{
			Type:   raftpb.ConfChangeRemoveNode,
			NodeID: nodeId,
		}
		h.confChangeC <- cc // raft 모듈로 노드 제거 엔트리 전달

		// example 편의 사항 : 해당 엔트리가 커밋되는 것을 기다리지 않음
		w.WriteHeader(http.StatusNoContent)

	default:
		w.Header().Set("Allow", "PUT")
		w.Header().Add("Allow", "GET")
		w.Header().Add("Allow", "POST")
		w.Header().Add("Allow", "DELETE")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// key-value 서버를 시작
func serveHttpKVAPI(kv *kvstore, port int, confChangeC chan<- raftpb.ConfChange, errorC <-chan error) {
	srv := http.Server{
		Addr: ":" + strconv.Itoa(port),
		Handler: &httpKVAPI{
			store:       kv,
			confChangeC: confChangeC,
		},
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// exit when raft goes down
	if err, ok := <-errorC; ok {
		log.Fatal(err)
	}
}
```