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
	"syslog"
	"fmt"
	"strconv"
	"net"
	"os"
)

type ServerContext interface {
	GetLogLevel() syslog.Priority
	GetHandlerPoolSize() int
	GetShared() interface{}
	IsBlocking() bool
	SetShared(interface{})
	NewServerHandlerContext(uint32) ServerHandlerContext
}

type ServerHandlerContext interface {
	GetLogLevel() syslog.Priority
	GetId() uint32
	GetLogger() *syslog.Writer
	SetLogger(*syslog.Writer)
	Handle(*TcpConnection) os.Error
	Cleanup()
}

type BasicServerContext struct {
	logLevel syslog.Priority
	poolSize int
	blocking bool
}

type BasicServerHandlerContext struct {
	sCtx ServerContext
	id uint32
	logger *syslog.Writer
}

func NewBasicServerContext(logLevel syslog.Priority, poolSize int, blocking bool) (bsCtx *BasicServerContext) {
	bsCtx = &BasicServerContext{logLevel:logLevel, poolSize:poolSize, blocking:blocking}
	return
}

func (bsCtx *BasicServerContext) GetLogLevel() (level syslog.Priority) {
	level = bsCtx.logLevel
	return
}

func (bsCtx *BasicServerContext) GetHandlerPoolSize() (size int) {
	size = bsCtx.poolSize
	return
}

func (bsCtx *BasicServerContext) IsBlocking() (blocking bool) {
	blocking = bsCtx.blocking
	return
}

func (bsCtx *BasicServerContext) SetShared(shared interface{}) {
}

func (bsCtx *BasicServerContext) GetShared() (shared interface{}) {
	return 
}

func (bsCtx *BasicServerContext) NewServerHandlerContext(id uint32) (shCtx ServerHandlerContext) {
	shCtx = &BasicServerHandlerContext{sCtx:bsCtx, id:id}
	return shCtx
}

func (bshCtx *BasicServerHandlerContext) GetLogLevel() (level syslog.Priority) {
	level = bshCtx.sCtx.GetLogLevel()
	return 
}

func (bshCtx *BasicServerHandlerContext) GetId() (id uint32) {
	id = bshCtx.id
	return 
}

func (bshCtx *BasicServerHandlerContext) SetId(id uint32) {
	bshCtx.id = id
}

func (bshCtx *BasicServerHandlerContext) GetLogger() (logger *syslog.Writer) {
	logger = bshCtx.logger
	return 
}

func (bshCtx *BasicServerHandlerContext) SetLogger(logger *syslog.Writer) {
	bshCtx.logger = logger
}

func (bshCtx *BasicServerHandlerContext) Handle(*TcpConnection) os.Error {
	return nil
}

func (bshCtx *BasicServerHandlerContext) Cleanup() {
}

func handleConnections(connectionQueue chan *TcpConnection, shCtx ServerHandlerContext) {
	id := int(shCtx.GetId())
	logger, _ := syslog.New(shCtx.GetLogLevel(), "Handler " + strconv.Itoa(id))
	shCtx.SetLogger(logger)

	defer func ()  {
		shCtx.Cleanup()
		if r := recover(); r != nil {
			logger.Crit(fmt.Sprintf("Recovered in server handler %v\n", r))
		}
	}()

	for {
		connection := <-connectionQueue
		err := shCtx.Handle(connection)
		if err == os.EOF {
			logger.Info(fmt.Sprintf("Server handler is closing connection because remote peer has closed it: %q\n", connection.RemoteAddr()))
			connection.Close()
		} else if err != nil {
			logger.Warning(fmt.Sprintf("Server handler is closing connection due to error: %v\n", err))
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
	logger, _ := syslog.New(sCtx.GetLogLevel(), "AcceptLoop")
	
	//create a queue to share incoming connections
	connectionQueue := make(chan *TcpConnection)
	
	//create a number of handler goroutines to process connections
	for i := 0; i < sCtx.GetHandlerPoolSize(); i ++ {
		shCtx := sCtx.NewServerHandlerContext(id)
		go handleConnections(connectionQueue, shCtx)
		id ++
	}
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
			   logger.Crit(fmt.Sprintf("Server: Accept error: %v", err))
			   continue
			}
			logger.Crit(fmt.Sprintf("Server: fatal error: %v", err))
		}
		connection := NewTcpConnection(conn)
		connectionQueue <- connection
	}
	panic("not reached")
}

func listen(addr string) (listener net.Listener, err os.Error) {
	if addr == "" {
		addr = ":http"
	}
	return net.Listen("tcp", addr)
}

func ListenAndServe(addr string, sCtx ServerContext) os.Error {
	listener, err := listen(addr)
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
