package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testHandler struct {
	wg sync.WaitGroup
}

func (h *testHandler) OnConn(conn net.Conn) {
	h.wg.Add(1) // add new client conn
	go func() {
		h.handleConn(conn)
		h.wg.Done() // indicate client done
	}()

}

func (h *testHandler) handleConn(conn net.Conn) {
	// conn.SetDeadline(time.Now().Add(1 * time.Second))
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		bs, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}
		conn.Write(append(bs, '\n'))
	}
}

func (h *testHandler) Close() {
	h.wg.Wait()
}

func TestServerStartStop(t *testing.T) {
	s, err := NewServer(":0", &testHandler{})
	require.NoError(t, err)
	s.Stop()
}

func TestServerSimpleHandleFastClients(t *testing.T) {
	var clientsConnected, clientsDisconnected sync.WaitGroup

	// clients connect to server, write trivial message then
	// wait for 100 milliseconds before disconnecting
	clientConnect := func(addr net.Addr) {
		conn, err := net.Dial(addr.Network(), addr.String())
		if err != nil {
			log.Fatal(err)
		}
		clientsConnected.Done()

		fmt.Fprintf(conn, "client conn msg")
		time.Sleep(100 * time.Millisecond)
		conn.Close()

		clientsDisconnected.Done()
	}

	// set up server
	s, err := NewServer(":0", &testHandler{})
	require.NoError(t, err)
	addr := s.listener.Addr()

	// set up N clients
	N := 2
	clientsConnected.Add(N)
	clientsDisconnected.Add(N)
	for i := 0; i < N; i++ {
		go clientConnect(addr)
	}

	// stop server before clients disconnect
	clientsConnected.Wait()
	s.Stop()
	clientsDisconnected.Wait()

	// should not be able to connect after stop
	_, err = net.Dial(addr.Network(), addr.String())
	if err == nil {
		t.Errorf("Expected connection error on client connect attempt after server stop")
	}
}
