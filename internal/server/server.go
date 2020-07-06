package server

import (
	"fmt"
	"net"
	"sync"
)

// OnConn is a callback that's called by the server whenever it receives
// a new connection. It should be non-blocking so that server can quickly receive
// other connections
type OnConn func(conn net.Conn)

// Server handles network details such as receiving new connections.
// structuring credit: https://eli.thegreenplace.net/2020/graceful-shutdown-of-a-tcp-server-in-go/#
type Server struct {
	listener net.Listener
	quit     chan struct{}
	wg       sync.WaitGroup
	onceStop sync.Once
	onConn   OnConn
}

// NewServer sets up server plus starts listening on new client connections
// in separate go routine. Callers should find a way to block
func NewServer(addr string, onConn OnConn) (*Server, error) {
	s := new(Server)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s.listener = listener
	s.quit = make(chan struct{})
	s.onConn = onConn

	go s.receiveConnections()

	fmt.Printf("server started on address: %s\n", s.listener.Addr())
	return s, nil
}

func (s *Server) receiveConnections() {
	s.wg.Add(1)
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			// check for quits first
			select {
			case <-s.quit:
				return
			default:
				fmt.Println("server accept error:", err)
			}
			continue
		}
		if conn == nil {
			fmt.Println("server create connection error:")
			continue
		}
		s.wg.Add(1) // add new client conn
		go func() {
			s.onConn(conn)
			s.wg.Done() // indicate client done
		}()
	}
}

// Stop shuts down server, waits for client sessions to close first
// safe to call even if server not started
func (s *Server) Stop() {
	s.onceStop.Do(func() {
		close(s.quit) // broadcast quit
		s.listener.Close()
		s.wg.Wait() // wait for clients ?
		fmt.Println("server closed")
	})
}
