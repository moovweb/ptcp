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
	IsBlocking() bool
	GetLogger() *syslog.Writer
	GetPoolSize() int
	NewServerHandlerContext(uint32) ServerHandlerContext
}

type ServerHandlerContext interface {
	GetLogger() *syslog.Writer
	GetId() uint32
	Handle(*TcpConnection) os.Error
	Cleanup()
}

type BasicServerContext struct {
	PoolSize int
	Blocking bool
	Logger *syslog.Writer
	LogLevel int
	LoggerPrefix string
}

type BasicServerHandlerContext struct {
	ServerCtx ServerContext
	Id uint32
	Logger *syslog.Writer
}

func NewBasicServerContext(logLevel int, poolSize int, blocking bool, loggerPrefix string) (bsCtx *BasicServerContext) {
	logger, err := syslog.New((syslog.Priority)(logLevel), loggerPrefix)
	if err != nil {
		panic("cannot write to syslog in basic server")
	}
	bsCtx = &BasicServerContext{PoolSize:poolSize, Blocking:blocking, Logger:logger, LogLevel:logLevel, LoggerPrefix: loggerPrefix}
	return
}

func (bsCtx *BasicServerContext) NewServerHandlerContext(id uint32) (shCtx ServerHandlerContext) {
	idStr := strconv.Itoa(int(id))
	logger, err := syslog.New((syslog.Priority)(bsCtx.LogLevel), bsCtx.LoggerPrefix + "("+ idStr + ")")
	if err != nil {
		panic("cannot write to syslog in basic server handler: " + idStr)
	}
	shCtx = &BasicServerHandlerContext{ServerCtx:bsCtx, Id:id, Logger:logger}
	return shCtx
}

func (bsCtx *BasicServerContext) IsBlocking() bool {
	return bsCtx.Blocking
}

func (bsCtx *BasicServerContext) GetLogger() *syslog.Writer {
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

func (bshCtx *BasicServerHandlerContext) GetLogger() *syslog.Writer {
	return bshCtx.Logger
}

func (bshCtx *BasicServerHandlerContext) GetId() uint32 {
	return bshCtx.Id
}

func handleConnections(connectionQueue chan *TcpConnection, shCtx ServerHandlerContext) {
	logger := shCtx.GetLogger()
	defer func ()  {
		shCtx.Cleanup()
		/*if r := recover(); r != nil {
			logger.Crit(fmt.Sprintf("Recovered in server handler %v\n", r))
		}*/
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
