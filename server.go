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
	"log"
	"net"
	"os"
)

func listen(addr string) (listener net.Listener, err os.Error) {
    if addr == "" {
        addr = ":http"
    }
    return net.Listen("tcp", addr)
}

/*
 * Serve accepts incoming connections on the Listener l, creating a
 * new service thread for each.  The service threads read requests and
 * then call connection.Handler to reply to them.
 */

func serve(listener net.Listener, handler ServerHandler, saveReadData bool) os.Error {
    defer listener.Close()

    //A server handler routine sends -1/1, indicating it starts handling on a connection or closes it.
    connectionCounterWriter := make(chan int)
    //get the current counter
    connectionCounterReader := make(chan int)
	
	//give each connection an id
	connId := 0

    //the go routine that maintains the connection counter
    //it blocks on receiving an counter update or sending the current counter 
    go func() {
        counter := 0
        for {
            select {
            case connectionCounterReader <- counter:
            case update := <- connectionCounterWriter:
                counter += update
            }
        }    
    }()

    for {
            conn, err := listener.Accept()
            if err != nil {
                if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
                    log.Printf("Server: Accept error: %v", err)
                    continue
                }
                log.Fatalf("Server: fatal error: %v", err)
            }
            connection := NewTcpConnection(conn)
            if saveReadData {
                connection.EnableSaveReadData()
            }
            if handler == nil {
                log.Fatalf("Server must provide a handler")
            }
            go func() {
                //after exiting the loop, the connection is closed
                defer func ()  {
                    connection.Close()
                    /*
                    if r := recover(); r != nil {
                        _, file, line, ok := runtime.Caller(0)
                        if ok {
                            log.Printf("caller is %q:%d\n", file, line)
                        }
                        log.Printf("Recovered in server handler %v\n", r)
                    }*/
                    connectionCounterWriter <- -1
                }()
                connectionCounterWriter <- 1
                handler.SetConnectionCounterReader(connectionCounterReader)
                for ; ; {
                    err := handler.Handle(connection)
                    if err == os.EOF {
                        log.Printf("Server handler closed connection because remote peer closed connection: %q\n", connection.RemoteAddr())
                        break
                    } else if err != nil {
                        log.Printf("Server handler closed connection due to error: %v\n", err)
                        break
                    }
                }
            }()
			connId += 1
    }
    panic("not reached")
}

func ListenAndServe(addr string, handler ServerHandler, saveReadData bool, block bool) os.Error {
    listener, err := listen(addr)
    if err != nil {
        return err
    }
    if block {
        serve(listener, handler, saveReadData)
    } else {
        go serve(listener, handler, saveReadData)
    }
    
    return nil
}

type ServerHandler interface {
    Handle(*TcpConnection) os.Error
    SetConnectionCounterReader(connectionCounterReader chan int)
}
