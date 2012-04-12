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
	"crypto/tls"
	"errors"
	"net"
)

var ErrorMissingClientHandler = errors.New("Client missing a handler")

type Request interface {
	Bytes() []byte
}

type Response interface {
	Bytes() []byte
}

func Connect(addr string) (connection *TcpConnection, err error) {
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

func ConnectTLS(addr string, hostName string, shouldVerifyHost bool) (connection *TcpConnection, err error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	// Initiate TLS and check remote host name against certificate.
	tlsConn := tls.Client(conn, nil)

	connection, err = NewTcpConnection(tlsConn)
	if err != nil {
		tlsConn.Close()
	} else if shouldVerifyHost {
		if err = tlsConn.VerifyHostname(hostName); err != nil {
			tlsConn.Close()
			return
		}
	}
	return
}

func SendAndReceive(connection *TcpConnection, handler ClientHandler, request Request) (Response, error) {
	if handler == nil {
		return nil, ErrorMissingClientHandler
	}
	return handler.Handle(connection, request)
}

type ClientHandler interface {
	Handle(*TcpConnection, Request) (Response, error)
}
