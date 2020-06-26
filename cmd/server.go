package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/go-uuid"
	"github.com/pkg/errors"
)

// Server
type server struct {
	connMap *sync.Map
	once    *sync.Once
}

func newServer() *server {
	return &server{
		connMap: &sync.Map{},
		once:    &sync.Once{},
	}
}

func (s *server) run() error {
	l, err := net.Listen("tcp", "localhost:9000")
	if err != nil {
		return err
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		id, _ := uuid.GenerateUUID()
		s.connMap.Store(id, conn)
		go s.handleNewConnection(id, conn)
	}
}

func (s *server) handleNewConnection(id string, conn net.Conn) {
	defer func() {
		s.connMap.Delete(id)
		conn.Close()
	}()
	closeCh := make(chan struct{})
	go func() {
		time.Sleep(10 * time.Second)
		close(closeCh)
	}()
	newClientSession(id, conn, s.handleMessage, closeCh).run()
}

func (s *server) handleMessage(msg []byte) {
	s.connMap.Range(func(key, value interface{}) bool {
		if conn, ok := value.(net.Conn); ok {
			conn.Write([]byte(msg))
		}
		return true
	})
}

//

// Client Session
type clientSession struct {
	id            string
	conn          net.Conn
	onSendMessage func(msg []byte)
	trace         ClientSessionTrace
	closeCh       <-chan struct{}
}

func newClientSession(id string, conn net.Conn, handleSendMessage func(msg []byte), closeCh <-chan struct{}) *clientSession {
	var trace ClientSessionTrace

	// trace session length
	// trace = trace.Compose(ClientSessionTrace{
	// 	OnRun: func() func(error) {
	// 		start := time.Now()
	// 		fmt.Println("session start")
	// 		return func(err error) {
	// 			fmt.Println("session close. time:", time.Since(start))
	// 		}
	// 	},
	// })

	// trace session error
	type stackTracer interface {
		StackTrace() errors.StackTrace
	}

	trace = trace.Compose(ClientSessionTrace{
		OnRun: func() func(error) {
			return func(err error) {
				if err == nil {
					fmt.Println("no err")
					return
				}
				//fmt.Println(err)
				if errors.Is(err, io.EOF) {
					fmt.Println("io.EOF")
				} else {
					cause := errors.Cause(err)
					fmt.Println("cause:", cause)
				}

				// if err, ok := err.(stackTracer); ok {
				// 	for _, f := range err.StackTrace()[:2] {
				// 		// [source file]:[source line]:[fn]
				// 		fmt.Printf("\t%s:%d:\n\t%n\n", f, f, f)
				// 	}
				// }
			}
		},
	})
	return &clientSession{id, conn, handleSendMessage, trace, closeCh}
}

func (c *clientSession) run() (err error) {
	done := c.trace.OnRun()
	r := bufio.NewReader(c.conn)
run:
	for {
		select {
		case <-c.closeCh:
			err = fmt.Errorf("\n[closeChErr]")
			break run
		default:
			var userMsg string
			userMsg, err = r.ReadString('\n')
			if err != nil {
				break run
			}
			c.onSendMessage([]byte(userMsg))
		}
	}
	if err != nil {
		err = errors.Wrap(err, "[RUN]\n")
	}

	done(err)
	return
}

// ClientSessionTrace client session trace
type ClientSessionTrace struct {
	OnRun func() func(error)
}

// Compose ...
func (a ClientSessionTrace) Compose(b ClientSessionTrace) (c ClientSessionTrace) {
	switch {
	case a.OnRun == nil:
		c.OnRun = b.OnRun
	case b.OnRun == nil:
		c.OnRun = a.OnRun
	default:
		c.OnRun = func() func(error) {
			doneA := a.OnRun()
			doneB := b.OnRun()
			switch {
			case doneA == nil:
				return doneB
			case doneB == nil:
				return doneA
			default:
				return func(err error) {
					doneA(err)
					doneB(err)
				}
			}
		}
	}
	return c
}
