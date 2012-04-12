package ptcp

import "testing"
import "sync"
import "log4go"
import "fmt"
import "log"
import "bufio"
import "bytes"
import "net/http"

//use a different port from the echo test because linux does not like the two tests using the same port
const TestAddr2 = "localhost:13253"

func TestHttpClientServer(t *testing.T) {
	address := TestAddr2
	wg := &sync.WaitGroup{}
	clientHandler := &HttpClientHandler{}
	logConfig := &log4go.LogConfig{ConsoleLogLevel: int(log4go.INFO)}
	serverHandler := NewHttpServerHandler(logConfig, 4, "test_http_srv")
	ListenAndServe(address, serverHandler, false)
	uHttpRequest := &UpstreamHttpRequest{}
	uHttpRequest.Request = ([]byte)("GET / HTTP/1.1\r\n\r\n")

	buffer := bytes.NewBuffer(uHttpRequest.Request)
	bf := bufio.NewReader(buffer)
	httpRequest, err := http.ReadRequest(bf)
	if err != nil {
		t.Errorf("err: %v\n", err)
		return
	}
	uHttpRequest.HttpRequest = httpRequest

	for i := 0; i < 10; i++ {
		connection, err := Connect(address)
		connection.EnableSaveReadData()
		if err != nil {
			t.Error("error when connecting to %s: $v\n", address, err)
		}
		wg.Add(1)
		go func() {
			defer connection.Close()
			response, err := SendAndReceive(connection, clientHandler, uHttpRequest)
			if string(response.Bytes()) != DefaultOKResponse || err != nil {
				t.Errorf("err: %v; received %q, expected %q\n", err, string(response.Bytes()), DefaultOKResponse)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkHttpClientServer(b *testing.B) {
	b.StopTimer()
	address := TestAddr2
	clientHandler := &HttpClientHandler{}
	logConfig := &log4go.LogConfig{ConsoleLogLevel: int(log4go.INFO)}
	serverHandler := NewHttpServerHandler(logConfig, 1, "test_http_srv")
	ListenAndServe(address, serverHandler, false)
	uHttpRequest := &UpstreamHttpRequest{}
	uHttpRequest.Request = ([]byte)("GET / HTTP/1.1\r\nConnection: keep-alive\r\n\r\n")
	buffer := bytes.NewBuffer(uHttpRequest.Request)
	bf := bufio.NewReader(buffer)
	httpRequest, err := http.ReadRequest(bf)
	if err != nil {
		log.Fatalf("err: %v\n", err)
	}
	uHttpRequest.HttpRequest = httpRequest

	connection, err := Connect(address)
	connection.EnableSaveReadData()
	if err != nil {
		fmt.Errorf("error when connecting to %s: $v\n", address, err)
		return
	}
	defer connection.Close()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := SendAndReceive(connection, clientHandler, uHttpRequest)
		if err != nil {
			log.Fatalf("err: %s", err.Error())
		}
	}
}
