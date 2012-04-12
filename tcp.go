// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
 * Copyright 2011 Moovweb Corp (zhigang.chen@moovweb.com). All rights reserved.
 */

//Package ptcp provides an extended capability to the basic tcp
package ptcp

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
)

// TcpConnection is a thin wrapper around TCP socket connection
type TcpConnection struct {
	tlsState *tls.ConnectionState //TLS state info
	rawData  *bytes.Buffer        //save the rawData as we parse it
	net.Conn                      //socket connection
}

//InitialBufferLength is the size of buffer allocated initially.
//The buffer can be expanded if needed
//The initial length should not be too big or small
const InitialBufferLength = 64 * 1024 //64K bytes

var (
	//Handshake failure
	ErrorTLSHandshake = errors.New("Handshake Failed")
	ErrorReadTimeout  = errors.New("Invalid Read Timeout")
	ErrorWriteTimeout = errors.New("Invalid Write Timeout")
)

//Wrap a tcp connection into a TcpConnection object
//
func NewTcpConnection(conn net.Conn) (connection *TcpConnection, err error) {
	if tlsConn, ok := conn.(*tls.Conn); ok {
		err = tlsConn.Handshake()
		if err != nil {
			return
		}
		tlsState := new(tls.ConnectionState)
		*tlsState = tlsConn.ConnectionState()
		if tlsState.HandshakeComplete {
			connection = &TcpConnection{Conn: conn, rawData: nil, tlsState: tlsState}
		} else {
			err = ErrorTLSHandshake
		}
	} else {
		connection = &TcpConnection{Conn: conn, rawData: nil}
	}
	return
}

func (connection *TcpConnection) EnableSaveReadData() {
	if connection.rawData == nil {
		buffer := make([]byte, 0, InitialBufferLength)
		connection.rawData = bytes.NewBuffer(buffer)
	}
}

func (connection *TcpConnection) DisableSaveReadData() {
	connection.rawData = nil
}

func (connection *TcpConnection) Read(data []byte) (n int, err error) {
	n, err = connection.Conn.Read(data) //calling the underlying socket's Read
	if (err == nil || err == io.EOF) && n > 0 && connection.rawData != nil {
		nn, err1 := connection.rawData.Write(data[:n])
		if err1 != nil {
			connection.rawData.Reset()
		}
		if nn != n {
			connection.rawData.Reset()
		}
	}
	return n, err
}

func (connection *TcpConnection) Close() error {
	if connection.rawData != nil {
		connection.rawData.Reset()
	}
	return connection.Conn.Close()
}

func (connection *TcpConnection) Reset() error {
	if connection.rawData != nil {
		connection.rawData.Reset()
	}
	return nil
}

func (connection *TcpConnection) RawData() (data []byte) {
	if connection.rawData != nil {
		return connection.rawData.Bytes()
	}
	return nil
}
