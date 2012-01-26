package ptcp

import (
	"testing"
	"os"
	"log4go"
	"fmt"
)

const requestBufferSize = 1000

type EchoServerContext struct {
	*BasicServerContext
	message string
}

type EchoServerHandlerContext struct {
	*BasicServerHandlerContext
	Buffer []byte
}

func NewEchoServerContext(message string, blocking bool, numHandlers int) ServerContext {
	esCtx := &EchoServerContext{message:message}
	logConfig := &log4go.LogConfig{ConsoleLogLevel: int(log4go.DEBUG), SysLogLevel: int(log4go.DEBUG)}
	esCtx.BasicServerContext = NewBasicServerContext(logConfig, numHandlers, blocking, "EchoServer")
	return esCtx
}

func (esCtx *EchoServerContext) NewServerHandlerContext(id uint32) ServerHandlerContext {
	bsCtx := esCtx.BasicServerContext
	shCtx := bsCtx.NewServerHandlerContext(id)
	eshCtx := &EchoServerHandlerContext{}
	eshCtx.BasicServerHandlerContext = shCtx.(*BasicServerHandlerContext)
	eshCtx.Buffer = make([]byte, requestBufferSize)
	//import!!! we need to set the server context to the caller object; otherwise it is pointed to eshCtx.BasicServerContext
	eshCtx.ServerCtx = esCtx
	return eshCtx
}

func (eshCtx *EchoServerHandlerContext) Handle (connection *TcpConnection) (err os.Error) {
	n, err := connection.Read(eshCtx.Buffer)
	if err != nil {
		return
	}
	if n <= 0 {
		return os.NewError("should receive more than 0 bytes")
	}
	request := eshCtx.Buffer[0:n]
	message := eshCtx.ServerCtx.(*EchoServerContext).message
	logPrefix := fmt.Sprintf("%v (%d)", eshCtx.GetServerTag(), eshCtx.GetId())
	logConfig := eshCtx.GetLogConfig()
	logger := log4go.NewLoggerFromConfig(logConfig, logPrefix)
	if (message != string(request)) {
		logger.Error("Wrong Message")
	}
	_, err = connection.Write(request)
	return err
}

type EchoClientHandler struct {
	Buffer []byte
}

func NewEchoClientHandler() *EchoClientHandler {
	ech := &EchoClientHandler{}
	ech.Buffer = make([]byte, requestBufferSize)
	return ech
}

func (ech *EchoClientHandler) Handle (connection *TcpConnection, request interface {}) (rawResponse []byte, response interface{}, err os.Error) {
	requestBytes, ok := request.([]byte)
	if ! ok {
		return nil, nil, os.NewError("EchoClientHandler cannot convert request into bytes")
	}
	_, err = connection.Write(requestBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "err : %v\n", err)
		return nil, nil, err
	}
	n, err := connection.Read(ech.Buffer)
	if err == os.EOF {
		err = nil
	}
	if err != nil {
		return nil, nil, err
	}
	return ech.Buffer[:n], ech.Buffer[:n], nil
}

func TestEcho(t *testing.T) {
	address := "localhost:9090"
	message := "hello world"
	clientHandler := NewEchoClientHandler()
	sCtx := NewEchoServerContext(message, false, 10)
	ListenAndServe(address, sCtx)
	for i := 0; i < 10; i++ {
		connection, err := Connect(address)
		connection.EnableSaveReadData()
		if err != nil {
			t.Error("error when connecting to %s: $v\n", address, err) 
		}
		go func() {
			defer connection.Close()
			rawResponse, _, err := SendAndReceive(connection, clientHandler, ([]byte)(message))
			if string(rawResponse) != message || err != nil {
				t.Errorf("failed in eccho \"hello world\": err: %v; received %q, expected %q\n", err, string(rawResponse), message)
			}
		}()
	}
}

func BenchmarkEcho(b *testing.B) {
	b.StopTimer()
	address := "localhost:9090"
	message := "hello world"
	clientHandler := NewEchoClientHandler()
	sCtx := NewEchoServerContext(message, false, 1)
	ListenAndServe(address, sCtx)

	connection, err := Connect(address)
	if err != nil {
		return
	}

	defer connection.Close()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		rawResponse, _, err := SendAndReceive(connection, clientHandler, ([]byte)(message))
		if string(rawResponse) != message || err != nil {
			fmt.Printf("failed in eccho \"hello world\": err: %v; received %q, expected %q\n", err, string(rawResponse), message)
		}
	}
}

