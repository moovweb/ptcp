package ptcp

import (
	"errors"
	"net/http"
)

import "bytes"

const (
	DefaultErrorResponse = "HTTP/1.1 500\r\nConnection: close\r\nContent-Type: text/html;\r\nContent-Length: 21\r\n\r\nInternal Server Error"
	DefaultOKResponse    = "HTTP/1.1 200 OK\r\nConnection: close\r\nContent-Type: text/plain;\r\nContent-Length: 2\r\n\r\nOK"
)

var HttpHeaderBodySepSig = []byte("\r\n\r\n")
var ContentEncodingKey = http.CanonicalHeaderKey("content-encoding")
var ErrorHttpServerShouldSaveReadData = errors.New("Server context should set SaveReadData to true")
var ErrorIncompleteRequest = errors.New("Incomplete Http Request")
var ErrorIncompleteResponse = errors.New("Incomplete Http Response")

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

type UpstreamHttpRequest struct {
	HttpRequest *http.Request
	Ssl         bool
	Request     []byte
}

func (req *UpstreamHttpRequest) Bytes() []byte {
	return req.Request
}

type UpstreamHttpResponse struct {
	Header       http.Header
	RawHeader    []byte
	Body         []byte
	HttpResponse *http.Response
}

func (resp *UpstreamHttpResponse) Bytes() []byte {
	data := append(resp.RawHeader, HttpHeaderBodySepSig...)
	data = append(data, resp.Body...)
	return data
}

func SeparateHttpHeaderBody(raw []byte) (header, body []byte, err error) {
	endOfHeader := bytes.Index(raw, HttpHeaderBodySepSig)
	if endOfHeader < 0 {
		err = ErrorIncompleteResponse
		return
	}
	endOfHeader += 4
	header = raw[:endOfHeader-4]
	body = raw[endOfHeader:]
	return
}
