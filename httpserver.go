package ptcp

import (
	"os"
	"http"
	"log4go"
	"fmt"
	"io"
	"io/ioutil"
	"bufio"
	"strings"
)

type HttpServerHandler struct {
	NumHandlers int
	Count       int
	id          uint32
	logConfig   *log4go.LogConfig
	logger      log4go.Logger
	tag         string
}

const DefaultConnectionQueueLength = 128

func NewHttpServerHandler(logConfig *log4go.LogConfig, numHandlers int, tag string) *HttpServerHandler {
	return &HttpServerHandler{logConfig: logConfig, NumHandlers: numHandlers, tag: tag}
}

func (h *HttpServerHandler) Spawn() (interface{}, os.Error) {
	if h.Count < h.NumHandlers {
		h.Count++
		handler := &HttpServerHandler{}
		handler.id = uint32(h.Count)
		handler.logConfig = h.logConfig
		handler.tag = h.tag
		return handler, nil
	}
	return nil, ErrHandlerLimitReached
}

func (h *HttpServerHandler) Logger() log4go.Logger {
	if h.logger == nil {
		logPrefix := fmt.Sprintf("%s", h.Tag())
		h.logger = log4go.NewLoggerFromConfig(h.logConfig, logPrefix)
	}
	return h.logger
}

func (h *HttpServerHandler) Tag() (tag string) {
	if h.id == 0 {
		tag = fmt.Sprintf("%s", h.tag)
	} else {
		tag = fmt.Sprintf("%s (%d)", h.tag, h.id)
	}
	return
}

func (h *HttpServerHandler) ConnectionQueueLength() int {
	return DefaultConnectionQueueLength
}

func (h *HttpServerHandler) Cleanup() {
	return
}

func (h *HttpServerHandler) Handle(connection *TcpConnection) (err os.Error) {
	uHttpRequest, err := h.ReceiveRequest(connection)
	if err != nil {
		if err != io.ErrUnexpectedEOF {
			h.logger.Error("ReceiveDownstreamRequest error: %v", err)
		} else {
			err = os.EOF //client has closed the connection?
		}
		return
	}

	closeAfterReply := false

	_, err = connection.Write([]byte(DefaultOKResponse))

	if err == nil && !WantsConnectionAlive(uHttpRequest.HttpRequest) {
		closeAfterReply = true
	}

	h.logger.Debug("Wrote downstream response:\n\n%v\n", string(DefaultOKResponse))

	if closeAfterReply {
		err = ErrorClientCloseConnection
	}
	return
}

func (h *HttpServerHandler) ReceiveRequest(connection *TcpConnection) (uHttpRequest *UpstreamHttpRequest, err os.Error) {
	br := bufio.NewReader(connection)
	httpRequest, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	_, err = ioutil.ReadAll(httpRequest.Body)
	if err != nil {
		recvd := connection.RawData()
		h.logger.Notice("Failed to receive a valid HTTP request body: %v (length: %d)", string(recvd), len(recvd))
		return
	}
	rawRequest := connection.RawData()
	if rawRequest == nil {
		err = ErrorHttpServerShouldSaveReadData
	}
	uHttpRequest = &UpstreamHttpRequest{HttpRequest: httpRequest, Request: rawRequest}
	return
}

func WantsConnectionAlive(request *http.Request) bool {
	return request.ProtoAtLeast(1, 1) && strings.Contains(strings.ToLower(request.Header.Get("Connection")), "keep-alive")
}
