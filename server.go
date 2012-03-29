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
	"net"
	"os"
	"crypto/rand"
	"crypto/tls"
	"time"
	"log4go"
)

type ServerHandler interface {
	/*
		GetId() uint32
		GetLogConfig() *log4go.LogConfig
		GetServerTag() string
		GetLogger() log4go.Logger
	*/
	Handle(*TcpConnection) os.Error
	Cleanup()
	Spawn() (ServerHandler, os.Error)
	Logger() log4go.Logger
	Tag() string
	ConnectionQueueLength() int
}

var (
	ErrHandlerLimitReached     = os.NewError("Handler limit reached")
	ErrorClientCloseConnection = os.EOF
	ErrorServerCloseConnection = os.NewError("server needs to close the connection")
)

func handleConnections(connectionQueue chan *TcpConnection, h ServerHandler) {
	logger := h.Logger()
	defer func() {
		h.Cleanup()
		if r := recover(); r != nil {
			logger.Error("Recovered in server handler %v\n", r)
		}
	}()

	for {
		connection := <-connectionQueue
		err := h.Handle(connection)
		if err == ErrorClientCloseConnection {
			logger.Info("Server handler is closing connection because the client has closed it: %q", connection.RemoteAddr())
			connection.Close()
		} else if err == ErrorServerCloseConnection {
			logger.Info("Server handler is closing connection: %q", connection.RemoteAddr())
			connection.Close()
		} else if err != nil {
			logger.Warn("Server handler is closing connection due to error: %v", err)
			connection.Close()
		} else {
			//put it back into the queue
			connectionQueue <- connection
		}
	}
}

/*
 * Serve accepts incoming connections on the Listener l, creating a
 * new service thread for each.  The service threads read requests and
 * then call connection.Handler to reply to them.
 */

func serve(listener net.Listener, h ServerHandler) os.Error {
	defer listener.Close()

	logger := h.Logger()
	connQueueLen := h.ConnectionQueueLength()

	//create a queue to share incoming connections
	//allow the queue to buffer up to a given number of connections 
	connectionQueue := make(chan *TcpConnection, connQueueLen)
	count := 0
	for newH, err := h.Spawn(); err == nil; newH, err = h.Spawn() {
		go handleConnections(connectionQueue, newH)
		count++
	}
	logger.Info("Created %d handlers for server: %q", count, h.Tag())

	for {
		conn, err := listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				logger.Error("Server: Accept error: %v", err)
				continue
			}
			logger.Critical("Server: fatal error: %v", err)
		}
		connection, err := NewTcpConnection(conn)
		if err != nil {
			conn.Close()
		} else {
			connection.EnableSaveReadData()
			connectionQueue <- connection
		}
	}
	panic("should not be reached")
}

func listen(addr string, ssl bool) (listener net.Listener, err os.Error) {
	if addr == "" {
		if ssl {
			addr = ":https"
		} else {
			addr = ":http"
		}
	}
	return net.Listen("tcp", addr)
}

func ListenAndServe(addr string, h ServerHandler, blocking bool) os.Error {
	listener, err := listen(addr, false)
	if err != nil {
		return err
	}
	if blocking {
		serve(listener, h)
	} else {
		go serve(listener, h)
	}
	return nil
}

func ListenAndServeTLS(addr string, h ServerHandler, blocking bool, certFile string, keyFile string) os.Error {
	config := &tls.Config{
		Rand:       rand.Reader,
		Time:       time.Seconds,
		NextProtos: []string{"http/1.1"},
	}

	var err os.Error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	conn, err := listen(addr, true)
	if err != nil {
		return err
	}
	tlsListener := tls.NewListener(conn, config)
	if err != nil {
		return err
	}

	if blocking {
		serve(tlsListener, h)
	} else {
		go serve(tlsListener, h)
	}
	return nil
}
