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
	"net"
	"os"
	"crypto/tls"
)

// TcpConnection is a thin wrapper around TCP socket connection
type TcpConnection struct {
	tlsState   *tls.ConnectionState //TLS state info
  	readTimeout	int64			//read timeout
	writeTimeout   int64		//write timeout
	rawData		*bytes.Buffer	//save the rawData as we parse it
  	net.Conn					//socket connection
}

//InitialBufferLength is the size of buffer allocated initially.
//The buffer can be expanded if needed
//The initial length should not be too big or small
const InitialBufferLength = 64*1024 //64K bytes

var (
	//Handshake failure
	ErrorTLSHandshake = os.NewError("Handshake Failed")
	ErrorReadTimeout = os.NewError("Invalid Read Timeout")
	ErrorWriteTimeout = os.NewError("Invalid Write Timeout")
)

//Wrap a tcp connection into a TcpConnection object
//
func NewTcpConnection(conn net.Conn) (connection *TcpConnection, err os.Error) {
	if tlsConn, ok := conn.(*tls.Conn); ok {
		tlsConn.Handshake()
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

func (connection *TcpConnection ) SetReadTimeout(readTimeout int64) (err os.Error) {
	if readTimeout > 0 {
		connection.readTimeout = readTimeout
		return connection.Conn.SetReadTimeout(connection.readTimeout)
	}
	return ErrorReadTimeout
}

func (connection *TcpConnection ) SetWriteTimeout(writeTimeout int64) (err os.Error) {
	if writeTimeout > 0 {
		connection.writeTimeout = writeTimeout
		return connection.Conn.SetWriteTimeout(connection.writeTimeout)
	}
	return ErrorWriteTimeout
}

func (connection *TcpConnection) EnableSaveReadData() {
	buffer := make([]byte, 0, InitialBufferLength)
	connection.rawData = bytes.NewBuffer(buffer)
}

func (connection *TcpConnection) DisableSaveReadData() {
	connection.rawData = nil
}

func (connection *TcpConnection) Read(data []byte) (n int, err os.Error) {
	n, err = connection.Conn.Read(data) //calling the underlying socket's Read
	if (err == nil || err == os.EOF) && n > 0 && connection.rawData != nil {
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

func (connection *TcpConnection) Close() os.Error {
	if connection.rawData != nil {
		connection.rawData.Reset()
	}
	return connection.Conn.Close()
}

func (connection *TcpConnection) Reset() os.Error {
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
