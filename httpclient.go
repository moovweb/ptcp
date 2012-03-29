package ptcp

import (
	"os"
	"http"
	"bufio"
	"io/ioutil"
	"bytes"
	"strings"
	"compress/gzip"
	"compress/flate"
	"strconv"
	"io"
)

type HttpClientHandler struct {

}

type UpstreamHttpRequest struct {
	Request    []byte
	HttpMethod string
}

type HttpResponse struct {
	Header http.Header
	Body   []byte
}

func (hch *HttpClientHandler) Handle(connection *TcpConnection, request interface{}) (rawResponse []byte, response interface{}, err os.Error) {
	upstreamReq, ok := request.(*UpstreamHttpRequest)
	if !ok {
		err = os.NewError("httpClientHandler cannot convert request to UpstreamHttpRequest")
		return
	}
	_, err = connection.Write(upstreamReq.Request)
	if err != nil {
		return
	}

	br := bufio.NewReader(connection)
	httpResponse, err := http.ReadResponse(br, &http.Request{Method: upstreamReq.HttpMethod})
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(httpResponse.Body)
	if err != nil {
		return
	}
	rawResponse = connection.RawData()

	//now we have finished reading the response
	//The response is saved in two places (essentially duplicated)
	//1. is the parsed response: httpResponse
	//2. is the raw response: rawResponse
	//we choose to use the raw response because we want minimum changes to the response string
	//However, we may still have to use the parsed response if it comes back in chunked encoding
	//the http pkg deals with the chunks, and the parsed http response is the assembled, complete response

	hResponse := &HttpResponse{}
	hResponse.Header = httpResponse.Header

	//separate the raw response into header and body
	rawResponse = connection.RawData()
	endOfHeader := bytes.Index(rawResponse, HttpHeaderBodySep)
	if endOfHeader < 0 {
		err = os.NewError("cannot find the end of unmodified response header in http response")
		return
	}
	endOfHeader += 4
	RawHeader := rawResponse[:endOfHeader]
	hResponse.Body = rawResponse[endOfHeader:]

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
		io.WriteString(w, "\r\n")
		RawHeader = w.Bytes()
		hResponse.Body = body
	}

	contentEncodings := hResponse.Header[ContentEncodingKey]

	if len(contentEncodings) > 0 && strings.ToLower(contentEncodings[0]) == "deflate" {
		reasponseAsReader := bytes.NewBuffer(hResponse.Body)
		decompressor := flate.NewReader(reasponseAsReader)
		unzipped, err := ioutil.ReadAll(decompressor)
		if err == nil {
			hResponse.Body = unzipped
		}
		decompressor.Close()
	}

	if len(contentEncodings) > 0 && strings.ToLower(contentEncodings[0]) == "gzip" {
		reasponseAsReader := bytes.NewBuffer(hResponse.Body)
		decompressor, err := gzip.NewReader(reasponseAsReader)
		if err == nil {
			unzipped, err := ioutil.ReadAll(decompressor)
			if err == nil {
				hResponse.Body = unzipped
			}
			decompressor.Close()
		}
	}
	rawResponse = append(RawHeader, hResponse.Body...)
	response = hResponse
	return
}