package ptcp

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

var ErrInvalidRequestType = errors.New("expect request to be of UpstreamHttpRequest")

type HttpClientHandler struct {
}

func (hch *HttpClientHandler) Handle(connection *TcpConnection, request Request) (response Response, err error) {
	upstreamReq, ok := request.(*UpstreamHttpRequest)
	if !ok {
		err = ErrInvalidRequestType
		return
	}

	_, err = connection.Write(upstreamReq.Request)
	if err != nil {
		return
	}

	br := bufio.NewReader(connection)
	httpResponse, err := http.ReadResponse(br, &http.Request{Method: upstreamReq.HttpRequest.Method})
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return
	}

	rawResponse := connection.RawData()

	//now we have finished reading the response
	//The response is saved in two places (essentially duplicated)
	//1. is the parsed response: httpResponse
	//2. is the raw response: rawResponse
	//we choose to use the raw response because we want minimum changes to the response string
	//However, we may still have to use the parsed response if it comes back in chunked encoding
	//the http pkg deals with the chunks, and the parsed http response is the assembled, complete response

	uResponse := &UpstreamHttpResponse{}
	uResponse.Header = httpResponse.Header

	//separate the raw response into header and body
	rawResponse = connection.RawData()
	RawHeader, RawBody, err := SeparateHttpHeaderBody(rawResponse)
	if err != nil {
		println("err here:", err.Error())
		return
	}

	uResponse.Body = RawBody

	//detect if the raw response contains chunked encoding
	//should use a regex
	if bytes.Index(RawHeader, []byte("Transfer-Encoding:")) >= 0 && bytes.Index(RawHeader, []byte("chunked")) >= 0 {
		//should use the TcpConnection's buffer
		w := bytes.NewBuffer(nil)
		if httpResponse.Request != nil {
			httpResponse.Request.Method = strings.ToUpper(httpResponse.Request.Method)
		}
		text := httpResponse.Status
		if text == "" {
			var ok bool
			text, ok = statusText[httpResponse.StatusCode]
			if !ok {
				text = "status code " + strconv.Itoa(httpResponse.StatusCode)
			}
		}
		io.WriteString(w, "HTTP/"+strconv.Itoa(httpResponse.ProtoMajor)+".")
		io.WriteString(w, strconv.Itoa(httpResponse.ProtoMinor)+" ")
		io.WriteString(w, text+"\r\n")
		httpResponse.Header.Write(w)
		//end of header
		//io.WriteString(w, "\r\n")
		RawHeader = w.Bytes()
		if headerLen := len(RawHeader); headerLen > 2 {
			if string(RawHeader[headerLen-2:]) == "\r\n" {
				RawHeader = RawHeader[:headerLen-2]
			}
		}
		uResponse.Body = body
	}

	contentEncodings := uResponse.Header[ContentEncodingKey]

	if len(contentEncodings) > 0 {
		enc := strings.ToLower(contentEncodings[0])
		if enc == "deflate" {
			reasponseAsReader := bytes.NewBuffer(uResponse.Body)
			decompressor := flate.NewReader(reasponseAsReader)
			unzipped, err := ioutil.ReadAll(decompressor)
			if err == nil {
				uResponse.Body = unzipped
			}
			decompressor.Close()
		} else if enc == "gzip" {
			reasponseAsReader := bytes.NewBuffer(uResponse.Body)
			decompressor, err := gzip.NewReader(reasponseAsReader)
			if err == nil {
				unzipped, err := ioutil.ReadAll(decompressor)
				if err == nil {
					uResponse.Body = unzipped
				}
				decompressor.Close()
			}
		}
		if err != nil {
			return
		}
	}
	uResponse.RawHeader = RawHeader
	response = uResponse
	return
}
