package server

import (
	"io/ioutil"
	"log"
	"net"
	"sync"
)

// ConnHandler encapsulates both an OnConn function that the server calls whenever there's
// a new connection and a Close function that's called when the server is closing down.
// The OnConn function should be non-blocking so that the server can quickly receive other
// connections. The ConnHandler is responsible for closing connections once done
type ConnHandler interface {
	OnConn(conn net.Conn)
	Close()
}

// Server handles network details such as receiving new connections.
// structuring credit: https://eli.thegreenplace.net/2020/graceful-shutdown-of-a-tcp-server-in-go/#
type Server struct {
	listener net.Listener
	quitCh   chan struct{}
	onceStop sync.Once
	handler  ConnHandler
	logger   *log.Logger
}

// NewServer sets up server plus starts listening on new client connections
// in separate go routine. Callers should find a way to block
func NewServer(addr string, handler ConnHandler) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s := &Server{
		listener: listener,
		quitCh:   make(chan struct{}),
		handler:  handler,
		logger:   log.New(ioutil.Discard, "", 0),
	}

	go s.receiveConnections()

	s.logger.Printf("server started on address: %s\n", s.listener.Addr())
	return s, nil
}

func (s *Server) receiveConnections() {
	for {
		select {
		case <-s.quitCh:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				s.logger.Println("server accept error:", err)
				continue
			}
			if conn == nil {
				s.logger.Println("server create connection error:")
				continue
			}
			s.handler.OnConn(conn)
		}

	}
}

// Stop shuts down server, waits for client sessions to close first
// safe to call even if server not started
func (s *Server) Stop() {
	s.onceStop.Do(func() {
		close(s.quitCh) // broadcast quit
		s.listener.Close()
		s.handler.Close()
		s.logger.Println("server closed")
	})
}
