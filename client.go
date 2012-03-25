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

var (
	ErrorMissingClientHandler = os.NewError("Client missing a handler")
)

func Connect(addr string) (connection *TcpConnection, err os.Error) {
	if addr == "" {
		addr = ":http"
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}
	connection, err = NewTcpConnection(conn)
	return
}

func ConnectTLS(addr string, hostName string, shouldVerifyHost bool) (connection *TcpConnection, err os.Error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	// Initiate TLS and check remote host name against certificate.
	tlsConn := tls.Client(conn, nil)

	connection, err = NewTcpConnection(tlsConn)
	if err != nil { //retry once if handshake failed
		conn, err = net.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
		tlsConn = tls.Client(conn, nil)
		connection, err = NewTcpConnection(tlsConn)
	}
	if err == nil && shouldVerifyHost {
		if err = tlsConn.VerifyHostname(hostName); err != nil {
			return
		}
	}
	return
}


func SendAndReceive(connection *TcpConnection, handler ClientHandler, request interface{}) ([]byte, interface{}, os.Error) {
	if handler == nil {
		return nil, nil, ErrorMissingClientHandler
	}
	return handler.Handle(connection, request)
}

type ClientHandler interface {
	Handle(*TcpConnection, interface{}) ([]byte, interface{}, os.Error)
}

