package test

import (
	"testing"
	"os"
	"log"
	"syslog"
	"ptcp"
)

const requestBufferSize = 1000

type EchoServerContext struct {
	*ptcp.BasicServerContext
	message string
}

type EchoServerHandlerContext struct {
	*ptcp.BasicServerHandlerContext
	Buffer []byte
}

func NewEchoServerContext(message string, blocking bool, numHandlers int) ptcp.ServerContext {
	esCtx := &EchoServerContext{message:message}
	esCtx.BasicServerContext = ptcp.NewBasicServerContext((int)(syslog.LOG_DEBUG), numHandlers, blocking, "EchoServer")
	return esCtx
}

func (esCtx *EchoServerContext) NewServerHandlerContext(id uint32) ptcp.ServerHandlerContext {
	bsCtx := esCtx.BasicServerContext
	shCtx := bsCtx.NewServerHandlerContext(id)
	eshCtx := &EchoServerHandlerContext{}
	eshCtx.BasicServerHandlerContext = shCtx.(*ptcp.BasicServerHandlerContext)
	eshCtx.Buffer = make([]byte, requestBufferSize)
	//import!!! we need to set the server context to the caller object; otherwise it is pointed to eshCtx.BasicServerContext
	eshCtx.SetServerContext(esCtx)
	return eshCtx
}

func (esCtx *EchoServerContext) GetShared() (shared interface{}) {
	return esCtx.message
}

func (eshCtx *EchoServerHandlerContext) Handle (connection *ptcp.TcpConnection) (err os.Error) {
	n, err := connection.Read(eshCtx.Buffer)
	if err != nil {
		return
	}
	if n <= 0 {
		return os.NewError("should receive more than 0 bytes")
	}
	request := eshCtx.Buffer[0:n]
	message := eshCtx.GetServerContext().GetShared().(string)
	if (message != string(request)) {
		eshCtx.GetLogger().Crit("Wrong Message\n")
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

func (ech *EchoClientHandler) Handle (connection *ptcp.TcpConnection, request interface {}) (response []byte, err os.Error) {
	requestBytes, ok := request.([]byte)
	if ! ok {
		return nil, os.NewError("EchoClientHandler cannot convert request into bytes")
	}
	_, err = connection.Write(requestBytes)
	if err != nil {
		log.Printf("err : %v\n", err)
		return nil, err
	}
	n, err := connection.Read(ech.Buffer)
	if err == os.EOF {
		err = nil
	}
	if err != nil {
		return nil, err
	}
	return ech.Buffer[:n], nil
}

func TestEcho(t *testing.T) {
	address := "localhost:9090"
	message := "hello world"
	clientHandler := NewEchoClientHandler()
	sCtx := NewEchoServerContext(message, false, 10)
	ptcp.ListenAndServe(address, sCtx)
	for i := 0; i < 10; i++ {
		connection, err := ptcp.Connect(address)
		connection.EnableSaveReadData()
		if err != nil {
			t.Error("error when connecting to %s: $v\n", address, err) 
		}
		go func() {
			defer connection.Close()
			response, err := ptcp.SendAndReceive(connection, clientHandler, ([]byte)(message))
			if string(response) != message || err != nil {
				t.Errorf("failed in eccho \"hello world\": err: %v; received %q, expected %q\n", err, string(response), message)
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
	ptcp.ListenAndServe(address, sCtx)

	connection, err := ptcp.Connect(address)
	if err != nil {
		return
	}

	defer connection.Close()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		response, err := ptcp.SendAndReceive(connection, clientHandler, ([]byte)(message))
		if string(response) != message || err != nil {
			log.Printf("failed in eccho \"hello world\": err: %v; received %q, expected %q\n", err, string(response), message)
		}
	}
}

