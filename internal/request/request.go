package request

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"unicode"

	"httpffomtcp.pinglu.dev/internal/headers"
)

var ERROR_MALFORMED_REQUEST_LINE = errors.New("malformed request line")
var ERROR_INVALID_METHOD = errors.New("invalid method")
var ERROR_UNSUPPORTED_HTTP_VERSION = errors.New("unsupported http version")
var ERROR_MISSING_HOST_HEADER = errors.New("missing host header")
var ERROR_CONTENT_LENGTH_EXCEEDED = errors.New("content length exceeded")
var CRLF = []byte("\r\n")

const BUFFER_SIZE = 8

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

type parserState string

const (
	INITIALIZED     parserState = "initialized"
	PARSING_HEADERS parserState = "parsing headers"
	PARSING_BODY    parserState = "parsing body"
	DONE            parserState = "done"
)

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	Body        []byte
	parserState parserState
}

func newRequest() *Request {
	return &Request{
		Headers:     headers.NewHeaders(),
		parserState: INITIALIZED,
	}
}

func (r *Request) parse(data []byte) (int, error) {
	totalBytesParsed := 0
	startIndex := 0

	// We need a for loop here, because if there is no more data,
	// net.Conn.Read() will just hang, waiting for more data to come,
	// so when there is no request body, request.parse() will never get
	// a chance to run to set r.parserState to DONE, and as a result, we
	// will be stuck in an infinite loop. For instance, the last chunk of
	// data net.Conn.Read() reads is "Accept: */*\r\n\r\n". If we don't
	// have the outer for loop, after the for loop for parsing the headers
	// terminates, r.parserState is set to PARSING_BODY and the control is
	// given back to the for loop in RequestFromReader(). Since request.done()
	// returns false, the for loop continues and net.Conn.Read() gets called
	// again. Because there is no more data to read, net.Conn.Read() just hangs,
	// waiting for more data to come, and request.parse() never gets called.
outer:
	for {
		switch r.parserState {
		case INITIALIZED:
			rl, n, err := parseRequestLine(data[startIndex:])
			if err != nil {
				return 0, err
			}

			if n == 0 {
				break outer
			}

			r.RequestLine = *rl
			totalBytesParsed += n
			startIndex = totalBytesParsed
			r.parserState = PARSING_HEADERS
		case PARSING_HEADERS:
			n, done, err := r.Headers.Parse(data[startIndex:])
			if err != nil {
				return 0, err
			}

			if n == 0 {
				break outer
			}

			totalBytesParsed += n
			startIndex = totalBytesParsed

			if done {
				if r.Headers.Get("host") == "" {
					return 0, ERROR_MISSING_HOST_HEADER
				}
				r.parserState = PARSING_BODY
			}
		case PARSING_BODY:
			contentLen := r.Headers.Get("content-length")
			if contentLen == "" || contentLen == "0" {
				r.parserState = DONE
				break outer
			}

			specifiedBodyLen, err := strconv.Atoi(contentLen)
			if err != nil {
				return 0, err
			}

			oldLen := len(r.Body)

			r.Body = append(r.Body, data[startIndex:]...)
			newLen := len(r.Body)

			if newLen > specifiedBodyLen {
				return 0, ERROR_CONTENT_LENGTH_EXCEEDED
			}

			totalBytesParsed += newLen - oldLen

			if newLen == specifiedBodyLen {
				r.parserState = DONE
			} else {
				break outer
			}
		case DONE:
			break outer
		}
	}

	return totalBytesParsed, nil
}

func (r *Request) done() bool {
	return r.parserState == DONE
}

// HTTP-version = HTTP-name "/" DIGIT "." DIGIT
// HTTP-name = %s"HTTP"
// request-line = method SP request-target SP HTTP-version
func parseRequestLine(data []byte) (*RequestLine, int, error) {
	index := bytes.Index(data, CRLF)
	if index == -1 {
		return nil, 0, nil
	}

	line := string(data[:index])
	bytesParsed := index + len(CRLF)

	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return nil, 0, ERROR_MALFORMED_REQUEST_LINE
	}

	method := parts[0]
	if !isUpper(method) {
		return nil, 0, ERROR_INVALID_METHOD
	}

	httpParts := strings.Split(parts[2], "/")
	if len(httpParts) != 2 || httpParts[0] != "HTTP" || httpParts[1] != "1.1" {
		return nil, 0, ERROR_UNSUPPORTED_HTTP_VERSION
	}

	rl := &RequestLine{
		Method:        parts[0],
		RequestTarget: parts[1],
		HttpVersion:   httpParts[1],
	}

	return rl, bytesParsed, nil
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := newRequest()

	buf := make([]byte, BUFFER_SIZE)
	bufLen := 0

	for !request.done() {
		if bufLen >= len(buf) {
			newBuf := make([]byte, len(buf)+BUFFER_SIZE)
			copy(newBuf, buf)
			buf = newBuf
		}

		n, err := reader.Read(buf[bufLen:])
		if err != nil {
			return nil, err
		}

		bufLen += n

		parsedN, err := request.parse(buf[:bufLen])
		if err != nil {
			return nil, err
		}

		copy(buf, buf[parsedN:bufLen])
		bufLen -= parsedN
	}

	return request, nil
}

func isUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) {
			return false
		}
	}

	return true
}
