package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// HandleConnection is the entry point that's called once a
// client connection is received. Handles I/O for the client or delegates
// to other handlers. Should be called within a gorouting to avoid blocking the main
// goroutine
func handleConnection(conn net.Conn) {
	// conn.SetDeadline(time.Now().Add(1 * time.Second))
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		bs, err := reader.ReadBytes('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Println("Failed to read data, err:", err)
			}
			return
		}
		//n, err := conn.Write(bs)
		// restore '\n'
		conn.Write(bs)
	}
}

func TestServerStartStop(t *testing.T) {
	s, err := NewServer(":0", handleConnection)
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
		defer conn.Close()
		fmt.Fprintf(conn, "client conn msg")
		time.Sleep(100 * time.Millisecond)
		conn.Close()
		clientsDisconnected.Done()
	}

	// set up server
	s, err := NewServer(":0", handleConnection)
	require.NoError(t, err)
	addr := s.listener.Addr()

	// set up 2 clients
	clientsConnected.Add(2)
	clientsDisconnected.Add(2)
	go clientConnect(addr)
	go clientConnect(addr)

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
