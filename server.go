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
	"fmt"
)

//server state info, shared by all server handlers
type ServerContext interface {
	GetPoolSize() int
	GetLogConfig() *log4go.LogConfig
	GetServerTag() string
	IsBlocking() bool
	ShouldSaveReadData() bool
	GetServerHandlerContextConstructor() func(uint32, ServerContext) ServerHandlerContext
	Cleanup()
}

//server handler state info
type ServerHandlerContext interface {
	GetServerContext() ServerContext
	GetId() uint32
	GetLogConfig() *log4go.LogConfig
	GetServerTag() string
	GetLogger() log4go.Logger
	Handle(*TcpConnection) os.Error
	Cleanup()
}

type DefaultServerContext struct {
	ServerTag string
	PoolSize int
	Blocking bool
	SaveReadData bool
	LogConfig *log4go.LogConfig
	ServerHandlerContextConstructor func(uint32, ServerContext) ServerHandlerContext
}

type DefaultServerHandlerContext struct {
	ServerCtx ServerContext
	Id uint32
	Logger log4go.Logger
}

var (
	ErrorClientCloseConnection = os.EOF
	ErrorServerCloseConnection = os.NewError("server needs to close the connection")
)

func NewDefaultServerContext(logConfig *log4go.LogConfig, poolSize int, blocking bool, serverTag string) (defaultServerCtx *DefaultServerContext) {
	defaultServerCtx = &DefaultServerContext{PoolSize:poolSize, Blocking:blocking, LogConfig:logConfig, ServerTag: serverTag}
	return
}

func NewDefaultServerHandlerContext(id uint32, serverCtx ServerContext) (defaultServerHandlerCtx *DefaultServerHandlerContext) {
	defaultServerHandlerCtx = &DefaultServerHandlerContext{ServerCtx:serverCtx, Id:id}
	return
}

func (defaultServerCtx *DefaultServerContext) IsBlocking() bool {
	return defaultServerCtx.Blocking
}

func (defaultServerCtx *DefaultServerContext) ShouldSaveReadData() bool {
	return defaultServerCtx.SaveReadData
}

func (defaultServerCtx *DefaultServerContext) GetPoolSize() int {
	return defaultServerCtx.PoolSize
}

func (defaultServerCtx *DefaultServerContext) GetLogConfig() *log4go.LogConfig {
	return defaultServerCtx.LogConfig
}

func (defaultServerCtx *DefaultServerContext) GetServerTag() string {
	return defaultServerCtx.ServerTag
}

func (defaultServerCtx *DefaultServerContext) GetServerHandlerContextConstructor() (constructor func(uint32, ServerContext) ServerHandlerContext) {
	constructor = defaultServerCtx.ServerHandlerContextConstructor
	return 
}

func (defaultServerCtx *DefaultServerContext) Cleanup() {
}

//--------------------------------------

func (defaultServerHandlerCtx *DefaultServerHandlerContext) Handle(*TcpConnection) os.Error {
	return nil
}

func (defaultServerHandlerCtx *DefaultServerHandlerContext) GetId() uint32 {
	return defaultServerHandlerCtx.Id
}

func (defaultServerHandlerCtx *DefaultServerHandlerContext) GetLogConfig() *log4go.LogConfig {
	return defaultServerHandlerCtx.GetServerContext().GetLogConfig()
}

func (defaultServerHandlerCtx *DefaultServerHandlerContext) GetLogger() log4go.Logger {
	if defaultServerHandlerCtx.Logger == nil {
		logPrefix := fmt.Sprintf("%v (%d)", defaultServerHandlerCtx.GetServerTag(), defaultServerHandlerCtx.GetId())
		logConfig := defaultServerHandlerCtx.GetLogConfig()
		defaultServerHandlerCtx.Logger = log4go.NewLoggerFromConfig(logConfig, logPrefix)
	}
	return defaultServerHandlerCtx.Logger
}

func (defaultServerHandlerCtx *DefaultServerHandlerContext) GetServerTag() string {
	return defaultServerHandlerCtx.GetServerContext().GetServerTag()
}

func (defaultServerHandlerCtx *DefaultServerHandlerContext) GetServerContext() (serverCtx ServerContext) {
	serverCtx = defaultServerHandlerCtx.ServerCtx
	return
}

func (bshCtx *DefaultServerHandlerContext) Cleanup() {
}

func handleConnections(connectionQueue chan *TcpConnection, serverHandlerCtx ServerHandlerContext) {
	logger := serverHandlerCtx.GetLogger()
	defer func ()  {
		serverHandlerCtx.Cleanup()
		if r := recover(); r != nil {
			logger.Error("Recovered in server handler %v\n", r)
		}
	}()

	for {
		connection := <-connectionQueue
		err := serverHandlerCtx.Handle(connection)
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

func serve(listener net.Listener, serverCtx ServerContext) os.Error {
	defer listener.Close()
	
	serverTag := serverCtx.GetServerTag()
	//create a logger with the proper prefix and config
	logPrefix := fmt.Sprintf("%v", serverTag)
	logConfig := serverCtx.GetLogConfig()
	logger := log4go.NewLoggerFromConfig(logConfig, logPrefix)

	var id uint32 = 0
	
	//get the handler pool size
	poolSize := serverCtx.GetPoolSize()
	
	//need at least one handler
	if poolSize <= 0 {
		logger.Error("You need at least one handler for server: %q", serverTag)
		panic("Need at least one handler for server")
	}
	
	//create a queue to share incoming connections
	//allow the queue to buffer up to poolSize connections
	connectionQueue := make(chan *TcpConnection, poolSize)
	
	//get the handler constructor
	handlerConstructor := serverCtx.GetServerHandlerContextConstructor()
	//create a number of handler goroutines to process connections
	for i := 0; i < poolSize; i ++ {
		serverHandlerCtx := handlerConstructor(id, serverCtx)
		go handleConnections(connectionQueue, serverHandlerCtx)
		id ++
	}
	logger.Info("Created %d handlers for server: %q", poolSize, serverCtx.GetServerTag())
	
	saveReadData := serverCtx.ShouldSaveReadData()
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
			//logger.Warn("Failed to create TCP connection: %v", err)
			conn.Close()
		} else {
			if saveReadData {
				connection.EnableSaveReadData()
			}
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

func ListenAndServe(addr string, serverCtx ServerContext) os.Error {
	listener, err := listen(addr, false)
	if err != nil {
		return err
	}
	if serverCtx.IsBlocking() {
		serve(listener, serverCtx)
	} else {
		go serve(listener, serverCtx)
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
