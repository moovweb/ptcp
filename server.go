// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Copyright 2011 Moovweb Corp (zhigang.chen@moovweb.com). All rights reserved.
 */

/*
 * Provide a basic TCP server.
 */

package ptcp

import (
	"log"
	"net"
	"os"
)

func listen(addr string) (listener net.Listener, err os.Error) {
	if addr == "" {
		addr = ":http"
	}
	return net.Listen("tcp", addr)
}

/*
 * Serve accepts incoming connections on the Listener l, creating a
 * new service thread for each.  The service threads read requests and
 * then call connection.Handler to reply to them.
 */

func serve(listener net.Listener, NewServerHandler func() ServerHandler, poolSize int, saveReadData bool) os.Error {
	defer listener.Close()
	
	//create a queue to share incoming connections
	connectionQueue := make(chan *TcpConnection)
	
	//create a number of handler goroutines to process connections
	for i := 0; i < poolSize; i ++ {
		go handleConnections(connectionQueue, NewServerHandler)
	}
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
			   log.Printf("Server: Accept error: %v", err)
			   continue
			}
			log.Fatalf("Server: fatal error: %v", err)
		}
		connection := NewTcpConnection(conn)
		if saveReadData {
			connection.EnableSaveReadData()
		}
		connectionQueue <- connection
	}
	panic("not reached")
}

func handleConnections(connectionQueue chan *TcpConnection, NewServerHandler func() ServerHandler) {
	handler := NewServerHandler()
	defer func ()  {
		handler.Cleanup()
		if r := recover(); r != nil {
			log.Printf("Recovered in server handler %v\n", r)
		}
	}()
	for {
		connection := <-connectionQueue
		err := handler.Handle(connection)
		if err == os.EOF {
			log.Printf("Server handler is closing connection because remote peer has closed it: %q\n", connection.RemoteAddr())
			connection.Close()
		} else if err != nil {
			log.Printf("Server handler is closing connection due to error: %v\n", err)
			connection.Close()
		} else {
			//put it back into the queue
			connectionQueue <- connection
		}
	}
}

func ListenAndServe(addr string, NewServerHandler func() ServerHandler, poolSize int, block bool, saveReadData bool) os.Error {
	listener, err := listen(addr)
	if err != nil {
		return err
	}
	if block {
		serve(listener, NewServerHandler, poolSize, saveReadData)
	} else {
		go serve(listener, NewServerHandler, poolSize, saveReadData)
	}
	return nil
}

type ServerHandler interface {
	Cleanup()
	Handle(*TcpConnection) os.Error
}
