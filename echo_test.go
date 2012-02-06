package ptcp

import (
	"testing"
	"os"
	"log4go"
	"fmt"
	"sync"
)

const requestBufferSize = 1000

type EchoServerContext struct {
	*DefaultServerContext
	message string
}

type EchoServerHandlerContext struct {
	*DefaultServerHandlerContext
	Buffer []byte
}

func NewEchoServerContext(message string, blocking bool, numHandlers int) ServerContext {
	echoServerCtx := &EchoServerContext{message:message}
	logConfig := &log4go.LogConfig{ConsoleLogLevel: int(log4go.DEBUG), SysLogLevel: int(log4go.DEBUG)}
	echoServerCtx.DefaultServerContext = NewDefaultServerContext(logConfig, numHandlers, blocking, "EchoServer")
	echoServerCtx.ServerHandlerContextConstructor = NewEchoServerHandlerContext
	echoServerCtx.SaveReadData = true
	return echoServerCtx
}

func NewEchoServerHandlerContext(id uint32, serverCtx ServerContext) ServerHandlerContext {
	echoServerHandlerCtx := &EchoServerHandlerContext{}
	echoServerHandlerCtx.Buffer = make([]byte, requestBufferSize)
	echoServerHandlerCtx.DefaultServerHandlerContext = NewDefaultServerHandlerContext(id, serverCtx)
	return echoServerHandlerCtx
}

func (echoServerHandlerCtx *EchoServerHandlerContext) GetLogger() log4go.Logger {
	if echoServerHandlerCtx.Logger == nil {
		logPrefix := fmt.Sprintf("%v (%d) Mikey", echoServerHandlerCtx.GetServerTag(), echoServerHandlerCtx.GetId())
		logConfig := echoServerHandlerCtx.GetLogConfig()
		echoServerHandlerCtx.Logger = log4go.NewLoggerFromConfig(logConfig, logPrefix)
	}
	return echoServerHandlerCtx.Logger
}

func (echoServerHandlerCtx *EchoServerHandlerContext) Handle (connection *TcpConnection) (err os.Error) {
	n, err := connection.Read(echoServerHandlerCtx.Buffer)
	if err != nil {
		return
	}
	if n <= 0 {
		return os.NewError("should receive more than 0 bytes")
	}
	request := echoServerHandlerCtx.Buffer[0:n]
	message := echoServerHandlerCtx.GetServerContext().(*EchoServerContext).message
	logger := echoServerHandlerCtx.GetLogger()
	if (message != string(request)) {
		logger.Error("Wrong Message")
	} else {
		logger.Info("message: %v", message)
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
	wg := &sync.WaitGroup{}
	clientHandler := NewEchoClientHandler()
	sCtx := NewEchoServerContext(message, false, 10)
	ListenAndServe(address, sCtx)
	for i := 0; i < 10; i++ {
		connection, err := Connect(address)
		if err != nil {
			t.Error("error when connecting to %s: $v\n", address, err) 
		}
		wg.Add(1)
		go func() {
			defer connection.Close()
			rawResponse, _, err := SendAndReceive(connection, clientHandler, ([]byte)(message))
			if string(rawResponse) != message || err != nil {
				t.Errorf("failed in eccho \"hello world\": err: %v; received %q, expected %q\n", err, string(rawResponse), message)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestEchoSSL(t *testing.T) {
	address := "localhost:9092"
	message := "hello world"
	wg := &sync.WaitGroup{}
	clientHandler := NewEchoClientHandler()
	sCtx := NewEchoServerContext(message, false, 10)
	ListenAndServeTLS(address, sCtx, "./keys/server.crt", "./keys/server.key")
	for i := 0; i < 10; i++ {
		connection, err := ConnectTLS(address, "localhost", false)
		if err != nil {
			t.Error("error when connecting to %s: $v\n", address, err) 
		}
		wg.Add(1)
		go func() {
			defer connection.Close()
			rawResponse, _, err := SendAndReceive(connection, clientHandler, ([]byte)(message))
			if string(rawResponse) != message || err != nil {
				t.Errorf("failed in eccho \"hello world\": err: %v; received %q, expected %q\n", err, string(rawResponse), message)
			}
			wg.Done()
		}()
	}
	wg.Wait()
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

