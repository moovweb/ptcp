package ptcp

import (
	"testing"
	"os"
	"log4go"
	"log"
	"fmt"
	"sync"
)

const requestBufferSize = 1000

const DefaultReuqest = "Hello"
const DefaultResponse = "World"
const HandlerLimit = 8

type DataStream string

func (d *DataStream) Bytes() []byte {
	return []byte(*d)
}

var (
	DefaultReuqestBytes  = []byte(DefaultReuqest)
	DefaultResponseBytes = []byte(DefaultResponse)
)

type EchoServerHandler struct {
	count  int
	id     uint32
	logger log4go.Logger
	buffer []byte
}

func (h *EchoServerHandler) Spawn() (newH interface{}, err os.Error) {
	if h.count < HandlerLimit {
		h.count++
		handler := &EchoServerHandler{}
		handler.id = uint32(h.count)
		handler.buffer = make([]byte, 100)
		newH = handler
	} else {
		err = ErrHandlerLimitReached
	}
	return
}

func (h *EchoServerHandler) Logger() log4go.Logger {
	if h.logger == nil {
		logPrefix := fmt.Sprintf("%s Mikey", h.Tag())
		logConfig := &log4go.LogConfig{ConsoleLogLevel: int(log4go.ERROR)}
		h.logger = log4go.NewLoggerFromConfig(logConfig, logPrefix)
	}
	return h.logger
}

func (h *EchoServerHandler) Handle(connection *TcpConnection) (err os.Error) {
	_, err = connection.Read(h.buffer)
	if err != nil {
		return
	}

	_, err = connection.Write(DefaultResponseBytes)
	return err
}

func (h *EchoServerHandler) Tag() (tag string) {
	if h.id == 0 {
		tag = fmt.Sprintf("Echo server")
	} else {
		tag = fmt.Sprintf("Echo server (%d)", h.id)
	}
	return
}

func (h *EchoServerHandler) ConnectionQueueLength() (length int) {
	length = 4
	return
}

func (h *EchoServerHandler) Cleanup() {
	return
}

type EchoClientHandler struct {
	Buffer []byte
}

func NewEchoClientHandler() *EchoClientHandler {
	ech := &EchoClientHandler{}
	ech.Buffer = make([]byte, requestBufferSize)
	return ech
}

func (ech *EchoClientHandler) Handle(connection *TcpConnection, request Request) (response Response, err os.Error) {
	_, err = connection.Write(DefaultReuqestBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "err : %v\n", err)
		return nil, err
	}
	n, err := connection.Read(ech.Buffer)
	if err == os.EOF {
		err = nil
	}
	if err != nil {
		return nil, err
	}
	data := DataStream(ech.Buffer[:n])
	return &data, nil
}

func TestEcho(t *testing.T) {
	address := "localhost:9090"
	wg := &sync.WaitGroup{}
	clientHandler := NewEchoClientHandler()
	serverHandler := &EchoServerHandler{}
	ListenAndServe(address, serverHandler, false)
	for i := 0; i < 10; i++ {
		connection, err := Connect(address)
		if err != nil {
			t.Error("error when connecting to %s: $v\n", address, err)
		}
		wg.Add(1)
		go func() {
			defer connection.Close()
			data := DataStream("")
			response, err := SendAndReceive(connection, clientHandler, &data)
			if string(response.Bytes()) != DefaultResponse || err != nil {
				t.Errorf("failed in eccho \"hello world\": err: %v; received %q, expected %q\n", err, string(response.Bytes()), DefaultResponse)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

/*
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
*/
func BenchmarkEcho(b *testing.B) {
	b.StopTimer()
	address := "localhost:9090"
	clientHandler := NewEchoClientHandler()
	serverHandler := &EchoServerHandler{}
	ListenAndServe(address, serverHandler, false)

	connection, err := Connect(address)
	if err != nil {
		return
	}

	defer connection.Close()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		data := DataStream("")
		response, err := SendAndReceive(connection, clientHandler, &data)
		if string(response.Bytes()) != DefaultResponse || err != nil {
			log.Fatalf("failed in eccho \"hello world\": err: %v; received %q, expected %q\n", err, string(response.Bytes()), DefaultResponse)
		}
	}
}
