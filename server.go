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
	"strconv"
	"net"
	"os"
	"crypto/rand"
	"crypto/tls"
	"time"
	"log4go"
)

type ServerContext interface {
	IsBlocking() bool
	GetLogger() log4go.Logger
	GetPoolSize() int
	NewServerHandlerContext(uint32) ServerHandlerContext
}

type ServerHandlerContext interface {
	GetLogger() log4go.Logger
	GetId() uint32
	Handle(*TcpConnection) os.Error
	Cleanup()
}

type BasicServerContext struct {
	PoolSize int
	Blocking bool
	Logger log4go.Logger
	LogConf *log4go.LogConfig
}

type BasicServerHandlerContext struct {
	ServerCtx ServerContext
	Id uint32
	Logger log4go.Logger
}

func NewBasicServerContext(logConfig *log4go.LogConfig, poolSize int, blocking bool) (bsCtx *BasicServerContext) {
	logger := log4go.NewLoggerFromConfig(logConfig, "proxy")
	bsCtx = &BasicServerContext{PoolSize:poolSize, Blocking:blocking, Logger:logger, LogConf:logConfig}
	return
}

func (bsCtx *BasicServerContext) NewServerHandlerContext(id uint32) (shCtx ServerHandlerContext) {
	idStr := strconv.Itoa(int(id))
	logger := log4go.NewLoggerFromConfig(bsCtx.LogConf, "proxy"+"("+ idStr + ")")
	shCtx = &BasicServerHandlerContext{ServerCtx:bsCtx, Id:id, Logger:logger}
	return shCtx
}

func (bsCtx *BasicServerContext) IsBlocking() bool {
	return bsCtx.Blocking
}

func (bsCtx *BasicServerContext) GetLogger() log4go.Logger {
	return bsCtx.Logger
}

func (bsCtx *BasicServerContext) GetPoolSize() int {
	return bsCtx.PoolSize
}

func (bshCtx *BasicServerHandlerContext) Handle(*TcpConnection) os.Error {
	return nil
}

func (bshCtx *BasicServerHandlerContext) Cleanup() {
}

func (bshCtx *BasicServerHandlerContext) GetLogger() log4go.Logger {
	return bshCtx.Logger
}

func (bshCtx *BasicServerHandlerContext) GetId() uint32 {
	return bshCtx.Id
}

func handleConnections(connectionQueue chan *TcpConnection, shCtx ServerHandlerContext) {
	logger := shCtx.GetLogger()
	defer func ()  {
		shCtx.Cleanup()
		if r := recover(); r != nil {
			logger.Error("Recovered in server handler %v\n", r)
		}
	}()

	for {
		connection := <-connectionQueue
		err := shCtx.Handle(connection)
		if err == os.EOF {
			logger.Info("Server handler is closing connection because remote peer has closed it: %q", connection.RemoteAddr())
			connection.Close()
		} else if err == os.NewError("client required the connection be closed\n") {
			//ok
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

func serve(listener net.Listener, sCtx ServerContext) os.Error {
	defer listener.Close()
	var id uint32 = 0
	
	//logger for the accept loop
	logger := sCtx.GetLogger()
	//get the handler pool size
	poolSize := sCtx.GetPoolSize()
	
	//create a queue to share incoming connections
	//allow the queue to buffer up to poolSize connections
	connectionQueue := make(chan *TcpConnection, poolSize)
	
	//create a number of handler goroutines to process connections
	for i := 0; i < poolSize; i ++ {
		shCtx := sCtx.NewServerHandlerContext(id)
		go handleConnections(connectionQueue, shCtx)
		id ++
	}
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
			   logger.Error("Server: Accept error: %v", err)
			   continue
			}
			logger.Critical("Server: fatal error: %v", err)
		}
		connection := NewTcpConnection(conn)
		connectionQueue <- connection
	}
	panic("not reached")
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

func ListenAndServe(addr string, sCtx ServerContext) os.Error {
	listener, err := listen(addr, false)
	if err != nil {
		return err
	}
	if sCtx.IsBlocking() {
		serve(listener, sCtx)
	} else {
		go serve(listener, sCtx)
	}
	return nil
}

func ListenAndServeTLS(addr string, sCtx ServerContext, certFile string, keyFile string) os.Error {
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
	
	if sCtx.IsBlocking() {
		serve(tlsListener, sCtx)
	} else {
		go serve(tlsListener, sCtx)
	}
	return nil
}
