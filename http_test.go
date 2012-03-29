package ptcp

import "testing"
import "sync"
import "log4go"
import "fmt"

func TestHttpClientServer(t *testing.T) {
	address := "localhost:9090"
	wg := &sync.WaitGroup{}
	clientHandler := &HttpClientHandler{}
	logConfig := &log4go.LogConfig{ConsoleLogLevel: int(log4go.INFO)}
	serverHandler := NewHttpServerHandler(logConfig, 4, "test_http_srv")
	ListenAndServe(address, serverHandler, false)
	upstreamHttpRequest := &UpstreamHttpRequest{}
	upstreamHttpRequest.Request = ([]byte)("GET / HTTP/1.1\r\n\r\n")
	upstreamHttpRequest.HttpMethod = "GET"
	for i := 0; i < 10; i++ {
		connection, err := Connect(address)
		connection.EnableSaveReadData()
		if err != nil {
			t.Error("error when connecting to %s: $v\n", address, err)
		}
		wg.Add(1)
		go func() {
			defer connection.Close()
			rawResponse, _, err := SendAndReceive(connection, clientHandler, upstreamHttpRequest)
			if string(rawResponse) != DefaultOKResponse || err != nil {
				t.Errorf("err: %v; received %q, expected %q\n", err, string(rawResponse), DefaultOKResponse)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkHttpClientServer(b *testing.B) {
	b.StopTimer()
	address := "localhost:9090"
	clientHandler := &HttpClientHandler{}
	logConfig := &log4go.LogConfig{ConsoleLogLevel: int(log4go.INFO)}
	serverHandler := NewHttpServerHandler(logConfig, 4, "test_http_srv")
	ListenAndServe(address, serverHandler, false)
	upstreamHttpRequest := &UpstreamHttpRequest{}
	upstreamHttpRequest.Request = ([]byte)("GET / HTTP/1.1\r\n\r\n")
	upstreamHttpRequest.HttpMethod = "GET"
	connection, err := Connect(address)
	connection.EnableSaveReadData()
	if err != nil {
		fmt.Errorf("error when connecting to %s: $v\n", address, err)
		return
	}
	defer connection.Close()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		rawResponse, _, err := SendAndReceive(connection, clientHandler, upstreamHttpRequest)
		if string(rawResponse) != DefaultOKResponse || err != nil {
			fmt.Errorf("err: %v; received %q, expected %q\n", err, string(rawResponse), DefaultOKResponse)
		}
	}
}