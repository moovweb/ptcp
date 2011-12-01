package ptcp

import (
	"testing"
	"os"
	"log"
)

const requestBufferSize = 1000

type EchoServerHandler struct {
	Buffer []byte
}


func NewEchoServerHandler(_ interface {}) ServerHandler {
	esh := &EchoServerHandler{}
	esh.Buffer = make([]byte, requestBufferSize)
	return esh
}

func (esh *EchoServerHandler) Cleanup() {
}

func (esh *EchoServerHandler) Handle (connection *TcpConnection) (err os.Error) {
	n, err := connection.Read(esh.Buffer)
	if err != nil {
		if err == os.EOF {
			err = nil
		}
		return
	}
	if n <= 0 {
		return os.NewError("should receive more than 0 bytes")
	}
	request := esh.Buffer[0:n]
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

func (ech *EchoClientHandler) Handle (connection *TcpConnection, request interface {}) (response []byte, err os.Error) {
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
	clientHandler := NewEchoClientHandler()
	ListenAndServe(address, nil, NewEchoServerHandler, 2, false, false)
	message := "hello world"
	for i := 0; i < 10; i++ {
		connection, err := Connect(address)
		if err != nil {
			t.Error("error when connecting to %s: $v\n", address, err) 
		}
		go func() {
			defer connection.Close()
			response, err := SendAndReceive(connection, clientHandler, true, ([]byte)(message))
			if string(response) != message || err != nil {
				t.Errorf("failed in eccho \"hello world\": err: %v; received %q, expected %q\n", err, string(response), message)
			}
		}()
	}
}

func BenchmarkEcho(b *testing.B) {
	b.StopTimer()
	address := "localhost:9090"
	clientHandler := NewEchoClientHandler()

	ListenAndServe(address, nil, NewEchoServerHandler, 2, false, false)
	message := "hello world"
	connection, err := Connect(address)
	if err != nil {
		return
	}
	defer connection.Close()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		response, err := SendAndReceive(connection, clientHandler, true, ([]byte)(message))
		if string(response) != message || err != nil {
			log.Printf("failed in eccho \"hello world\": err: %v; received %q, expected %q\n", err, string(response), message)
		}
	}
}

