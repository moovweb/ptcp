// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Copyright 2011 Moovweb Corp (zhigang.chen@moovweb.com). All rights reserved.
 */

/*
 * Provide a basic TCP client with SSL support.
 */

package ptcp

import (
	"net"
	"os"
	"crypto/tls"
)

func Connect(addr string) (connection *TcpConnection, err os.Error) {
	if addr == "" {
		addr = ":http"
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	connection = NewTcpConnection(conn)
	return connection, nil
}

func ConnectTLS(addr string, hostName string) (connection *TcpConnection, err os.Error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	// Initiate TLS and check remote host name against certificate.
	tlsConn := tls.Client(conn, nil)
	if err = tlsConn.Handshake(); err != nil {
		return nil, err
	}
	if err = tlsConn.VerifyHostname(hostName); err != nil {
		return nil, err
	}
	connection = NewTcpConnection(tlsConn)
	return connection, nil
}


func SendAndReceive(connection *TcpConnection, handler ClientHandler, request interface{}) ([]byte, interface{}, os.Error) {
	if handler == nil {
		return nil, nil, os.NewError("client must provide a handler")
	}
	return handler.Handle(connection, request)
}

type ClientHandler interface {
	Handle(*TcpConnection, interface{}) ([]byte, interface{}, os.Error)
}

