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

const (
	DefaultErrorResponse = "HTTP/1.1 500\r\nConnection: close\r\nContent-Type: text/html;\r\nContent-Length: 21\r\n\r\nInternal Server Error"
	DefaultOKResponse    = "HTTP/1.0 200 OK\r\nConnection: close\r\nContent-Type: text/plain;\r\nContent-Length: 2\r\n\r\nOK"
)

var HttpHeaderBodySep = []byte("\r\n\r\n")
var ContentEncodingKey = http.CanonicalHeaderKey("content-encoding")
var ErrorHttpServerShouldSaveReadData = os.NewError("Server context should set SaveReadData to true")

var statusText = map[int]string{
	http.StatusContinue:           "Continue",
	http.StatusSwitchingProtocols: "Switching Protocols",

	http.StatusOK:                   "OK",
	http.StatusCreated:              "Created",
	http.StatusAccepted:             "Accepted",
	http.StatusNonAuthoritativeInfo: "Non-Authoritative Information",
	http.StatusNoContent:            "No Content",
	http.StatusResetContent:         "Reset Content",
	http.StatusPartialContent:       "Partial Content",

	http.StatusMultipleChoices:   "Multiple Choices",
	http.StatusMovedPermanently:  "Moved Permanently",
	http.StatusFound:             "Found",
	http.StatusSeeOther:          "See Other",
	http.StatusNotModified:       "Not Modified",
	http.StatusUseProxy:          "Use Proxy",
	http.StatusTemporaryRedirect: "Temporary Redirect",

	http.StatusBadRequest:                   "Bad Request",
	http.StatusUnauthorized:                 "Unauthorized",
	http.StatusPaymentRequired:              "Payment Required",
	http.StatusForbidden:                    "Forbidden",
	http.StatusNotFound:                     "Not Found",
	http.StatusMethodNotAllowed:             "Method Not Allowed",
	http.StatusNotAcceptable:                "Not Acceptable",
	http.StatusProxyAuthRequired:            "Proxy Authentication Required",
	http.StatusRequestTimeout:               "Request Timeout",
	http.StatusConflict:                     "Conflict",
	http.StatusGone:                         "Gone",
	http.StatusLengthRequired:               "Length Required",
	http.StatusPreconditionFailed:           "Precondition Failed",
	http.StatusRequestEntityTooLarge:        "Request Entity Too Large",
	http.StatusRequestURITooLong:            "Request URI Too Long",
	http.StatusUnsupportedMediaType:         "Unsupported Media Type",
	http.StatusRequestedRangeNotSatisfiable: "Requested Range Not Satisfiable",
	http.StatusExpectationFailed:            "Expectation Failed",

	http.StatusInternalServerError:     "Internal Server Error",
	http.StatusNotImplemented:          "Not Implemented",
	http.StatusBadGateway:              "Bad Gateway",
	http.StatusServiceUnavailable:      "Service Unavailable",
	http.StatusGatewayTimeout:          "Gateway Timeout",
	http.StatusHTTPVersionNotSupported: "HTTP Version Not Supported",
}

type HttpServerHandler struct {
	NumHandlers int
	count       int
	id          uint32
	logConfig   *log4go.LogConfig
	logger      log4go.Logger
	tag         string
}

func NewHttpServerHandler(logConfig *log4go.LogConfig, numHandlers int, tag string) *HttpServerHandler {
	return &HttpServerHandler{logConfig: logConfig, NumHandlers: numHandlers, tag: tag}
}

func (h *HttpServerHandler) Spawn() (ServerHandler, os.Error) {
	if h.count < h.NumHandlers {
		h.count++
		handler := &HttpServerHandler{}
		handler.id = uint32(h.count)
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

func (h *HttpServerHandler) Handle(connection *TcpConnection) (err os.Error) {
	var response []byte
	httpRequest, request, err := h.ReceiveDownstreamRequest(connection)
	if err != nil {
		if err != io.ErrUnexpectedEOF {
			h.logger.Error("ReceiveDownstreamRequest error: %v", err)
		} else {
			err = os.EOF //client has closed the connection?
		}
		return
	}

	h.logger.Debug("Received downstream request:\n\n%v\n", string(request))
	//logger.Info("Request URL: %s", httpRequest.RawURL)
	if httpRequest.RawURL == "/moov_check" {
		h.logger.Debug("moov_check")
		response = []byte(DefaultOKResponse)
		connection.Write([]byte(response))
		err = ErrorServerCloseConnection
		h.logger.Debug(err.String())
		return
	} else if httpRequest.RawURL == "/moov_fail" {
		h.logger.Error("moov_fail test")
		response = []byte(DefaultErrorResponse)
		connection.Write([]byte(response))
		err = ErrorServerCloseConnection
		h.logger.Debug(err.String())
		return
	}

	_, err = connection.Write([]byte(response))

	if err == nil && !WantsHttp10KeepAlive(httpRequest) {
		err = ErrorClientCloseConnection
	}

	h.logger.Debug("Wrote downstream response:\n\n%v\n", string(response))
	return
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
	return 0
}

func (h *HttpServerHandler) Cleanup() {
	return
}

func (h *HttpServerHandler) ReceiveDownstreamRequest(connection *TcpConnection) (httpRequest *http.Request, request []byte, err os.Error) {
	br := bufio.NewReader(connection)
	httpRequest, err = http.ReadRequest(br)
	if err != nil {
		return
	}
	_, err = ioutil.ReadAll(httpRequest.Body)
	if err != nil {
		recvd := connection.RawData()
		h.logger.Notice("Failed to receive a valid HTTP request body: %v (length: %d)", string(recvd), len(recvd))
		return
	}

	request = connection.RawData()
	if request == nil {
		err = ErrorHttpServerShouldSaveReadData
	}
	return
}

func WantsHttp10KeepAlive(request *http.Request) bool {
	if request.ProtoMajor != 1 || request.ProtoMinor != 0 {
		return false
	}
	return strings.Contains(strings.ToLower(request.Header.Get("Connection")), "keep-alive")
}
