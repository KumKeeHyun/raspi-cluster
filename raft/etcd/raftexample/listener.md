# listener

## 사용
- `raftexample/raft.go`
```go
ln, err := newStoppableListener(url.Host, rc.httpstopc)

// func (srv *Server) Serve(l net.Listener) error
(&http.Server{Handler: rc.transport.Handler()}).Serve(ln)
```

## 코드
```go
// net.Listener를 래핑
// Accept()에서 블록되는동안 stopc를 통한 종료신호가 오면 TCP 연결 요청을 기다리지 않고 바로 종료
// 연결된 TCP 요청에는 keep-alive 설정
type stoppableListener struct {
	*net.TCPListener
	stopc <-chan struct{}
}

func newStoppableListener(addr string, stopc <-chan struct{}) (*stoppableListener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &stoppableListener{ln.(*net.TCPListener), stopc}, nil // AcceptTCP를 위해 Listener interface를 TCPListener로 assertion
}

// net.TCPListener의 Accept() 오버라이딩
func (ln stoppableListener) Accept() (c net.Conn, err error) {
	connc := make(chan *net.TCPConn, 1)
	errc := make(chan error, 1)
	go func() {
		tc, err := ln.AcceptTCP() // TCPConn을 얻음
		if err != nil {
			errc <- err
			return
		}
		connc <- tc
	}()
	select {
	case <-ln.stopc: // AcceptTCP에 의해 블록되는 동안 stopc, errc에서 종료신호가 오면 종료
		return nil, errors.New("server stopped")
	case err := <-errc:
		return nil, err
	case tc := <-connc: // TCPConn을 얻으면 keep-alive 설정후 리턴
		tc.SetKeepAlive(true)
		tc.SetKeepAlivePeriod(3 * time.Minute)
		return tc, nil
	}
}
```