package ptcp

import (
	"os"
	"http"
	"log4go"
	"fmt"
)

const (
	DefaultErrorResponse = "HTTP/1.1 500\r\nConnection: close\r\nContent-Type: text/html;\r\nContent-Length: 21\r\n\r\nInternal Server Error"
	DefaultOKResponse    = "HTTP/1.0 200 OK\r\nConnection: close\r\nContent-Type: text/plain;\r\nContent-Length: 2\r\n\r\nOK"
)

var HttpHeaderBodySep = []byte("\r\n\r\n")
var ContentEncodingKey = http.CanonicalHeaderKey("content-encoding")

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
